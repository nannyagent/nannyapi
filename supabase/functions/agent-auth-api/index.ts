import { createClient } from "npm:@supabase/supabase-js@2.32.0";
const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';
const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || '';
const SUPABASE_ANON_KEY = Deno.env.get('SUPABASE_ANON_KEY') || '';
if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY || !SUPABASE_ANON_KEY) {
  console.error('Missing required environment variables');
}
const supabaseAdmin = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {
  auth: {
    persistSession: false
  }
});
const supabase = createClient(SUPABASE_URL, SUPABASE_ANON_KEY, {
  auth: {
    persistSession: false
  }
});
function extractHostnameFromClientId(clientId) {
  if (!clientId || typeof clientId !== 'string') return 'nannyagent';
  const prefix = 'nannyagent-';
  if (clientId.startsWith(prefix)) {
    const hostname = clientId.substring(prefix.length).trim();
    return hostname || 'nannyagent';
  }
  return clientId || 'nannyagent';
}
async function findUniqueAgentName(baseHostname, userId, currentAgentId) {
  const sanitized = baseHostname.replace(/[^a-zA-Z0-9-_]/g, '-').substring(0, 50);
  let query = supabaseAdmin.from('agents').select('name').eq('owner', userId).ilike('name', `${sanitized}%`).order('name', {
    ascending: true
  });
  if (currentAgentId) query = query.neq('id', currentAgentId);
  const { data: existing, error } = await query;
  if (error) {
    console.error('Error checking existing agent names:', error);
    return sanitized;
  }
  if (!existing || existing.length === 0) return sanitized;
  const exactMatch = existing.find((a)=>a.name === sanitized);
  if (!exactMatch) return sanitized;
  const suffixes = existing.map((a)=>{
    const match = a.name.match(new RegExp(`^${sanitized}-(\\d+)$`));
    return match ? parseInt(match[1]) : 0;
  }).filter((n)=>n > 0);
  const nextSuffix = suffixes.length > 0 ? Math.max(...suffixes) + 1 : 1;
  return `${sanitized}-${nextSuffix}`;
}
async function validateSupabaseAuth(req) {
  const authHeader = req.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return {
    user: null,
    error: 'Missing or invalid authorization header'
  };
  const token = authHeader.substring(7);
  try {
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (error || !user) return {
      user: null,
      error: 'Invalid token'
    };
    return {
      user
    };
  } catch (error) {
    return {
      user: null,
      error: 'Token validation failed'
    };
  }
}
async function validateAgentToken(req) {
  const authHeader = req.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) return {
    agent: null,
    error: 'Missing or invalid authorization header'
  };
  const token = authHeader.substring(7);
  try {
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (!error && user && (user.app_metadata?.role === 'agent' || user.user_metadata?.agent_type)) {
      return {
        agent: {
          agent_id: user.id,
          user_id: user.user_metadata?.authorized_by || user.id,
          email: user.email,
          agent_type: user.user_metadata?.agent_type || 'nannyagent',
          device_session_id: user.user_metadata?.device_session_id,
          created_at: user.created_at,
          agent_registrations: {
            agent_name: user.user_metadata?.agent_type || 'NannyAgent',
            agent_type: user.user_metadata?.agent_type || 'nannyagent'
          }
        }
      };
    }
    const AGENT_TOKEN_HMAC_SECRET = Deno.env.get('AGENT_TOKEN_HMAC_SECRET') || '';
    if (!AGENT_TOKEN_HMAC_SECRET) return {
      agent: null,
      error: 'Token validation system not properly configured'
    };
    const { createHmac } = await import('node:crypto');
    const tokenHash = createHmac('sha256', AGENT_TOKEN_HMAC_SECRET).update(token).digest('hex');
    const { data: agentToken, error: tokenError } = await supabaseAdmin.from('agent_tokens').select('*, agent_registrations(*)').eq('token_hash', tokenHash).eq('revoked', false).single();
    if (tokenError || !agentToken) return {
      agent: null,
      error: 'Invalid agent token'
    };
    if (new Date(agentToken.expires_at) < new Date()) return {
      agent: null,
      error: 'Token expired'
    };
    return {
      agent: agentToken
    };
  } catch (error) {
    return {
      agent: null,
      error: 'Agent token validation failed'
    };
  }
}
function safeMetricValue(value) {
  return typeof value === 'number' && !isNaN(value) ? value : null;
}
function safeStringValue(value) {
  return typeof value === 'string' && value.trim() !== '' ? value.trim() : null;
}
// Validation functions for metrics with proper ranges
function validateCpuPercent(value) {
  const num = safeMetricValue(value);
  return num !== null && num >= 0 && num <= 100 ? num : null;
}
function validateMemoryMb(value) {
  const num = safeMetricValue(value);
  return num !== null && num > 0 ? num : null;
}
function validateDiskPercent(value) {
  const num = safeMetricValue(value);
  return num !== null && num >= 0 && num <= 100 ? num : null;
}
function validateNetworkKbps(value) {
  const num = safeMetricValue(value);
  return num !== null && num >= 0 ? num : null;
}
function validateLoadAverage(value) {
  const num = safeMetricValue(value);
  return num !== null && num >= 0 ? num : null;
}
async function handleAgentMetrics(req) {
  try {
    const { agent, error: authError } = await validateAgentToken(req);
    if (authError) return new Response(JSON.stringify({
      error: authError
    }), {
      status: 401,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    const body = await req.json().catch(()=>({}));
    const { metrics, timestamp, agent_id } = body;
    if (!metrics || typeof metrics !== 'object') return new Response(JSON.stringify({
      error: 'metrics object is required'
    }), {
      status: 400,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    const finalAgentId = agent_id || agent.agent_id;
    const metricsHostname = safeStringValue(metrics.hostname);
    if (metricsHostname) {
      const { data: agentRecord } = await supabaseAdmin.from('agents').select('owner, name, metadata').eq('id', finalAgentId).single();
      if (agentRecord) {
        const newAgentName = await findUniqueAgentName(metricsHostname, agentRecord.owner, finalAgentId);
        if (newAgentName !== agentRecord.name) {
          console.log('Updating agent name from metrics:', {
            agent_id: finalAgentId,
            old_name: agentRecord.name,
            new_name: newAgentName,
            hostname: metricsHostname
          });
          await supabaseAdmin.from('agents').update({
            name: newAgentName,
            metadata: {
              ...agentRecord.metadata || {},
              hostname: metricsHostname,
              last_hostname_update: new Date().toISOString()
            }
          }).eq('id', finalAgentId);
        }
      }
    }
    const metricsRecord = {
      agent_id: finalAgentId,
      recorded_at: timestamp || new Date().toISOString(),
      cpu_percent: validateCpuPercent(metrics.cpu_usage) || validateCpuPercent(metrics.cpu_percent),
      memory_mb: validateMemoryMb(metrics.memory_usage != null ? Math.round(Number(metrics.memory_usage)) : null),
      disk_percent: validateDiskPercent(metrics.disk_usage) || validateDiskPercent(metrics.disk_percent),
      network_in_kbps: validateNetworkKbps(metrics.network_in_kbps),
      network_out_kbps: validateNetworkKbps(metrics.network_out_kbps),
      ip_address: safeStringValue(metrics.ip_address),
      location: safeStringValue(metrics.location),
      agent_version: safeStringValue(metrics.agent_version),
      kernel_version: safeStringValue(metrics.kernel_version),
      device_fingerprint: safeStringValue(metrics.device_fingerprint),
      load_averages: {
        load1: validateLoadAverage(metrics.load_averages?.load1 || metrics.load1),
        load5: validateLoadAverage(metrics.load_averages?.load5 || metrics.load5),
        load15: validateLoadAverage(metrics.load_averages?.load15 || metrics.load15)
      },
      os_info: {
        name: safeStringValue(metrics.os_info?.name || metrics.os_name),
        version: safeStringValue(metrics.os_info?.version || metrics.os_version),
        architecture: safeStringValue(metrics.os_info?.architecture || metrics.os_architecture),
        platform: safeStringValue(metrics.os_info?.platform || metrics.os_platform),
        family: safeStringValue(metrics.os_info?.family || metrics.os_family)
      },
      filesystem_info: metrics.filesystem_info || [],
      block_devices: metrics.block_devices || [],
      network_stats: {
        network_in_kbps: validateNetworkKbps(metrics.network_stats?.network_in_kbps || metrics.network_in_kbps),
        network_out_kbps: validateNetworkKbps(metrics.network_stats?.network_out_kbps || metrics.network_out_kbps)
      },
      extra: {
        user_id: agent.user_id,
        agent_type: agent.agent_type,
        timestamp: timestamp || new Date().toISOString(),
        schema_version: 'v3.3.0-metrics-normalized',
        hostname: metricsHostname
      }
    };
    const { error } = await supabaseAdmin.from('agent_metrics').upsert(metricsRecord, {
      onConflict: 'agent_id',
      ignoreDuplicates: false
    });
    if (error) return new Response(JSON.stringify({
      error: 'Failed to store metrics',
      details: error.message
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    return new Response(JSON.stringify({
      success: true,
      agent_id: finalAgentId
    }), {
      headers: {
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error.message
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
}
async function handleAgentHeartbeat(req) {
  try {
    const { agent, error: authError } = await validateAgentToken(req);
    if (authError) return new Response(JSON.stringify({
      error: authError
    }), {
      status: 401,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    const body = await req.json().catch(()=>({}));
    const { timestamp, agent_id } = body;
    const finalAgentId = agent_id || agent.agent_id;
    const { error } = await supabaseAdmin.from('agent_metrics').upsert({
      agent_id: finalAgentId,
      recorded_at: timestamp || new Date().toISOString(),
      extra: {
        heartbeat: true
      }
    }, {
      onConflict: 'agent_id'
    });
    if (error) return new Response(JSON.stringify({
      error: 'Failed to record heartbeat'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    return new Response(JSON.stringify({
      success: true,
      agent_id: finalAgentId
    }), {
      headers: {
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    return new Response(JSON.stringify({
      error: 'Internal server error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
}
function handleOptions() {
  return new Response(null, {
    status: 204,
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization'
    }
  });
}
Deno.serve(async (req)=>{
  if (req.method === 'OPTIONS') return handleOptions();
  const corsHeaders = {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization'
  };
  try {
    const url = new URL(req.url);
    let response;
    if (req.method === 'POST' && url.pathname.includes('metrics')) response = await handleAgentMetrics(req);
    else if (req.method === 'POST' && url.pathname.includes('heartbeat')) response = await handleAgentHeartbeat(req);
    else response = new Response(JSON.stringify({
      error: 'Not Found'
    }), {
      status: 404,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    Object.entries(corsHeaders).forEach(([k, v])=>response.headers.set(k, v));
    return response;
  } catch (error) {
    const errorResponse = new Response(JSON.stringify({
      error: 'Internal server error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
    Object.entries(corsHeaders).forEach(([k, v])=>errorResponse.headers.set(k, v));
    return errorResponse;
  }
});
