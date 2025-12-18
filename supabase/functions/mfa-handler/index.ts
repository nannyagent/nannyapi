import { serve } from "https://deno.land/std@0.168.0/http/server.ts";
import { createClient } from "https://esm.sh/@supabase/supabase-js@2.38.4";

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type"
};

// Request logging helper - AUDIT TRAIL
function logRequest(action: string, userId: string, status: string, details?: any) {
  console.log(JSON.stringify({
    timestamp: new Date().toISOString(),
    service: 'mfa-handler',
    action,
    userId: userId.substring(0, 8) + '...', // Log partial for privacy
    status,
    ...details
  }));
}

const sendError = (message: string, status = 500, action?: string, userId?: string) => {
  if (action && userId) {
    logRequest(action, userId, 'error', { message, status });
  }
  return new Response(JSON.stringify({ error: message }), {
    status,
    headers: {
      ...corsHeaders,
      "Content-Type": "application/json"
    }
  });
};

const sendSuccess = (data: any, status = 200, action?: string, userId?: string) => {
  if (action && userId) {
    logRequest(action, userId, 'success', { status });
  }
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      ...corsHeaders,
      "Content-Type": "application/json"
    }
  });
};

// Cache for config values to reduce DB queries (5 minute TTL)
const configCache = new Map<string, { value: string; expiry: number }>();
const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

// Fetch config value from database or cache
async function getConfigValue(
  serviceClient: any,
  key: string,
  defaultValue: string
): Promise<string> {
  // Check cache first
  const cached = configCache.get(key);
  if (cached && cached.expiry > Date.now()) {
    return cached.value;
  }

  try {
    const { data, error } = await serviceClient
      .from("system_config")
      .select("value")
      .eq("key", key)
      .maybeSingle();

    if (error || !data) {
      return defaultValue;
    }

    // Cache the value
    configCache.set(key, {
      value: data.value,
      expiry: Date.now() + CACHE_TTL_MS
    });

    return data.value;
  } catch (e) {
    console.error(`Error fetching config ${key}:`, e);
    return defaultValue;
  }
}

// Base32 decode
function base32Decode(str: string): Uint8Array {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let bits = "";
  for (const char of str) {
    const val = alphabet.indexOf(char);
    if (val === -1) throw new Error("Invalid character in base32");
    bits += val.toString(2).padStart(5, "0");
  }
  const result = [];
  for (let i = 0; i + 8 <= bits.length; i += 8) {
    result.push(parseInt(bits.substring(i, i + 8), 2));
  }
  return new Uint8Array(result);
}

// TOTP implementation with format validation
async function verifyTotp(token: string, secret: string, window = 1): Promise<boolean> {
  try {
    // SECURITY: Validate TOTP code format (must be 6 digits)
    if (!/^\d{6}$/.test(token)) {
      return false;
    }

    const secretBytes = base32Decode(secret.toUpperCase());
    const now = Math.floor(Date.now() / 1000);
    // Check current time window and adjacent windows
    for (let offset = -window; offset <= window; offset++) {
      const epoch = Math.floor((now + offset * 30) / 30);
      const counter = new ArrayBuffer(8);
      const view = new DataView(counter);
      view.setBigUint64(0, BigInt(epoch), false);
      const hmacKey = await crypto.subtle.importKey("raw", secretBytes, {
        name: "HMAC",
        hash: "SHA-1"
      }, false, ["sign"]);
      const hmacResult = await crypto.subtle.sign("HMAC", hmacKey, counter);
      const hmacBytes = new Uint8Array(hmacResult);
      const offset_num = hmacBytes[hmacBytes.length - 1] & 0x0f;
      const p = ((hmacBytes[offset_num] & 0x7f) << 24) |
                ((hmacBytes[offset_num + 1] & 0xff) << 16) |
                ((hmacBytes[offset_num + 2] & 0xff) << 8) |
                (hmacBytes[offset_num + 3] & 0xff);
      const code = (p % 1000000).toString().padStart(6, "0");
      if (code === token) {
        return true;
      }
    }
    return false;
  } catch (e) {
    return false;
  }
}

// Validation helpers - SECURITY
function validateAction(action: string): { valid: boolean; message?: string } {
  const validActions = ['setup', 'confirm', 'verify-totp', 'verify-backup-code', 'disable', 'check-backup-codes'];
  if (!validActions.includes(action)) {
    return { valid: false, message: `Invalid action. Must be one of: ${validActions.join(', ')}` };
  }
  return { valid: true };
}

function validateTotpCode(code: string): { valid: boolean; message?: string } {
  if (!code) {
    return { valid: false, message: 'TOTP code is required' };
  }
  if (!/^\d{6}$/.test(code)) {
    return { valid: false, message: 'TOTP code must be 6 digits' };
  }
  return { valid: true };
}

function validateBackupCode(code: string): { valid: boolean; message?: string } {
  if (!code) {
    return { valid: false, message: 'Backup code is required' };
  }
  if (code.length !== 8) {
    return { valid: false, message: 'Backup code must be 8 characters' };
  }
  return { valid: true };
}

// Improved backup code generation - STRONGER ENTROPY
function generateBackupCode(): string {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let code = "";
  // Use 5 bytes (40 bits) for better entropy - generates base32 with 8 chars
  const bytes = new Uint8Array(5);
  crypto.getRandomValues(bytes);
  let bits = 0;
  let bitsCount = 0;
  for (let i = 0; i < bytes.length; i++) {
    bits = (bits << 8) | bytes[i];
    bitsCount += 8;
    while (bitsCount >= 5) {
      bitsCount -= 5;
      code += alphabet[(bits >> bitsCount) & 0x1f];
    }
  }
  if (bitsCount > 0) {
    code += alphabet[((bits << (5 - bitsCount)) & 0x1f)];
  }
  
  // Ensure exactly 8 characters (trim if needed, should be rare)
  const result = code.substring(0, 8);
  if (result.length < 8) {
    // Fallback: shouldn't happen, but generate another byte if needed
    const extra = new Uint8Array(1);
    crypto.getRandomValues(extra);
    return (result + alphabet[extra[0] % 32]).substring(0, 8);
  }
  return result;
}

// Check if user is MFA locked (too many failed attempts)
async function checkMFALockout(serviceClient: any, userId: string): Promise<{ locked: boolean; lockedUntil?: string }> {
  try {
    const { data: lockout } = await serviceClient
      .from("mfa_lockout")
      .select("locked_until")
      .eq("user_id", userId)
      .gt("locked_until", new Date().toISOString())
      .maybeSingle();

    if (lockout) {
      return { locked: true, lockedUntil: lockout.locked_until };
    }
    return { locked: false };
  } catch (e) {
    console.error("Error checking MFA lockout:", e);
    return { locked: false };
  }
}

// Record failed MFA attempt and check if should lock user
async function recordMFAFailedAttempt(
  serviceClient: any,
  userId: string,
  action: string,
  ip: string,
  userAgent: string
): Promise<{ shouldLock: boolean; failCount: number }> {
  try {
    // Record the failed attempt
    await serviceClient
      .from("user_mfa_failed_attempts")
      .insert({
        user_id: userId,
        action,
        ip_address: ip,
        user_agent: userAgent
      });

    // Get MFA config
    const mfaFailLimitStr = await getConfigValue(serviceClient, "security.mfa_failed_attempts_limit", "5");
    const mfaCheckWindowStr = await getConfigValue(serviceClient, "security.mfa_check_window_hours", "24");
    
    const mfaFailLimit = parseInt(mfaFailLimitStr, 10);
    const mfaCheckWindowHours = parseInt(mfaCheckWindowStr, 10);

    // Count failed attempts in the window
    const { data: failedAttempts } = await serviceClient
      .from("user_mfa_failed_attempts")
      .select("id", { count: "exact" })
      .eq("user_id", userId)
      .eq("action", action)
      .gt("failed_at", new Date(Date.now() - mfaCheckWindowHours * 60 * 60 * 1000).toISOString());

    const failCount = failedAttempts?.length || 0;

    if (failCount >= mfaFailLimit) {
      // Lock MFA
      const mfaLockoutDurationStr = await getConfigValue(serviceClient, "security.mfa_lockout_duration_hours", "1");
      const mfaLockoutDurationHours = parseInt(mfaLockoutDurationStr, 10);
      const lockedUntil = new Date(Date.now() + mfaLockoutDurationHours * 60 * 60 * 1000).toISOString();

      await serviceClient
        .from("mfa_lockout")
        .insert({
          user_id: userId,
          locked_until: lockedUntil,
          reason: `Too many failed MFA ${action} attempts (${failCount}+ attempts)`,
          ip_address: ip,
          failed_attempts_count: failCount
        });

      return { shouldLock: true, failCount };
    }

    return { shouldLock: false, failCount };
  } catch (e) {
    console.error("Error recording MFA failed attempt:", e);
    return { shouldLock: false, failCount: 0 };
  }
}

// Hash code for secure storage
async function hashCode(code: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(code.toUpperCase().trim());
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
}

// SECURITY: Check if user is an agent (agents cannot manage user MFA)
function isAgent(user: any): boolean {
  return user?.app_metadata?.role === 'agent' || user?.user_metadata?.agent_type;
}
serve(async (req) => {
  if (req.method === "OPTIONS") {
    return new Response("ok", {
      headers: corsHeaders
    });
  }

  try {
    const authHeader = req.headers.get("authorization");
    if (!authHeader) return sendError("Unauthorized", 401);

    let body;
    try {
      const text = await req.text();
      if (!text) return sendError("Request body is empty", 400);
      body = JSON.parse(text);
    } catch (e) {
      return sendError("Invalid JSON", 400);
    }

    const { action } = body;
    if (!action) return sendError("Action is required", 400);

    // VALIDATE ACTION BEFORE PROCESSING
    const actionValidation = validateAction(action);
    if (!actionValidation.valid) {
      return sendError(actionValidation.message || "Invalid action", 400);
    }

    const supabaseUrl = Deno.env.get("SUPABASE_URL") || "";
    const anonKey = Deno.env.get("SUPABASE_ANON_KEY") || "";
    const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY") || "";

    if (!supabaseUrl || !anonKey || !serviceRoleKey) {
      return sendError("Missing config", 500);
    }

    const token = authHeader.replace("Bearer ", "");
    const anonClient = createClient(supabaseUrl, anonKey);
    const { data: { user }, error: userError } = await anonClient.auth.getUser(token);

    if (userError || !user) return sendError("Unauthorized", 401);

    // SECURITY: Agents cannot manage user MFA
    if (isAgent(user)) {
      logRequest(action, user.id, 'error', { message: 'Agent attempted to manage MFA' });
      return sendError("Agents cannot manage user MFA settings", 403);
    }

    // Get IP address for audit trail
    const xff = req.headers.get("x-forwarded-for");
    const ip = xff ? xff.split(",")[0].trim() : "0.0.0.0";

    const serviceClient = createClient(supabaseUrl, serviceRoleKey);

    // verify-totp
    if (action === "verify-totp") {
      const { code, secret } = body;
      
      // CHECK MFA LOCKOUT FIRST
      const mfaLockoutCheck = await checkMFALockout(serviceClient, user.id);
      if (mfaLockoutCheck.locked) {
        logRequest(action, user.id, 'mfa_locked', { lockedUntil: mfaLockoutCheck.lockedUntil, ip });
        return sendError(`MFA is locked due to too many failed attempts. Try again after ${mfaLockoutCheck.lockedUntil}.`, 429);
      }
      
      // VALIDATE TOTP CODE
      const codeValidation = validateTotpCode(code);
      if (!codeValidation.valid) {
        logRequest(action, user.id, 'error', { message: codeValidation.message, ip });
        return sendError(codeValidation.message || "Code required", 400);
      }

      let secretToUse = secret;
      if (!secretToUse) {
        const { data: mfaSettings, error } = await serviceClient
          .from("user_mfa_settings")
          .select("totp_secret")
          .eq("user_id", user.id)
          .eq("mfa_enabled", true)
          .maybeSingle();

        if (error || !mfaSettings) {
          logRequest(action, user.id, 'error', { message: 'MFA not enabled', ip });
          return sendError("MFA not enabled", 404);
        }
        secretToUse = mfaSettings.totp_secret;
      }

      const valid = await verifyTotp(code, secretToUse, 1);
      
      // If verification failed, record attempt and check if should lock
      if (!valid) {
        const userAgent = req.headers.get("user-agent") || "unknown";
        const { shouldLock, failCount } = await recordMFAFailedAttempt(
          serviceClient,
          user.id,
          action,
          ip,
          userAgent
        );

        if (shouldLock) {
          logRequest(action, user.id, 'mfa_lockout_triggered', { failCount, ip });
          return sendError(`Too many failed MFA attempts (${failCount}). MFA locked for 1 hour.`, 429);
        }

        logRequest(action, user.id, 'failed_validation', { attemptNumber: failCount, ip });
      } else {
        logRequest(action, user.id, 'success', { ip });
      }

      return sendSuccess({ valid }, 200, action, user.id);
    }

    // verify-backup-code - backup codes are now hashed
    if (action === "verify-backup-code") {
      const { code } = body;
      
      // CHECK MFA LOCKOUT FIRST
      const mfaLockoutCheck = await checkMFALockout(serviceClient, user.id);
      if (mfaLockoutCheck.locked) {
        logRequest(action, user.id, 'mfa_locked', { lockedUntil: mfaLockoutCheck.lockedUntil, ip });
        return sendError(`MFA is locked due to too many failed attempts. Try again after ${mfaLockoutCheck.lockedUntil}.`, 429);
      }

      if (!code) return sendError("Code required", 400);

      const { data: mfaSettings } = await serviceClient
        .from("user_mfa_settings")
        .select("backup_codes_hash")
        .eq("user_id", user.id)
        .eq("mfa_enabled", true)
        .maybeSingle();

      if (!mfaSettings || !mfaSettings.backup_codes_hash) {
        return sendError("MFA not enabled", 404);
      }

      const codeHash = await hashCode(code);
      
      // Check if code has already been used
      const { data: used } = await serviceClient
        .from("user_mfa_backup_codes_used")
        .select("id")
        .eq("user_id", user.id)
        .eq("code_hash", codeHash)
        .maybeSingle();

      if (used) {
        // Record failed attempt (reuse attempt)
        const userAgent = req.headers.get("user-agent") || "unknown";
        await recordMFAFailedAttempt(serviceClient, user.id, action, ip, userAgent);
        return sendError("Code already used", 400);
      }

      // Verify code exists in the hashed backup codes
      if (!mfaSettings.backup_codes_hash.includes(codeHash)) {
        // Record failed attempt
        const userAgent = req.headers.get("user-agent") || "unknown";
        const { shouldLock, failCount } = await recordMFAFailedAttempt(
          serviceClient,
          user.id,
          action,
          ip,
          userAgent
        );

        if (shouldLock) {
          logRequest(action, user.id, 'mfa_lockout_triggered', { failCount, ip });
          return sendError(`Too many failed MFA attempts (${failCount}). MFA locked for 1 hour.`, 429);
        }

        logRequest(action, user.id, 'failed_validation', { attemptNumber: failCount, ip });
        return sendError("Invalid backup code", 400);
      }

      // Record code as used
      await serviceClient
        .from("user_mfa_backup_codes_used")
        .insert({
          user_id: user.id,
          code_hash: codeHash,
          backup_code_index: mfaSettings.backup_codes_hash.indexOf(codeHash),
          ip_address: ip,
          user_agent: req.headers.get("user-agent") || "unknown",
          used_for_login: true
        });

      const { data: usedCodes } = await serviceClient
        .from("user_mfa_backup_codes_used")
        .select("code_hash")
        .eq("user_id", user.id);

      const remaining = mfaSettings.backup_codes_hash.length - (usedCodes?.length || 0);

      return sendSuccess({ valid: true, remaining });
    }

    // setup - Generate new TOTP secret and backup codes
    if (action === "setup") {
      const bytes = new Uint8Array(32);
      crypto.getRandomValues(bytes);
      const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
      let secret = "";
      for (let i = 0; i < bytes.length; i++) {
        secret += alphabet[bytes[i] % 32];
      }

      // Fetch configurable backup code count from database
      const backupCodesCountStr = await getConfigValue(
        serviceClient,
        "security.backup_codes_count",
        "8"
      );
      const backupCodesCount = parseInt(backupCodesCountStr, 10);

      // Generate backup codes (configurable count, 8 characters each)
      const codes = [];
      for (let i = 0; i < backupCodesCount; i++) {
        codes.push(generateBackupCode());
      }

      return sendSuccess({
        totp_secret: secret,
        backup_codes: codes,
        backupCodesCount: backupCodesCount
      });
    }

    // confirm - Enable MFA after TOTP verification
    if (action === "confirm") {
      const { totp_secret, totp_code, backup_codes } = body;

      if (!totp_secret) return sendError("Secret required", 400);
      if (!totp_code) return sendError("Code required", 400);
      if (!Array.isArray(backup_codes) || backup_codes.length === 0) {
        return sendError("Backup codes required", 400);
      }

      // Verify TOTP code format and validity
      const valid = await verifyTotp(totp_code, totp_secret, 1);
      if (!valid) return sendError("Invalid code", 401);

      // Hash all backup codes before storing (SECURITY)
      const backupCodesHash = await Promise.all(
        backup_codes.map(code => hashCode(code))
      );

      // Clear any previous used backup codes when enabling new MFA
      await serviceClient
        .from("user_mfa_backup_codes_used")
        .delete()
        .eq("user_id", user.id);

      const { data: existing } = await serviceClient
        .from("user_mfa_settings")
        .select("id")
        .eq("user_id", user.id)
        .maybeSingle();

      if (existing) {
        await serviceClient
          .from("user_mfa_settings")
          .update({
            totp_secret,
            backup_codes_hash: backupCodesHash,
            mfa_enabled: true,
            updated_at: new Date().toISOString()
          })
          .eq("user_id", user.id);
      } else {
        await serviceClient
          .from("user_mfa_settings")
          .insert({
            user_id: user.id,
            totp_secret,
            backup_codes_hash: backupCodesHash,
            mfa_enabled: true
          });
      }

      return sendSuccess({ success: true });
    }

    // disable - Disable MFA and clear all codes (SECURITY)
    if (action === "disable") {
      // Clear all MFA settings including backup codes
      await serviceClient
        .from("user_mfa_settings")
        .update({
          mfa_enabled: false,
          backup_codes_hash: null,
          totp_secret: null,
          updated_at: new Date().toISOString()
        })
        .eq("user_id", user.id);

      // Clear backup code usage history
      await serviceClient
        .from("user_mfa_backup_codes_used")
        .delete()
        .eq("user_id", user.id);

      return sendSuccess({ success: true });
    }

    return sendError("Unknown action", 400);
  } catch (error) {
    console.error("MFA Handler Error:", error);
    return sendError("Server error", 500);
  }
});
