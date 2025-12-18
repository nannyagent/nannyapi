import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";
const supabase = createClient(Deno.env.get('SUPABASE_URL') ?? '', Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '');
const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization, apikey, x-client-info'
};
// ClickHouse Configuration
const CLICKHOUSE_URL = Deno.env.get('CLICKHOUSE_URL') || 'https://clickhouse.nannyai.dev';
const CLICKHOUSE_DATABASE = Deno.env.get('CLICKHOUSE_DATABASE') || 'tensorzero';
const CLICKHOUSE_USER = Deno.env.get('CLICKHOUSE_USER') || 'default';
const CLICKHOUSE_PASSWORD = Deno.env.get('CLICKHOUSE_PASSWORD') || '';
// Fetch all inferences for an episode with full details
async function fetchClickHouseDataByEpisode(episodeId, limit = 50, offset = 0) {
  const query = `
    SELECT 
      ci.id,
      ci.function_name,
      ci.variant_name,
      ci.episode_id,
      ci.input,
      ci.output,
      ci.tool_params,
      ci.inference_params,
      ci.processing_time_ms,
      ci.ttft_ms,
      ci.tags,
      ci.extra_body,
      ci.timestamp,
      mi.id as model_inference_id,
      mi.model_name,
      mi.model_provider_name,
      mi.input_tokens,
      mi.output_tokens,
      mi.response_time_ms,
      mi.ttft_ms as model_ttft_ms,
      mi.raw_request,
      mi.raw_response,
      mi.timestamp as model_timestamp
    FROM ChatInference ci
    LEFT JOIN ModelInference mi ON ci.id = mi.inference_id
    WHERE ci.episode_id = '${episodeId}'
    ORDER BY ci.timestamp ASC
    LIMIT ${limit} OFFSET ${offset}
    FORMAT JSONCompact
  `;
  const url = `${CLICKHOUSE_URL}/?user=${encodeURIComponent(CLICKHOUSE_USER)}&password=${encodeURIComponent(CLICKHOUSE_PASSWORD)}&database=${encodeURIComponent(CLICKHOUSE_DATABASE)}`;
  const response = await fetch(url, {
    method: 'POST',
    body: query
  });
  if (!response.ok) {
    console.error('ClickHouse query error:', await response.text());
    return null;
  }
  return await response.json();
}
// Fetch individual inference details by inference_id with ModelInference metadata
async function fetchClickHouseInferenceById(inferenceId) {
  const query = `
    SELECT 
      ci.id,
      ci.function_name,
      ci.variant_name,
      ci.episode_id,
      ci.input,
      ci.output,
      ci.tool_params,
      ci.inference_params,
      ci.processing_time_ms,
      ci.ttft_ms,
      ci.tags,
      ci.extra_body,
      ci.timestamp,
      mi.id as model_inference_id,
      mi.model_name,
      mi.model_provider_name,
      mi.input_tokens,
      mi.output_tokens,
      mi.response_time_ms,
      mi.ttft_ms as model_ttft_ms,
      mi.raw_request,
      mi.raw_response,
      mi.timestamp as model_timestamp
    FROM ChatInference ci
    LEFT JOIN ModelInference mi ON ci.id = mi.inference_id
    WHERE ci.id = '${inferenceId}'
    FORMAT JSONCompact
  `;
  const url = `${CLICKHOUSE_URL}/?user=${encodeURIComponent(CLICKHOUSE_USER)}&password=${encodeURIComponent(CLICKHOUSE_PASSWORD)}&database=${encodeURIComponent(CLICKHOUSE_DATABASE)}`;
  const response = await fetch(url, {
    method: 'POST',
    body: query
  });
  if (!response.ok) {
    console.error('ClickHouse query error:', await response.text());
    return null;
  }
  return await response.json();
}
// Fetch feedback for an inference
async function fetchClickHouseFeedback(inferenceId) {
  const query = `
    SELECT 
      id,
      target_id,
      metric_name,
      value,
      timestamp
    FROM BooleanMetricFeedback
    WHERE target_id = '${inferenceId}'
    UNION ALL
    SELECT 
      id,
      target_id,
      metric_name,
      value,
      timestamp
    FROM FloatMetricFeedback
    WHERE target_id = '${inferenceId}'
    UNION ALL
    SELECT 
      id,
      target_id,
      metric_name,
      value,
      timestamp
    FROM CommentFeedback
    WHERE target_id = '${inferenceId}'
    UNION ALL
    SELECT 
      id,
      target_id,
      metric_name,
      value,
      timestamp
    FROM DemonstrationFeedback
    WHERE target_id = '${inferenceId}'
    FORMAT JSONCompact
  `;
  const url = `${CLICKHOUSE_URL}/?user=${encodeURIComponent(CLICKHOUSE_USER)}&password=${encodeURIComponent(CLICKHOUSE_PASSWORD)}&database=${encodeURIComponent(CLICKHOUSE_DATABASE)}`;
  const response = await fetch(url, {
    method: 'POST',
    body: query
  });
  if (!response.ok) {
    console.error('ClickHouse feedback query error:', await response.text());
    return null;
  }
  return await response.json();
}
async function validateToken(req) {
  const authHeader = req.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return {
      user: null,
      error: 'Missing auth header'
    };
  }
  const token = authHeader.substring(7);
  try {
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (error || !user) return {
      user: null,
      error: 'Invalid token'
    };
    return {
      user,
      error: null
    };
  } catch (e) {
    return {
      user: null,
      error: 'Token validation failed'
    };
  }
}
function generateInvestigationID() {
  const now = new Date();
  const dateStr = now.toISOString().split('T')[0].replace(/-/g, '');
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let random = '';
  for(let i = 0; i < 6; i++){
    random += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return `INV-${dateStr}-${random}`;
}
// Handle GET /episode/{episodeId}/inferences
async function handleGetEpisodeInferences(req, episodeId) {
  const { user, error } = await validateToken(req);
  if (error || !user) {
    return new Response(JSON.stringify({
      error: error || 'Unauthorized'
    }), {
      status: 401,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  try {
    const url = new URL(req.url);
    const limit = parseInt(url.searchParams.get('limit') || '50');
    const offset = parseInt(url.searchParams.get('offset') || '0');
    // Fetch from ClickHouse
    const rawData = await fetchClickHouseDataByEpisode(episodeId, limit, offset);
    if (!rawData || !rawData.data) {
      return new Response(JSON.stringify({
        inferences: [],
        pagination: {
          limit,
          offset,
          total: 0,
          has_more: false
        }
      }), {
        status: 200,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    // Parse the raw ClickHouse response into structured inferences
    const inferences = rawData.data.map((row)=>({
        id: row[0],
        function_name: row[1],
        variant_name: row[2],
        episode_id: row[3],
        input: row[4],
        output: row[5],
        tool_params: row[6],
        inference_params: row[7],
        processing_time_ms: row[8],
        ttft_ms: row[9],
        tags: row[10],
        extra_body: row[11],
        timestamp: row[12],
        model_inference: row[13] ? {
          id: row[13],
          model_name: row[14],
          model_provider_name: row[15],
          input_tokens: row[16],
          output_tokens: row[17],
          response_time_ms: row[18],
          ttft_ms: row[19],
          raw_request: row[20],
          raw_response: row[21],
          timestamp: row[22]
        } : null
      }));
    // Query total count for pagination
    const countQuery = `
      SELECT COUNT(*) as total
      FROM ChatInference
      WHERE episode_id = '${episodeId}'
      FORMAT JSONCompact
    `;
    const url_string = `${CLICKHOUSE_URL}/?user=${encodeURIComponent(CLICKHOUSE_USER)}&password=${encodeURIComponent(CLICKHOUSE_PASSWORD)}&database=${encodeURIComponent(CLICKHOUSE_DATABASE)}`;
    const countResponse = await fetch(url_string, {
      method: 'POST',
      body: countQuery
    });
    let total = 0;
    if (countResponse.ok) {
      const countData = await countResponse.json();
      total = countData.data?.[0]?.[0] || 0;
    }
    return new Response(JSON.stringify({
      inferences,
      pagination: {
        limit,
        offset,
        total,
        has_more: offset + limit < total
      }
    }), {
      status: 200,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Error fetching episode inferences:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
}
// Handle GET /inference/{inferenceId}
async function handleGetInferenceDetails(req, inferenceId) {
  const { user, error } = await validateToken(req);
  if (error || !user) {
    return new Response(JSON.stringify({
      error: error || 'Unauthorized'
    }), {
      status: 401,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  try {
    // Fetch inference data from ClickHouse (includes ModelInference join)
    const inferenceData = await fetchClickHouseInferenceById(inferenceId);
    if (!inferenceData || !inferenceData.data || inferenceData.data.length === 0) {
      return new Response(JSON.stringify({
        error: 'Inference not found'
      }), {
        status: 404,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    // Parse the raw ClickHouse response into structured data
    const row = inferenceData.data[0];
    const structuredInference = {
      id: row[0],
      function_name: row[1],
      variant_name: row[2],
      episode_id: row[3],
      input: row[4],
      output: row[5],
      tool_params: row[6],
      inference_params: row[7],
      processing_time_ms: row[8],
      ttft_ms: row[9],
      tags: row[10],
      extra_body: row[11],
      timestamp: row[12],
      model_inference: row[13] ? {
        id: row[13],
        model_name: row[14],
        model_provider_name: row[15],
        input_tokens: row[16],
        output_tokens: row[17],
        response_time_ms: row[18],
        ttft_ms: row[19],
        raw_request: row[20],
        raw_response: row[21],
        timestamp: row[22]
      } : null
    };
    // Fetch and structure feedback data
    const feedbackData = await fetchClickHouseFeedback(inferenceId);
    const structuredFeedback = feedbackData?.data?.map((fbRow)=>({
        id: fbRow[0],
        target_id: fbRow[1],
        metric_name: fbRow[2],
        value: fbRow[3],
        timestamp: fbRow[4]
      })) || [];
    return new Response(JSON.stringify({
      inference: structuredInference,
      feedback: structuredFeedback
    }), {
      status: 200,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Error fetching inference details:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
}
async function handleGetInvestigations(req) {
  const { user, error } = await validateToken(req);
  if (error || !user) {
    return new Response(JSON.stringify({
      error: 'Unauthorized'
    }), {
      status: 401,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  try {
    const url = new URL(req.url);
    const page = parseInt(url.searchParams.get('page') || '1');
    const limit = parseInt(url.searchParams.get('limit') || '10');
    const offset = (page - 1) * limit;
    const { data: userAgents, error: agentsError } = await supabase.from('agents').select('id').eq('owner', user.id);
    if (agentsError || !userAgents || userAgents.length === 0) {
      return new Response(JSON.stringify({
        investigations: [],
        pagination: {
          page,
          limit,
          total: 0,
          total_pages: 0
        }
      }), {
        status: 200,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const agentIds = userAgents.map((a)=>a.id);
    const { data: investigations, count, error: invError } = await supabase.from('investigations').select('*, agents(id, name, status)', {
      count: 'exact'
    }).in('agent_id', agentIds).order('created_at', {
      ascending: false
    }).range(offset, offset + limit - 1);
    if (invError) {
      return new Response(JSON.stringify({
        error: 'Failed to fetch investigations'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const totalPages = count ? Math.ceil(count / limit) : 0;
    return new Response(JSON.stringify({
      investigations: investigations || [],
      pagination: {
        page,
        limit,
        total: count || 0,
        total_pages: totalPages,
        has_next: page < totalPages,
        has_prev: page > 1
      }
    }), {
      status: 200,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Error fetching investigations:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error'
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
}
Deno.serve(async (req)=>{
  if (req.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: corsHeaders
    });
  }
  if (req.method === 'GET') {
    const url = new URL(req.url);
    const pathParts = url.pathname.split('/').filter((p)=>p);
    // Check for /episode/{episodeId}/inferences
    if (pathParts.length >= 3 && pathParts[pathParts.length - 3] === 'episode' && pathParts[pathParts.length - 1] === 'inferences') {
      const episodeId = pathParts[pathParts.length - 2];
      return handleGetEpisodeInferences(req, episodeId);
    }
    // Check for /inference/{inferenceId}
    if (pathParts.length >= 2 && pathParts[pathParts.length - 2] === 'inference') {
      const inferenceId = pathParts[pathParts.length - 1];
      return handleGetInferenceDetails(req, inferenceId);
    }
    // Default: list investigations
    return handleGetInvestigations(req);
  }
  if (req.method !== 'POST') {
    return new Response(JSON.stringify({
      error: 'Method not allowed'
    }), {
      status: 405,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
  try {
    const body = await req.json();
    const { agent_id, issue, priority = 'medium', initiated_by } = body;
    console.log('[EDGE v31] Investigation request - agent_id:', agent_id, 'issue:', issue);
    if (!agent_id || !issue) {
      console.error('[EDGE v31] Missing required fields');
      return new Response(JSON.stringify({
        error: 'agent_id and issue are required'
      }), {
        status: 400,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const { data: agent, error: agentError } = await supabase.from('agents').select('*').eq('id', agent_id).single();
    if (agentError || !agent) {
      console.error('[EDGE v31] Agent not found:', agentError);
      return new Response(JSON.stringify({
        error: 'Agent not found'
      }), {
        status: 404,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    if (!agent.websocket_connected) {
      console.error('[EDGE v31] Agent not connected');
      return new Response(JSON.stringify({
        error: 'Agent not connected'
      }), {
        status: 503,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    const investigationId = generateInvestigationID();
    console.log('[EDGE v31] Generated investigation ID:', investigationId);
    const { error: invError } = await supabase.from('investigations').insert({
      investigation_id: investigationId,
      agent_id,
      issue,
      priority,
      status: 'pending',
      target_agents: [
        agent_id
      ],
      initiated_by: initiated_by || 'backend',
      initiated_at: new Date().toISOString()
    });
    if (invError) {
      console.error('[EDGE v31] Failed to create investigation:', invError);
      return new Response(JSON.stringify({
        error: 'Failed to create investigation'
      }), {
        status: 500,
        headers: {
          ...corsHeaders,
          'Content-Type': 'application/json'
        }
      });
    }
    console.log('[EDGE v31] Investigation created:', investigationId);
    return new Response(JSON.stringify({
      success: true,
      investigation_id: investigationId,
      message: 'Investigation queued - agent will process via websocket',
      ai_powered: true
    }), {
      status: 200,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('[EDGE v31] Error:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error.message
    }), {
      status: 500,
      headers: {
        ...corsHeaders,
        'Content-Type': 'application/json'
      }
    });
  }
});
