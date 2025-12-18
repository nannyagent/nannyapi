import { createClient } from "npm:@supabase/supabase-js@2.28.0";
import { randomBytes } from "node:crypto";
import process from "node:process";
const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || process.env.SUPABASE_URL || '';
const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || process.env.SUPABASE_SERVICE_ROLE_KEY || '';
if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY) {
  console.error('Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY');
}
const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {
  auth: {
    persistSession: false
  }
});
const TTL_DEVICE_SESSION_SECONDS = 10 * 60; // 10 minutes (changed from 15)
const POLL_INTERVAL_SECONDS = 5;
// CORS headers for browser requests
const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'authorization, x-client-info, apikey, content-type',
  'Access-Control-Allow-Methods': 'POST, OPTIONS'
};
function generateRandomString(len = 48) {
  return randomBytes(len).toString('hex');
}
// Generate a 10-character alphanumeric user code for better entropy
function generateUserCode() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let result = '';
  const bytes = randomBytes(10);
  for(let i = 0; i < 10; i++){
    result += chars[bytes[i] % chars.length];
  }
  return result;
}
async function hashWithPepper(raw) {
  const encoder = new TextEncoder();
  const data = encoder.encode(raw + 'device-auth-pepper');
  const hashBuf = await crypto.subtle.digest('SHA-256', data);
  const arr = Array.from(new Uint8Array(hashBuf));
  return arr.map((b)=>b.toString(16).padStart(2, '0')).join('');
}
// Base64 URL decode function for Deno
function base64UrlDecode(str) {
  // Add padding if needed
  str += '='.repeat((4 - str.length % 4) % 4);
  // Replace URL-safe characters
  str = str.replace(/-/g, '+').replace(/_/g, '/');
  // Decode using atob (available in Deno)
  return atob(str);
}
// Extract hostname from client_id (format: "nannyagent-hostname")
function extractHostnameFromClientId(clientId) {
  if (!clientId || typeof clientId !== 'string') {
    return 'nannyagent';
  }
  // Remove "nannyagent-" prefix
  const prefix = 'nannyagent-';
  if (clientId.startsWith(prefix)) {
    const hostname = clientId.substring(prefix.length).trim();
    return hostname || 'nannyagent';
  }
  return clientId || 'nannyagent';
}
// Find unique agent name with deduplication (-1, -2, etc.)
async function findUniqueAgentName(baseHostname, userId) {
  // Sanitize hostname - remove special characters, max length 50
  const sanitized = baseHostname.replace(/[^a-zA-Z0-9-_]/g, '-').substring(0, 50);
  // Check if base name is available for this user
  const { data: existing, error } = await supabase.from('agents').select('name').eq('owner', userId).ilike('name', `${sanitized}%`).order('name', {
    ascending: true
  });
  if (error) {
    console.error('Error checking existing agent names:', error);
    return sanitized; // Fallback to base name
  }
  if (!existing || existing.length === 0) {
    // No conflicts, use base name
    return sanitized;
  }
  // Check if exact base name exists
  const exactMatch = existing.find((a)=>a.name === sanitized);
  if (!exactMatch) {
    return sanitized;
  }
  // Find next available suffix
  const suffixes = existing.map((a)=>{
    const match = a.name.match(new RegExp(`^${sanitized}-(\\d+)$`));
    return match ? parseInt(match[1]) : 0;
  }).filter((n)=>n > 0);
  const nextSuffix = suffixes.length > 0 ? Math.max(...suffixes) + 1 : 1;
  return `${sanitized}-${nextSuffix}`;
}
async function handleAuthorize(req) {
  const body = await req.json().catch(()=>({}));
  const client_id = body.client_id || 'unknown';
  const scope = body.scope || [
    'agent:register'
  ];
  const device_code = generateRandomString(24);
  const user_code = generateUserCode(); // Generate 10-character code
  const device_code_hash = await hashWithPepper(device_code);
  const now = new Date();
  const expires_at = new Date(now.getTime() + TTL_DEVICE_SESSION_SECONDS * 1000).toISOString();
  const insertData = {
    id: crypto.randomUUID(),
    device_id: device_code,
    requested_scopes: scope,
    device_code_hash: device_code_hash,
    user_code: user_code,
    client_metadata: {
      client_id
    },
    status: 'pending',
    created_at: now.toISOString(),
    expires_at: expires_at,
    interval_seconds: POLL_INTERVAL_SECONDS
  };
  const insert = await supabase.from('device_sessions').insert(insertData).select();
  if (insert.error) {
    return new Response(JSON.stringify({
      error: 'server_error',
      details: insert.error.message
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  const response = {
    device_code,
    user_code,
    verification_uri: `${SUPABASE_URL}/device/verify`,
    expires_in: TTL_DEVICE_SESSION_SECONDS,
    interval: POLL_INTERVAL_SECONDS
  };
  return new Response(JSON.stringify(response), {
    headers: {
      ...corsHeaders,
      'Content-Type': 'application/json'
    }
  });
}
async function extractUserIdFromJwt(jwt) {
  try {
    const parts = jwt.split('.');
    if (parts.length < 2) {
      return null;
    }
    const payload = JSON.parse(base64UrlDecode(parts[1]));
    const userId = payload.sub || payload.user_id || null;
    return userId;
  } catch (e) {
    return null;
  }
}

// Validate user code format (alphanumeric, 10 characters)
function validateUserCodeFormat(userCode: string): { valid: boolean; message?: string } {
  if (!userCode) {
    return { valid: false, message: 'Code is required' };
  }
  if (!/^[A-Z0-9]{10}$/.test(userCode)) {
    return { valid: false, message: 'Code must be 10 alphanumeric characters (A-Z, 0-9)' };
  }
  return { valid: true };
}

// Check rate limiting for failed attempts
async function checkRateLimitAndRecord(clientId: string, ipAddress: string, userCode: string, isFailed: boolean): Promise<{ allowed: boolean; message?: string; attemptCount?: number }> {
  // Get failed attempts in last 24 hours for this client
  const twentyFourHoursAgo = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
  
  const { data: failedAttempts, error } = await supabase
    .from('device_failed_attempts')
    .select('id', { count: 'exact' })
    .eq('client_id', clientId)
    .gt('attempted_at', twentyFourHoursAgo);

  if (error) {
    console.error('Error checking rate limit:', error);
  }

  const attemptCount = failedAttempts?.length || 0;
  const maxAttempts = 10;

  if (isFailed) {
    // Record the failed attempt
    const { error: insertError } = await supabase
      .from('device_failed_attempts')
      .insert({
        client_id: clientId,
        user_code: userCode,
        ip_address: ipAddress,
        attempted_at: new Date().toISOString()
      });

    if (insertError) {
      console.error('Error recording failed attempt:', insertError);
    }

    const newAttemptCount = attemptCount + 1;

    if (newAttemptCount >= maxAttempts) {
      return {
        allowed: false,
        message: `Rate limit exceeded. Maximum ${maxAttempts} failed attempts allowed in 24 hours.`,
        attemptCount: newAttemptCount
      };
    }

    return {
      allowed: true,
      attemptCount: newAttemptCount
    };
  }

  return {
    allowed: true,
    attemptCount
  };
}

// Check if code has already been consumed by another agent
async function checkCodeAlreadyConsumed(userCode: string): Promise<{ consumed: boolean; agentId?: string }> {
  const { data, error } = await supabase
    .from('device_code_consumption')
    .select('agent_id')
    .eq('user_code', userCode)
    .maybeSingle();

  if (error) {
    console.error('Error checking code consumption:', error);
    return { consumed: false };
  }

  if (data) {
    return { consumed: true, agentId: data.agent_id };
  }

  return { consumed: false };
}
async function handleApprove(req) {
  const auth = req.headers.get('authorization') || '';
  if (!auth.startsWith('Bearer ')) {
    return new Response(JSON.stringify({
      error: 'unauthorized',
      message: 'Bearer token required'
    }), {
      status: 401,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  const jwt = auth.split(' ')[1];
  const body = await req.json().catch(()=>({}));
  const user_code = body.user_code;
  if (!user_code) {
    return new Response(JSON.stringify({
      error: 'invalid_request',
      message: 'user_code is required'
    }), {
      status: 400,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }

  // Validate code format (alphanumeric, 10 characters)
  const formatValidation = validateUserCodeFormat(user_code);
  if (!formatValidation.valid) {
    return new Response(JSON.stringify({
      error: 'invalid_code_format',
      message: formatValidation.message
    }), {
      status: 400,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  const userId = await extractUserIdFromJwt(jwt);
  if (!userId) {
    return new Response(JSON.stringify({
      error: 'invalid_token',
      message: 'Invalid JWT token or missing user ID'
    }), {
      status: 401,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }

  // Check rate limiting and record failed attempt if needed
  // Get client_id from headers (used for rate limiting)
  const clientIdFromHeader = req.headers.get('x-client-id') || 'unknown-client';
  const ipAddress = req.headers.get('x-forwarded-for') || req.headers.get('x-real-ip') || 'unknown';
  
  // Find pending device session by user_code
  const { data: sessions, error } = await supabase.from('device_sessions').select('*').eq('user_code', user_code).eq('status', 'pending').limit(1);
  if (error) {
    // Record failed attempt
    await checkRateLimitAndRecord(clientIdFromHeader, ipAddress, user_code, true);
    
    return new Response(JSON.stringify({
      error: 'server_error',
      details: error.message
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  if (!sessions || sessions.length === 0) {
    // Record failed attempt
    await checkRateLimitAndRecord(clientIdFromHeader, ipAddress, user_code, true);
    
    return new Response(JSON.stringify({
      error: 'not_found',
      message: 'Device code not found or already used'
    }), {
      status: 404,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  const session = sessions[0];
  // Check if expired
  if (session.expires_at && new Date(session.expires_at) < new Date()) {
    return new Response(JSON.stringify({
      error: 'expired_token',
      message: 'Device code has expired'
    }), {
      status: 400,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }

  // Check rate limiting
  const rateLimitCheck = await checkRateLimitAndRecord(clientIdFromHeader, ipAddress, user_code, false);
  if (!rateLimitCheck.allowed) {
    return new Response(JSON.stringify({
      error: 'rate_limit_exceeded',
      message: rateLimitCheck.message,
      attempt_count: rateLimitCheck.attemptCount
    }), {
      status: 429,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }

  // Check if code has already been consumed by another agent
  const consumptionCheck = await checkCodeAlreadyConsumed(user_code);
  if (consumptionCheck.consumed) {
    return new Response(JSON.stringify({
      error: 'code_already_used',
      message: 'Device code has already been consumed by another agent',
      consumed_by_agent: consumptionCheck.agentId
    }), {
      status: 400,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  // *** FIXED: Extract hostname from client_id ***
  const clientId = session.client_metadata?.client_id || 'nannyagent';
  const baseHostname = extractHostnameFromClientId(clientId);
  // *** FIXED: Find unique agent name with deduplication ***
  const agentName = await findUniqueAgentName(baseHostname, userId);
  console.log('Creating agent with hostname-based name:', {
    client_id: clientId,
    base_hostname: baseHostname,
    final_agent_name: agentName,
    owner: userId
  });
  // *** KEY CHANGE: Create a Supabase user for the agent ***
  // Generate unique credentials for the agent
  const agentEmail = `agent-${session.id}@nannyagent.internal`;
  const agentPassword = generateRandomString(32);
  try {
    // Create agent user in Supabase Auth using Admin API
    const { data: agentUser, error: createError } = await supabase.auth.admin.createUser({
      email: agentEmail,
      password: agentPassword,
      email_confirm: true,
      user_metadata: {
        agent_type: 'nannyagent',
        authorized_by: userId,
        device_session_id: session.id,
        created_via: 'device_flow',
        hostname: baseHostname,
        agent_name: agentName // Store final agent name
      },
      app_metadata: {
        role: 'agent',
        authorizing_user: userId
      }
    });
    if (createError) {
      console.error('Failed to create agent user:', createError);
      return new Response(JSON.stringify({
        error: 'server_error',
        message: 'Failed to create agent user'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    // *** FIXED: Create agent record with hostname-based name and deduplication ***
    const agentRecord = {
      id: agentUser.user.id,
      owner: userId,
      name: agentName,
      status: 'active',
      created_at: new Date().toISOString(),
      metadata: {
        device_session_id: session.id,
        authorized_via: 'device_flow',
        agent_type: 'nannyagent',
        email: agentEmail,
        client_id: clientId,
        hostname: baseHostname,
        original_hostname: baseHostname
      }
    };
    const { error: agentInsertError } = await supabase.from('agents').insert(agentRecord);
    if (agentInsertError) {
      console.error('Failed to create agent record:', agentInsertError);
      // Clean up the auth user if agent record creation fails
      await supabase.auth.admin.deleteUser(agentUser.user.id);
      return new Response(JSON.stringify({
        error: 'server_error',
        message: 'Failed to create agent record'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    console.log('Created agent user and agent record:', {
      auth_user_id: agentUser.user.id,
      agent_record_id: agentRecord.id,
      agent_name: agentName,
      hostname: baseHostname,
      owner: userId
    });
    // Update device session with approval and agent info
    const updateData = {
      authorized: true,
      authorized_at: new Date().toISOString(),
      authorized_by: userId,
      approved_by: userId,
      approved_at: new Date().toISOString(),
      status: 'approved',
      agent_user_id: agentUser.user.id,
      agent_email: agentEmail,
      agent_password: agentPassword,
      stored_agent_id: agentUser.user.id // Store the agent ID for reference
    };
    const update = await supabase.from('device_sessions').update(updateData).eq('device_code_hash', session.device_code_hash);
    if (update.error) {
      // Clean up if device session update fails
      await supabase.auth.admin.deleteUser(agentUser.user.id);
      await supabase.from('agents').delete().eq('id', agentUser.user.id);
      return new Response(JSON.stringify({
        error: 'server_error',
        details: update.error.message
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }

    // Track code consumption to prevent reuse by other agents
    const { error: consumptionError } = await supabase
      .from('device_code_consumption')
      .insert({
        user_code: user_code,
        agent_id: agentUser.user.id,
        consumed_at: new Date().toISOString()
      });

    if (consumptionError) {
      console.error('Error recording code consumption:', consumptionError);
      // Don't fail the whole operation, but log it
    }

    return new Response(JSON.stringify({
      success: true,
      message: 'Device approved successfully',
      agent_id: agentUser.user.id,
      agent_name: agentName,
      hostname: baseHostname
    }), {
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Error creating agent user:', error);
    return new Response(JSON.stringify({
      error: 'server_error',
      message: 'Failed to create agent user'
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
}
// Handle token refresh using Supabase's refreshSession API
async function handleRefreshToken(refreshToken) {
  try {
    console.log('Refreshing token using Supabase Auth API');
    // Use Supabase's refreshSession method
    const { data, error } = await supabase.auth.refreshSession({
      refresh_token: refreshToken
    });
    if (error || !data.session) {
      console.error('Token refresh failed:', error);
      return null;
    }
    console.log('Token refreshed successfully for user:', data.user?.id);
    return {
      access_token: data.session.access_token,
      token_type: 'bearer',
      expires_in: data.session.expires_in || 3600,
      refresh_token: data.session.refresh_token,
      agent_id: data.user.id
    };
  } catch (error) {
    console.error('Error refreshing token:', error);
    return null;
  }
}
async function handleToken(req) {
  const contentType = req.headers.get('content-type') || '';
  let grant_type, device_code, refresh_token;
  // Handle both form data and JSON requests
  if (contentType.includes('application/x-www-form-urlencoded')) {
    const formText = await req.text().catch(()=>'');
    const params = new URLSearchParams(formText);
    grant_type = params.get('grant_type');
    device_code = params.get('device_code');
    refresh_token = params.get('refresh_token');
  } else {
    // Handle JSON requests
    const body = await req.json().catch(()=>({}));
    grant_type = body.grant_type;
    device_code = body.device_code;
    refresh_token = body.refresh_token;
  }
  // Handle refresh token requests
  if (grant_type === 'refresh_token') {
    if (!refresh_token) {
      return new Response(JSON.stringify({
        error: 'invalid_request',
        message: 'refresh_token is required'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const refreshResult = await handleRefreshToken(refresh_token);
    if (!refreshResult) {
      return new Response(JSON.stringify({
        error: 'invalid_grant',
        message: 'Invalid or expired refresh token'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    return new Response(JSON.stringify(refreshResult), {
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  // Handle device code requests (existing functionality)
  if (grant_type === 'urn:ietf:params:oauth:grant-type:device_code') {
    if (!device_code) {
      return new Response(JSON.stringify({
        error: 'invalid_request',
        message: 'device_code is required'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const device_code_hash = await hashWithPepper(String(device_code));
    const { data: sessions, error } = await supabase.from('device_sessions').select('*').eq('device_code_hash', device_code_hash).limit(1);
    if (error) {
      return new Response(JSON.stringify({
        error: 'server_error',
        details: error.message
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    if (!sessions || sessions.length === 0) {
      return new Response(JSON.stringify({
        error: 'invalid_grant',
        message: 'Invalid device code'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const session = sessions[0];
    if (session.expires_at && new Date(session.expires_at) < new Date()) {
      return new Response(JSON.stringify({
        error: 'expired_token',
        message: 'Device code has expired'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    if (!session.authorized && !session.approved_by) {
      return new Response(JSON.stringify({
        error: 'authorization_pending',
        message: 'User has not approved the device yet'
      }), {
        status: 428,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    // *** KEY CHANGE: Sign in the agent user to get real Supabase JWT tokens ***
    if (!session.agent_email || !session.agent_password) {
      return new Response(JSON.stringify({
        error: 'server_error',
        message: 'Agent credentials not available'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    try {
      // Sign in as the agent user to get proper Supabase JWT tokens
      const { data: authData, error: signInError } = await supabase.auth.signInWithPassword({
        email: session.agent_email,
        password: session.agent_password
      });
      if (signInError || !authData.session) {
        console.error('Failed to sign in agent:', signInError);
        return new Response(JSON.stringify({
          error: 'server_error',
          message: 'Failed to generate agent tokens'
        }), {
          status: 500,
          headers: {
            ...corsHeaders,
            'Content-Type': 'application/json'
          }
        });
      }
      // Clear the stored password for security
      await supabase.from('device_sessions').update({
        agent_password: null
      }).eq('device_code_hash', session.device_code_hash);
      // Return real Supabase JWT tokens!
      const tokenResponse = {
        access_token: authData.session.access_token,
        token_type: 'bearer',
        expires_in: authData.session.expires_in || 3600,
        refresh_token: authData.session.refresh_token,
        scope: session.requested_scopes || [
          'agent:register'
        ],
        agent_id: authData.user.id,
        user_id: authData.user.id // For compatibility
      };
      return new Response(JSON.stringify(tokenResponse), {
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    } catch (error) {
      console.error('Error signing in agent:', error);
      return new Response(JSON.stringify({
        error: 'server_error',
        message: 'Failed to generate agent tokens'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
  }
  return new Response(JSON.stringify({
    error: 'unsupported_grant_type',
    message: 'Supported grant types: urn:ietf:params:oauth:grant-type:device_code, refresh_token'
  }), {
    status: 400,
    headers: {
      ...corsHeaders,
      'Content-Type': 'application/json'
    }
  });
}
Deno.serve(async (req)=>{
  // Handle CORS preflight requests
  if (req.method === 'OPTIONS') {
    return new Response('ok', {
      headers: corsHeaders
    });
  }
  const url = new URL(req.url);
  if (req.method === 'POST' && url.pathname.endsWith('/device/authorize')) {
    return handleAuthorize(req);
  }
  if (req.method === 'POST' && url.pathname.endsWith('/device/approve')) {
    return handleApprove(req);
  }
  if (req.method === 'POST' && url.pathname.endsWith('/token')) {
    return handleToken(req);
  }
  return new Response(JSON.stringify({
    error: 'not_found',
    message: 'Endpoint not found'
  }), {
    status: 404,
    headers: {
      ...corsHeaders,
      'Content-Type': 'application/json'
    }
  });
});
