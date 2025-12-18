import { serve } from "https://deno.land/std@0.168.0/http/server.ts";
import { createClient } from "https://esm.sh/@supabase/supabase-js@2.38.4";

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Headers": "authorization, x-client-info, apikey, content-type"
};

const SPECIAL_CHAR_REGEX = /[!@#$%^&*()_+\-={}[\];':"\\|,.<>/?]/;

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

// Password requirements validation
const validatePasswordRequirements = (password: string) => {
  const errors = [];
  const requirements = {
    minLength: password.length >= 8,
    hasUppercase: /[A-Z]/.test(password),
    hasLowercase: /[a-z]/.test(password),
    hasNumber: /[0-9]/.test(password),
    hasSpecialChar: SPECIAL_CHAR_REGEX.test(password)
  };

  if (!requirements.minLength) {
    errors.push("Password must be at least 8 characters long");
  }
  if (!requirements.hasUppercase) {
    errors.push("Password must contain at least one uppercase letter");
  }
  if (!requirements.hasLowercase) {
    errors.push("Password must contain at least one lowercase letter");
  }
  if (!requirements.hasNumber) {
    errors.push("Password must contain at least one number");
  }
  if (!requirements.hasSpecialChar) {
    errors.push("Password must contain at least one special character (!@#$%^&*)");
  }

  return {
    isValid: errors.length === 0,
    errors,
    requirements
  };
};

// Hash password for storage (to check reuse)
async function hashPassword(password: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(password);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
}

// Check if user is an agent (SECURITY: agents cannot change user passwords)
function isAgent(user: any): boolean {
  return user?.app_metadata?.role === 'agent' || user?.user_metadata?.agent_type;
}

serve(async (req) => {
  // Handle CORS preflight
  if (req.method === "OPTIONS") {
    return new Response("ok", {
      headers: corsHeaders
    });
  }

  try {
    // SECURITY: Require authentication
    const authHeader = req.headers.get("authorization");
    if (!authHeader) {
      return new Response(JSON.stringify({ error: "Unauthorized" }), {
        status: 401,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    const token = authHeader.replace("Bearer ", "");
    const supabaseUrl = Deno.env.get("SUPABASE_URL") || "";
    const anonKey = Deno.env.get("SUPABASE_ANON_KEY") || "";
    const serviceRoleKey = Deno.env.get("SUPABASE_SERVICE_ROLE_KEY") || "";

    if (!supabaseUrl || !anonKey || !serviceRoleKey) {
      return new Response(JSON.stringify({ error: "Missing config" }), {
        status: 500,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    const anonClient = createClient(supabaseUrl, anonKey);
    const { data: { user }, error: userError } = await anonClient.auth.getUser(token);

    if (userError || !user) {
      return new Response(JSON.stringify({ error: "Unauthorized" }), {
        status: 401,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    // SECURITY: Agents cannot change user passwords
    if (isAgent(user)) {
      return new Response(JSON.stringify({ 
        error: "Agents cannot change user passwords"
      }), {
        status: 403,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    const serviceClient = createClient(supabaseUrl, serviceRoleKey);
    
    let body;
    try {
      const text = await req.text();
      if (!text) {
        return new Response(JSON.stringify({
          error: "Request body is empty"
        }), {
          status: 400,
          headers: {
            ...corsHeaders,
            "Content-Type": "application/json"
          }
        });
      }
      body = JSON.parse(text);
    } catch (e) {
      return new Response(JSON.stringify({
        error: "Invalid JSON"
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    const { password } = body;

    if (!password) {
      return new Response(JSON.stringify({
        error: "Password is required"
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    let ip = "0.0.0.0";
    const xff = req.headers.get("x-forwarded-for");
    if (xff) ip = xff.split(",")[0].trim();

    // SECURITY: Fetch config from database
    const passwordChangeLimitStr = await getConfigValue(
      serviceClient,
      "security.password_change_limit_per_24h",
      "5"
    );
    const passwordChangeLimit = parseInt(passwordChangeLimitStr, 10);

    const passwordHistoryWindowStr = await getConfigValue(
      serviceClient,
      "security.password_history_window_hours",
      "24"
    );
    const passwordHistoryWindow = parseInt(passwordHistoryWindowStr, 10);

    const accountLockoutDurationStr = await getConfigValue(
      serviceClient,
      "security.account_lockout_duration_hours",
      "24"
    );
    const accountLockoutDuration = parseInt(accountLockoutDurationStr, 10);

    const failedLoginAttemptsLimitStr = await getConfigValue(
      serviceClient,
      "security.failed_login_attempts_limit",
      "10"
    );
    const failedLoginAttemptsLimit = parseInt(failedLoginAttemptsLimitStr, 10);

    // SECURITY: Check if user account is locked from too many failed login attempts
    const { data: lockout } = await serviceClient
      .from("account_lockout")
      .select("locked_until")
      .eq("user_id", user.id)
      .gt("locked_until", new Date().toISOString())
      .maybeSingle();

    if (lockout) {
      return new Response(JSON.stringify({
        error: "Account is locked due to too many failed login attempts. Try again later.",
        lockedUntil: lockout.locked_until
      }), {
        status: 429,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    // SECURITY: Check for too many failed login attempts - lock account after threshold
    // Account-based only, no IP-based checking (NAT-aware design)
    const { data: failedAttempts } = await serviceClient
      .from("password_change_attempts")
      .select("id", { count: "exact" })
      .eq("user_id", user.id)
      .eq("success", false) // Only count failed attempts
      .gt("attempted_at", new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString());

    const failedCount = failedAttempts?.length || 0;

    if (failedCount >= failedLoginAttemptsLimit) {
      // Lock account for configured duration
      const lockUntil = new Date(Date.now() + accountLockoutDuration * 60 * 60 * 1000).toISOString();
      
      await serviceClient
        .from("account_lockout")
        .insert({
          user_id: user.id,
          locked_until: lockUntil,
          reason: `Too many failed login attempts (${failedLoginAttemptsLimit}+ attempts)`,
          ip_address: ip
        });

      return new Response(JSON.stringify({
        error: `Too many failed login attempts. Account locked for ${accountLockoutDuration} hours.`,
        lockedUntil: lockUntil
      }), {
        status: 429,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    // SECURITY: Check password change frequency (configurable limit per 24h)
    const { data: recentChanges } = await serviceClient
      .from("password_change_history")
      .select("id", { count: "exact" })
      .eq("user_id", user.id)
      .gt("changed_at", new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString());

    if ((recentChanges?.length || 0) >= passwordChangeLimit) {
      // Lock account for configured duration
      const lockUntil = new Date(Date.now() + accountLockoutDuration * 60 * 60 * 1000).toISOString();
      
      await serviceClient
        .from("account_lockout")
        .insert({
          user_id: user.id,
          locked_until: lockUntil,
          reason: `Too many password changes (>${passwordChangeLimit} in 24h)`,
          ip_address: ip
        });

      return new Response(JSON.stringify({
        error: `Too many password changes. Account locked for ${accountLockoutDuration} hours.`,
        reason: `Maximum ${passwordChangeLimit} password changes allowed per 24 hours`,
        lockedUntil: lockUntil
      }), {
        status: 429,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    // SECURITY: Check password history - ensure not reused in last N hours
    const passwordHash = await hashPassword(password);
    const { data: recentHashes } = await serviceClient
      .from("password_change_history")
      .select("password_hash")
      .eq("user_id", user.id)
      .gt("changed_at", new Date(Date.now() - passwordHistoryWindow * 60 * 60 * 1000).toISOString());

    if (recentHashes?.some(h => h.password_hash === passwordHash)) {
      return new Response(JSON.stringify({
        error: `Password was recently used. Choose a different password. (Check history window: ${passwordHistoryWindow} hours)`
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          "Content-Type": "application/json"
        }
      });
    }

    // Validate password requirements
    const validationResult = validatePasswordRequirements(password);

    // Record attempt
    await serviceClient
      .from("password_change_attempts")
      .insert({
        user_id: user.id,
        ip_address: ip,
        success: validationResult.isValid
      });

    // If password is valid, add to history
    if (validationResult.isValid) {
      await serviceClient
        .from("password_change_history")
        .insert({
          user_id: user.id,
          password_hash: passwordHash,
          ip_address: ip,
          user_agent: req.headers.get("user-agent") || "unknown",
          changed_by_agent: false
        });
    }

    return new Response(JSON.stringify(validationResult), {
      status: validationResult.isValid ? 200 : 400,
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  } catch (error) {
    console.error("Error validating password:", error);
    return new Response(JSON.stringify({
      error: "Internal server error"
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        "Content-Type": "application/json"
      }
    });
  }
});
