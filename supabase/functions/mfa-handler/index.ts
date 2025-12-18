import { serve } from "https://deno.land/std@0.168.0/http/server.ts";
import { createClient } from "https://esm.sh/@supabase/supabase-js@2.38.4";

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type"
};

const sendError = (message: string, status = 500) => {
  return new Response(JSON.stringify({ error: message }), {
    status,
    headers: {
      ...corsHeaders,
      "Content-Type": "application/json"
    }
  });
};

const sendSuccess = (data: any, status = 200) => {
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

// Generate backup code (8 characters exactly as per requirements)
function generateBackupCode(): string {
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
  let code = "";
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
  return code.substring(0, 8); // Ensure exactly 8 characters
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
      return sendError("Agents cannot manage user MFA settings", 403);
    }

    const serviceClient = createClient(supabaseUrl, serviceRoleKey);

    // verify-totp
    if (action === "verify-totp") {
      const { code, secret } = body;
      if (!code) return sendError("Code required", 400);

      let secretToUse = secret;
      if (!secretToUse) {
        const { data: mfaSettings, error } = await serviceClient
          .from("user_mfa_settings")
          .select("totp_secret")
          .eq("user_id", user.id)
          .eq("mfa_enabled", true)
          .maybeSingle();

        if (error || !mfaSettings) return sendError("MFA not enabled", 404);
        secretToUse = mfaSettings.totp_secret;
      }

      const valid = await verifyTotp(code, secretToUse, 1);
      return sendSuccess({ valid });
    }

    // verify-backup-code - backup codes are now hashed
    if (action === "verify-backup-code") {
      const { code } = body;
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

      if (used) return sendError("Code already used", 400);

      // Verify code exists in the hashed backup codes
      if (!mfaSettings.backup_codes_hash.includes(codeHash)) {
        return sendError("Invalid backup code", 400);
      }

      let ip = "0.0.0.0";
      const xff = req.headers.get("x-forwarded-for");
      if (xff) ip = xff.split(",")[0].trim();

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
