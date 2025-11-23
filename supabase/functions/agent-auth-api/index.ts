import { createClient } from "npm:@supabase/supabase-js@2.32.0";import { createClient } from "npm:@supabase/supabase-js@2.32.0";

const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';

const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || '';const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || '';

const SUPABASE_ANON_KEY = Deno.env.get('SUPABASE_ANON_KEY') || '';const SUPABASE_ANON_KEY = Deno.env.get('SUPABASE_ANON_KEY') || '';

if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY || !SUPABASE_ANON_KEY) {if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY || !SUPABASE_ANON_KEY) {

  console.error('Missing required environment variables');  console.error('Missing required environment variables');

}}

// Service role client for admin operations// Service role client for admin operations

const supabaseAdmin = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {const supabaseAdmin = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {

  auth: {  auth: {

    persistSession: false    persistSession: false

  }  }

});});

// Regular client for auth validation// Regular client for auth validation

const supabase = createClient(SUPABASE_URL, SUPABASE_ANON_KEY, {const supabase = createClient(SUPABASE_URL, SUPABASE_ANON_KEY, {

  auth: {  auth: {

    persistSession: false    persistSession: false

  }  }

});});

// Middleware to validate JWT token from Supabase auth (for regular users)// Middleware to validate JWT token from Supabase auth (for regular users)

async function validateSupabaseAuth(req) {async function validateSupabaseAuth(req) {

  const authHeader = req.headers.get('Authorization');  const authHeader = req.headers.get('Authorization');

  if (!authHeader || !authHeader.startsWith('Bearer ')) {  if (!authHeader || !authHeader.startsWith('Bearer ')) {

    return {    return {

      user: null,      user: null,

      error: 'Missing or invalid authorization header'      error: 'Missing or invalid authorization header'

    };    };

  }  }

  const token = authHeader.substring(7);  const token = authHeader.substring(7);

  try {  try {

    const { data: { user }, error } = await supabase.auth.getUser(token);    const { data: { user }, error } = await supabase.auth.getUser(token);

    if (error || !user) {    if (error || !user) {

      return {      return {

        user: null,        user: null,

        error: 'Invalid token'        error: 'Invalid token'

      };      };

    }    }

    return {    return {

      user      user

    };    };

  } catch (error) {  } catch (error) {

    return {    return {

      user: null,      user: null,

      error: 'Token validation failed'      error: 'Token validation failed'

    };    };

  }  }

}}

// Middleware to validate agent token (now also Supabase JWT tokens)// Middleware to validate agent token (now also Supabase JWT tokens)

async function validateAgentToken(req) {async function validateAgentToken(req) {

  const authHeader = req.headers.get('Authorization');  const authHeader = req.headers.get('Authorization');

  if (!authHeader || !authHeader.startsWith('Bearer ')) {  if (!authHeader || !authHeader.startsWith('Bearer ')) {

    return {    return {

      agent: null,      agent: null,

      error: 'Missing or invalid authorization header'      error: 'Missing or invalid authorization header'

    };    };

  }  }

  const token = authHeader.substring(7);  const token = authHeader.substring(7);

  try {  try {

    // Try to validate as Supabase JWT token first (new system)    // Try to validate as Supabase JWT token first (new system)

    const { data: { user }, error } = await supabase.auth.getUser(token);    const { data: { user }, error } = await supabase.auth.getUser(token);

    if (!error && user) {    if (!error && user) {

      // This is a valid Supabase JWT token      // This is a valid Supabase JWT token

      // Check if this user is an agent (has agent metadata)      // Check if this user is an agent (has agent metadata)

      if (user.app_metadata?.role === 'agent' || user.user_metadata?.agent_type) {      if (user.app_metadata?.role === 'agent' || user.user_metadata?.agent_type) {

        console.log('Agent authenticated via Supabase JWT:', {        console.log('Agent authenticated via Supabase JWT:', {

          id: user.id,          id: user.id,

          email: user.email,          email: user.email,

          role: user.app_metadata?.role,          role: user.app_metadata?.role,

          agent_type: user.user_metadata?.agent_type          agent_type: user.user_metadata?.agent_type

        });        });

        return {        return {

          agent: {          agent: {

            agent_id: user.id,            agent_id: user.id,

            user_id: user.user_metadata?.authorized_by || user.id,            user_id: user.user_metadata?.authorized_by || user.id,

            email: user.email,            email: user.email,

            agent_type: user.user_metadata?.agent_type || 'nannyagent',            agent_type: user.user_metadata?.agent_type || 'nannyagent',

            device_session_id: user.user_metadata?.device_session_id,            device_session_id: user.user_metadata?.device_session_id,

            created_at: user.created_at,            created_at: user.created_at,

            agent_registrations: {            agent_registrations: {

              agent_name: user.user_metadata?.agent_type || 'NannyAgent',              agent_name: user.user_metadata?.agent_type || 'NannyAgent',

              agent_type: user.user_metadata?.agent_type || 'nannyagent'              agent_type: user.user_metadata?.agent_type || 'nannyagent'

            }            }

          }          }

        };        };

      }      }

    }    }

    // If Supabase JWT validation failed or user is not an agent,     // If Supabase JWT validation failed or user is not an agent, 

    // fall back to old HMAC system for backward compatibility    // fall back to old HMAC system for backward compatibility

    console.log('Supabase JWT validation failed, trying legacy HMAC system...');    console.log('Supabase JWT validation failed, trying legacy HMAC system...');

    // Legacy HMAC validation (keeping for backward compatibility)    // Legacy HMAC validation (keeping for backward compatibility)

    const AGENT_TOKEN_HMAC_SECRET = Deno.env.get('AGENT_TOKEN_HMAC_SECRET') || '';    const AGENT_TOKEN_HMAC_SECRET = Deno.env.get('AGENT_TOKEN_HMAC_SECRET') || '';

    if (!AGENT_TOKEN_HMAC_SECRET) {    if (!AGENT_TOKEN_HMAC_SECRET) {

      return {      return {

        agent: null,        agent: null,

        error: 'Token validation system not properly configured'        error: 'Token validation system not properly configured'

      };      };

    }    }

    const { createHmac } = await import('node:crypto');    const { createHmac } = await import('node:crypto');

    const tokenHash = createHmac('sha256', AGENT_TOKEN_HMAC_SECRET).update(token).digest('hex');    const tokenHash = createHmac('sha256', AGENT_TOKEN_HMAC_SECRET).update(token).digest('hex');

    const { data: agentToken, error: tokenError } = await supabaseAdmin.from('agent_tokens').select('*, agent_registrations(*)').eq('token_hash', tokenHash).eq('revoked', false).single();    const { data: agentToken, error: tokenError } = await supabaseAdmin.from('agent_tokens').select('*, agent_registrations(*)').eq('token_hash', tokenHash).eq('revoked', false).single();

    if (tokenError || !agentToken) {    if (tokenError || !agentToken) {

      return {      return {

        agent: null,        agent: null,

        error: 'Invalid agent token'        error: 'Invalid agent token'

      };      };

    }    }

    if (new Date(agentToken.expires_at) < new Date()) {    if (new Date(agentToken.expires_at) < new Date()) {

      return {      return {

        agent: null,        agent: null,

        error: 'Token expired'        error: 'Token expired'

      };      };

    }    }

    console.log('Agent authenticated via legacy HMAC system');    console.log('Agent authenticated via legacy HMAC system');

    return {    return {

      agent: agentToken      agent: agentToken

    };    };

  } catch (error) {  } catch (error) {

    console.error('Agent token validation error:', error);    console.error('Agent token validation error:', error);

    return {    return {

      agent: null,      agent: null,

      error: 'Agent token validation failed'      error: 'Agent token validation failed'

    };    };

  }  }

}}

// POST /agent/register - Requires Supabase JWT auth (deprecated - use device-auth flow)// POST /agent/register - Requires Supabase JWT auth (deprecated - use device-auth flow)

async function handleAgentRegister(req) {async function handleAgentRegister(req) {

  try {  try {

    const { user, error: authError } = await validateSupabaseAuth(req);    const { user, error: authError } = await validateSupabaseAuth(req);

    if (authError) {    if (authError) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: authError        error: authError

      }), {      }), {

        status: 401,        status: 401,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    const body = await req.json().catch(()=>({}));    const body = await req.json().catch(()=>({}));

    const { device_code, agent_name, agent_type, metadata } = body;    const { device_code, agent_name, agent_type, metadata } = body;

    if (!device_code) {    if (!device_code) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'device_code is required. Use the new device-auth flow instead.'        error: 'device_code is required. Use the new device-auth flow instead.'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // This endpoint is deprecated - redirect to device-auth flow    // This endpoint is deprecated - redirect to device-auth flow

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      error: 'This endpoint is deprecated. Please use the device-auth flow at /device-auth/device/authorize'      error: 'This endpoint is deprecated. Please use the device-auth flow at /device-auth/device/authorize'

    }), {    }), {

      status: 410,      status: 410,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleAgentRegister:', error);    console.error('Error in handleAgentRegister:', error);

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      error: 'Internal server error'      error: 'Internal server error'

    }), {    }), {

      status: 500,      status: 500,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  }  }

}}

// Helper function to safely convert metrics values// Helper function to safely convert metrics values

function safeMetricValue(value) {function safeMetricValue(value) {

  if (typeof value === 'number' && !isNaN(value)) {  if (typeof value === 'number' && !isNaN(value)) {

    return value;    return value;

  }  }

  return null;  return null;

}}

// Helper function to safely convert string values// Helper function to safely convert string values

function safeStringValue(value) {function safeStringValue(value) {

  if (typeof value === 'string' && value.trim() !== '') {  if (typeof value === 'string' && value.trim() !== '') {

    return value.trim();    return value.trim();

  }  }

  return null;  return null;

}}

// POST /agent/metrics - Uses existing database schema with existing columns (FIXED block_devices)// POST /agent/metrics - Uses existing database schema with existing columns (FIXED block_devices)

async function handleAgentMetrics(req) {async function handleAgentMetrics(req) {

  try {  try {

    const { agent, error: authError } = await validateAgentToken(req);    const { agent, error: authError } = await validateAgentToken(req);

    if (authError) {    if (authError) {

      console.error('Agent auth error:', authError);      console.error('Agent auth error:', authError);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: authError        error: authError

      }), {      }), {

        status: 401,        status: 401,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    const body = await req.json().catch(()=>({}));    const body = await req.json().catch(()=>({}));

    const { metrics, timestamp, agent_id } = body;    const { metrics, timestamp, agent_id } = body;

    if (!metrics || typeof metrics !== 'object') {    if (!metrics || typeof metrics !== 'object') {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'metrics object is required'        error: 'metrics object is required'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Use agent_id from request body if provided, otherwise use from token    // Use agent_id from request body if provided, otherwise use from token

    const finalAgentId = agent_id || agent.agent_id;    const finalAgentId = agent_id || agent.agent_id;

    console.log('Processing metrics for agent:', finalAgentId);    console.log('Processing metrics for agent:', finalAgentId);

    console.log('Metrics payload:', JSON.stringify(metrics, null, 2));    console.log('Metrics payload:', JSON.stringify(metrics, null, 2));

    // Build record using EXISTING schema columns only    // Build record using EXISTING schema columns only

    const metricsRecord = {    const metricsRecord = {

      agent_id: finalAgentId,      agent_id: finalAgentId,

      recorded_at: timestamp || new Date().toISOString(),      recorded_at: timestamp || new Date().toISOString(),

      // Existing individual columns (keep these for now)      // Existing individual columns (keep these for now)

      cpu_percent: safeMetricValue(metrics.cpu_usage) || safeMetricValue(metrics.cpu_percent),      cpu_percent: safeMetricValue(metrics.cpu_usage) || safeMetricValue(metrics.cpu_percent),

      memory_mb: metrics.memory_usage != null ? Math.round(Number(metrics.memory_usage)) : null,      memory_mb: metrics.memory_usage != null ? Math.round(Number(metrics.memory_usage)) : null,

      disk_percent: safeMetricValue(metrics.disk_usage) || safeMetricValue(metrics.disk_percent),      disk_percent: safeMetricValue(metrics.disk_usage) || safeMetricValue(metrics.disk_percent),

      network_in_kbps: safeMetricValue(metrics.network_in_kbps),      network_in_kbps: safeMetricValue(metrics.network_in_kbps),

      network_out_kbps: safeMetricValue(metrics.network_out_kbps),      network_out_kbps: safeMetricValue(metrics.network_out_kbps),

      ip_address: safeStringValue(metrics.ip_address),      ip_address: safeStringValue(metrics.ip_address),

      location: safeStringValue(metrics.location),      location: safeStringValue(metrics.location),

      agent_version: safeStringValue(metrics.agent_version),      agent_version: safeStringValue(metrics.agent_version),

      kernel_version: safeStringValue(metrics.kernel_version),      kernel_version: safeStringValue(metrics.kernel_version),

      device_fingerprint: safeStringValue(metrics.device_fingerprint),      device_fingerprint: safeStringValue(metrics.device_fingerprint),

      // JSON columns that DO exist in schema      // JSON columns that DO exist in schema

      load_averages: metrics.load_averages || {      load_averages: metrics.load_averages || {

        load1: safeMetricValue(metrics.load_avg) || safeMetricValue(metrics.load1),        load1: safeMetricValue(metrics.load_avg) || safeMetricValue(metrics.load1),

        load5: safeMetricValue(metrics.load5),        load5: safeMetricValue(metrics.load5),

        load15: safeMetricValue(metrics.load15)        load15: safeMetricValue(metrics.load15)

      },      },

      os_info: metrics.os_info || {      os_info: metrics.os_info || {

        name: safeStringValue(metrics.os_name),        name: safeStringValue(metrics.os_name),

        version: safeStringValue(metrics.os_version),        version: safeStringValue(metrics.os_version),

        architecture: safeStringValue(metrics.os_architecture),        architecture: safeStringValue(metrics.os_architecture),

        platform: safeStringValue(metrics.os_platform),        platform: safeStringValue(metrics.os_platform),

        family: safeStringValue(metrics.os_family),        family: safeStringValue(metrics.os_family),

        kernel_version: safeStringValue(metrics.kernel_version)        kernel_version: safeStringValue(metrics.kernel_version)

      },      },

      filesystem_info: metrics.filesystem_info || [],      filesystem_info: metrics.filesystem_info || [],

      // FIXED: Added block_devices column that was missing      // FIXED: Added block_devices column that was missing

      block_devices: metrics.block_devices || [],      block_devices: metrics.block_devices || [],

      network_stats: metrics.network_stats || {      network_stats: metrics.network_stats || {

        network_in_kbps: safeMetricValue(metrics.network_in_kbps),        network_in_kbps: safeMetricValue(metrics.network_in_kbps),

        network_out_kbps: safeMetricValue(metrics.network_out_kbps),        network_out_kbps: safeMetricValue(metrics.network_out_kbps),

        ip_address: safeStringValue(metrics.ip_address)        ip_address: safeStringValue(metrics.ip_address)

      },      },

      // Extra metadata (this column exists)      // Extra metadata (this column exists)

      extra: {      extra: {

        user_id: agent.user_id,        user_id: agent.user_id,

        agent_type: agent.agent_type,        agent_type: agent.agent_type,

        timestamp: timestamp || new Date().toISOString(),        timestamp: timestamp || new Date().toISOString(),

        schema_version: 'v3.1.1-block-devices-fixed',        schema_version: 'v3.1.1-block-devices-fixed',

        original_metrics: metrics,        original_metrics: metrics,

        // Store system metrics in extra since system_metrics column doesn't exist        // Store system metrics in extra since system_metrics column doesn't exist

        system_metrics: {        system_metrics: {

          cpu_percent: safeMetricValue(metrics.cpu_usage) || safeMetricValue(metrics.cpu_percent),          cpu_percent: safeMetricValue(metrics.cpu_usage) || safeMetricValue(metrics.cpu_percent),

          memory_mb: metrics.memory_usage != null ? Math.round(Number(metrics.memory_usage)) : null,          memory_mb: metrics.memory_usage != null ? Math.round(Number(metrics.memory_usage)) : null,

          disk_percent: safeMetricValue(metrics.disk_usage) || safeMetricValue(metrics.disk_percent),          disk_percent: safeMetricValue(metrics.disk_usage) || safeMetricValue(metrics.disk_percent),

          agent_version: safeStringValue(metrics.agent_version),          agent_version: safeStringValue(metrics.agent_version),

          location: safeStringValue(metrics.location),          location: safeStringValue(metrics.location),

          device_fingerprint: safeStringValue(metrics.device_fingerprint)          device_fingerprint: safeStringValue(metrics.device_fingerprint)

        }        }

      }      }

    };    };

    console.log('Upserting metrics record:', JSON.stringify(metricsRecord, null, 2));    console.log('Upserting metrics record:', JSON.stringify(metricsRecord, null, 2));

    // Use UPSERT with agent_id as primary key    // Use UPSERT with agent_id as primary key

    const { error } = await supabaseAdmin.from('agent_metrics').upsert(metricsRecord, {    const { error } = await supabaseAdmin.from('agent_metrics').upsert(metricsRecord, {

      onConflict: 'agent_id',      onConflict: 'agent_id',

      ignoreDuplicates: false      ignoreDuplicates: false

    });    });

    if (error) {    if (error) {

      console.error('Metrics upsert error:', error);      console.error('Metrics upsert error:', error);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Failed to store metrics',        error: 'Failed to store metrics',

        details: error.message,        details: error.message,

        hint: 'Check if agent_id exists and schema matches existing columns'        hint: 'Check if agent_id exists and schema matches existing columns'

      }), {      }), {

        status: 500,        status: 500,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    console.log('Metrics stored successfully via UPSERT');    console.log('Metrics stored successfully via UPSERT');

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      success: true,      success: true,

      message: 'Metrics stored successfully',      message: 'Metrics stored successfully',

      agent_id: finalAgentId,      agent_id: finalAgentId,

      schema_version: 'v3.1.1-block-devices-fixed'      schema_version: 'v3.1.1-block-devices-fixed'

    }), {    }), {

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleAgentMetrics:', error);    console.error('Error in handleAgentMetrics:', error);

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      error: 'Internal server error',      error: 'Internal server error',

      details: error.message      details: error.message

    }), {    }), {

      status: 500,      status: 500,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  }  }

}}

// GET /agent-auth-api/health/:id - Uses existing schema columns (FIXED block_devices)// GET /agent-auth-api/health/:id - Uses existing schema columns (FIXED block_devices)

async function handleAgentHealth(req) {async function handleAgentHealth(req) {

  try {  try {

    const url = new URL(req.url);    const url = new URL(req.url);

    const pathParts = url.pathname.split('/');    const pathParts = url.pathname.split('/');

    // URL structure: /agent-auth-api/health/{agentId}    // URL structure: /agent-auth-api/health/{agentId}

    // pathParts: ['', 'agent-auth-api', 'health', '{agentId}']    // pathParts: ['', 'agent-auth-api', 'health', '{agentId}']

    const agentId = pathParts[3];    const agentId = pathParts[3];

    if (!agentId) {    if (!agentId) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Agent ID is required'        error: 'Agent ID is required'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    const { agent, error: authError } = await validateAgentToken(req);    const { agent, error: authError } = await validateAgentToken(req);

    if (authError) {    if (authError) {

      console.error('Agent auth error:', authError);      console.error('Agent auth error:', authError);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: authError        error: authError

      }), {      }), {

        status: 401,        status: 401,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    console.log('Debug agent object:', JSON.stringify(agent, null, 2));    console.log('Debug agent object:', JSON.stringify(agent, null, 2));

    console.log('Debug agentId from URL:', agentId);    console.log('Debug agentId from URL:', agentId);

    // Verify the agent ID matches the token    // Verify the agent ID matches the token

    if (agent.agent_id !== agentId) {    if (agent.agent_id !== agentId) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Agent ID mismatch',        error: 'Agent ID mismatch',

        expected: agentId,        expected: agentId,

        actual: agent.agent_id,        actual: agent.agent_id,

        debug: 'Check agent token structure'        debug: 'Check agent token structure'

      }), {      }), {

        status: 403,        status: 403,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Get latest metrics using existing schema    // Get latest metrics using existing schema

    const { data: latestMetrics, error } = await supabaseAdmin.from('agent_metrics').select('*').eq('agent_id', agentId).order('recorded_at', {    const { data: latestMetrics, error } = await supabaseAdmin.from('agent_metrics').select('*').eq('agent_id', agentId).order('recorded_at', {

      ascending: false      ascending: false

    }).limit(1).single();    }).limit(1).single();

    const lastSeen = latestMetrics ? latestMetrics.recorded_at : agent.created_at;    const lastSeen = latestMetrics ? latestMetrics.recorded_at : agent.created_at;

    const isHealthy = latestMetrics && new Date().getTime() - new Date(lastSeen).getTime() < 5 * 60 * 1000; // 5 minutes    const isHealthy = latestMetrics && new Date().getTime() - new Date(lastSeen).getTime() < 5 * 60 * 1000; // 5 minutes

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      agent_id: agentId,      agent_id: agentId,

      status: isHealthy ? 'healthy' : 'stale',      status: isHealthy ? 'healthy' : 'stale',

      last_seen: lastSeen,      last_seen: lastSeen,

      agent_name: agent.agent_registrations?.agent_name || 'Unknown',      agent_name: agent.agent_registrations?.agent_name || 'Unknown',

      uptime: Math.floor((new Date().getTime() - new Date(agent.created_at).getTime()) / 1000),      uptime: Math.floor((new Date().getTime() - new Date(agent.created_at).getTime()) / 1000),

      latest_metrics: latestMetrics ? {      latest_metrics: latestMetrics ? {

        // Individual columns that exist        // Individual columns that exist

        cpu_percent: latestMetrics.cpu_percent,        cpu_percent: latestMetrics.cpu_percent,

        memory_mb: latestMetrics.memory_mb,        memory_mb: latestMetrics.memory_mb,

        disk_percent: latestMetrics.disk_percent,        disk_percent: latestMetrics.disk_percent,

        network_in_kbps: latestMetrics.network_in_kbps,        network_in_kbps: latestMetrics.network_in_kbps,

        network_out_kbps: latestMetrics.network_out_kbps,        network_out_kbps: latestMetrics.network_out_kbps,

        ip_address: latestMetrics.ip_address,        ip_address: latestMetrics.ip_address,

        location: latestMetrics.location,        location: latestMetrics.location,

        agent_version: latestMetrics.agent_version,        agent_version: latestMetrics.agent_version,

        kernel_version: latestMetrics.kernel_version,        kernel_version: latestMetrics.kernel_version,

        device_fingerprint: latestMetrics.device_fingerprint,        device_fingerprint: latestMetrics.device_fingerprint,

        // JSON columns that exist        // JSON columns that exist

        load_averages: latestMetrics.load_averages,        load_averages: latestMetrics.load_averages,

        load1: latestMetrics.load_averages?.load1,        load1: latestMetrics.load_averages?.load1,

        load5: latestMetrics.load_averages?.load5,        load5: latestMetrics.load_averages?.load5,

        load15: latestMetrics.load_averages?.load15,        load15: latestMetrics.load_averages?.load15,

        os_info: latestMetrics.os_info,        os_info: latestMetrics.os_info,

        filesystem_info: latestMetrics.filesystem_info,        filesystem_info: latestMetrics.filesystem_info,

        // FIXED: Include block_devices in health response        // FIXED: Include block_devices in health response

        block_devices: latestMetrics.block_devices,        block_devices: latestMetrics.block_devices,

        network_stats: latestMetrics.network_stats,        network_stats: latestMetrics.network_stats,

        // System metrics from extra field        // System metrics from extra field

        system_metrics: latestMetrics.extra?.system_metrics,        system_metrics: latestMetrics.extra?.system_metrics,

        recorded_at: latestMetrics.recorded_at,        recorded_at: latestMetrics.recorded_at,

        schema_version: latestMetrics.extra?.schema_version || 'legacy'        schema_version: latestMetrics.extra?.schema_version || 'legacy'

      } : null      } : null

    }), {    }), {

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleAgentHealth:', error);    console.error('Error in handleAgentHealth:', error);

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      error: 'Internal server error',      error: 'Internal server error',

      details: error.message      details: error.message

    }), {    }), {

      status: 500,      status: 500,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  }  }

}}

// POST /agent/heartbeat - Simple heartbeat endpoint// POST /agent/heartbeat - Simple heartbeat endpoint

async function handleAgentHeartbeat(req) {async function handleAgentHeartbeat(req) {

  try {  try {

    const { agent, error: authError } = await validateAgentToken(req);    const { agent, error: authError } = await validateAgentToken(req);

    if (authError) {    if (authError) {

      console.error('Agent auth error:', authError);      console.error('Agent auth error:', authError);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: authError        error: authError

      }), {      }), {

        status: 401,        status: 401,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    const body = await req.json().catch(()=>({}));    const body = await req.json().catch(()=>({}));

    const { timestamp, agent_id } = body;    const { timestamp, agent_id } = body;

    // Use agent_id from request body if provided, otherwise use from token    // Use agent_id from request body if provided, otherwise use from token

    const finalAgentId = agent_id || agent.agent_id;    const finalAgentId = agent_id || agent.agent_id;

    // Simple heartbeat - just update the timestamp    // Simple heartbeat - just update the timestamp

    const heartbeatRecord = {    const heartbeatRecord = {

      agent_id: finalAgentId,      agent_id: finalAgentId,

      recorded_at: timestamp || new Date().toISOString(),      recorded_at: timestamp || new Date().toISOString(),

      extra: {      extra: {

        heartbeat: true,        heartbeat: true,

        user_id: agent.user_id,        user_id: agent.user_id,

        agent_type: agent.agent_type,        agent_type: agent.agent_type,

        timestamp: timestamp || new Date().toISOString()        timestamp: timestamp || new Date().toISOString()

      }      }

    };    };

    // Use UPSERT for heartbeat    // Use UPSERT for heartbeat

    const { error } = await supabaseAdmin.from('agent_metrics').upsert(heartbeatRecord, {    const { error } = await supabaseAdmin.from('agent_metrics').upsert(heartbeatRecord, {

      onConflict: 'agent_id',      onConflict: 'agent_id',

      ignoreDuplicates: false      ignoreDuplicates: false

    });    });

    if (error) {    if (error) {

      console.error('Heartbeat upsert error:', error);      console.error('Heartbeat upsert error:', error);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Failed to record heartbeat',        error: 'Failed to record heartbeat',

        details: error.message        details: error.message

      }), {      }), {

        status: 500,        status: 500,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Silent success - no console.log for successful heartbeats    // Silent success - no console.log for successful heartbeats

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      success: true,      success: true,

      agent_id: finalAgentId,      agent_id: finalAgentId,

      timestamp: timestamp || new Date().toISOString()      timestamp: timestamp || new Date().toISOString()

    }), {    }), {

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleAgentHeartbeat:', error);    console.error('Error in handleAgentHeartbeat:', error);

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      error: 'Internal server error',      error: 'Internal server error',

      details: error.message      details: error.message

    }), {    }), {

      status: 500,      status: 500,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

  }  }

}}

function handleOptions() {function handleOptions() {

  return new Response(null, {  return new Response(null, {

    status: 204,    status: 204,

    headers: {    headers: {

      'Access-Control-Allow-Origin': '*',      'Access-Control-Allow-Origin': '*',

      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',

      'Access-Control-Allow-Headers': 'Content-Type, Authorization',      'Access-Control-Allow-Headers': 'Content-Type, Authorization',

      'Access-Control-Max-Age': '86400'      'Access-Control-Max-Age': '86400'

    }    }

  });  });

}}

Deno.serve(async (req)=>{Deno.serve(async (req)=>{

  const url = new URL(req.url);  const url = new URL(req.url);

  // Handle CORS preflight requests  // Handle CORS preflight requests

  if (req.method === 'OPTIONS') {  if (req.method === 'OPTIONS') {

    return handleOptions();    return handleOptions();

  }  }

  // Add CORS headers to all responses  // Add CORS headers to all responses

  const corsHeaders = {  const corsHeaders = {

    'Access-Control-Allow-Origin': '*',    'Access-Control-Allow-Origin': '*',

    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',

    'Access-Control-Allow-Headers': 'Content-Type, Authorization'    'Access-Control-Allow-Headers': 'Content-Type, Authorization'

  };  };

  try {  try {

    let response;    let response;

    console.log(`API Request: ${req.method} ${url.pathname}`);    console.log(`API Request: ${req.method} ${url.pathname}`);

    // Handle different endpoint routing - Supabase functions see full path    // Handle different endpoint routing - Supabase functions see full path

    if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('register'))) {    if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('register'))) {

      // Check if this is a register request by looking at the body      // Check if this is a register request by looking at the body

      const body = await req.clone().json().catch(()=>({}));      const body = await req.clone().json().catch(()=>({}));

      if (body.device_code) {      if (body.device_code) {

        console.log('Handling agent registration (deprecated)');        console.log('Handling agent registration (deprecated)');

        response = await handleAgentRegister(req);        response = await handleAgentRegister(req);

      } else {      } else {

        response = new Response(JSON.stringify({        response = new Response(JSON.stringify({

          error: 'Missing device_code for registration. Use device-auth flow instead.'          error: 'Missing device_code for registration. Use device-auth flow instead.'

        }), {        }), {

          status: 400,          status: 400,

          headers: {          headers: {

            'Content-Type': 'application/json'            'Content-Type': 'application/json'

          }          }

        });        });

      }      }

    } else if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('metrics'))) {    } else if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('metrics'))) {

      response = await handleAgentMetrics(req);      response = await handleAgentMetrics(req);

    } else if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('heartbeat'))) {    } else if (req.method === 'POST' && (url.pathname === '/' || url.pathname.includes('heartbeat'))) {

      response = await handleAgentHeartbeat(req);      response = await handleAgentHeartbeat(req);

    } else if (req.method === 'GET' && (url.pathname === '/' || url.pathname.includes('health'))) {    } else if (req.method === 'GET' && (url.pathname === '/' || url.pathname.includes('health'))) {

      response = await handleAgentHealth(req);      response = await handleAgentHealth(req);

    } else {    } else {

      response = new Response(JSON.stringify({      response = new Response(JSON.stringify({

        error: 'Not Found',        error: 'Not Found',

        method: req.method,        method: req.method,

        path: url.pathname,        path: url.pathname,

        available_endpoints: [        available_endpoints: [

          'POST /register (deprecated - use device-auth flow)',          'POST /register (deprecated - use device-auth flow)',

          'POST /metrics (UPSERT with existing schema + block_devices)',          'POST /metrics (UPSERT with existing schema + block_devices)',

          'POST /heartbeat (silent heartbeat)',          'POST /heartbeat (silent heartbeat)',

          'GET /:id/health (existing schema compatible + block_devices)'          'GET /:id/health (existing schema compatible + block_devices)'

        ]        ]

      }), {      }), {

        status: 404,        status: 404,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Add CORS headers to response    // Add CORS headers to response

    Object.entries(corsHeaders).forEach(([key, value])=>{    Object.entries(corsHeaders).forEach(([key, value])=>{

      response.headers.set(key, value);      response.headers.set(key, value);

    });    });

    return response;    return response;

  } catch (error) {  } catch (error) {

    console.error('Unhandled error:', error);    console.error('Unhandled error:', error);

    const errorResponse = new Response(JSON.stringify({    const errorResponse = new Response(JSON.stringify({

      error: 'Internal server error',      error: 'Internal server error',

      details: error.message      details: error.message

    }), {    }), {

      status: 500,      status: 500,

      headers: {      headers: {

        'Content-Type': 'application/json'        'Content-Type': 'application/json'

      }      }

    });    });

    Object.entries(corsHeaders).forEach(([key, value])=>{    Object.entries(corsHeaders).forEach(([key, value])=>{

      errorResponse.headers.set(key, value);      errorResponse.headers.set(key, value);

    });    });

    return errorResponse;    return errorResponse;

  }  }

});});
