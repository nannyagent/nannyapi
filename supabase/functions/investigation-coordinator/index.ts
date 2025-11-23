import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";
const supabase = createClient(Deno.env.get('SUPABASE_URL') ?? '', Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '');

// CORS Headers for all responses
const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization, apikey, x-client-info',
  'Content-Type': 'application/json'
};

// ClickHouse Configuration
const CLICKHOUSE_URL = Deno.env.get('CLICKHOUSE_URL') || 'https://clickhouse.nannyai.dev';
const CLICKHOUSE_DATABASE = Deno.env.get('CLICKHOUSE_DATABASE') || 'tensorzero';
const CLICKHOUSE_USER = Deno.env.get('CLICKHOUSE_USER') || 'default';
const CLICKHOUSE_PASSWORD = Deno.env.get('CLICKHOUSE_PASSWORD') || '';

// Fetch TensorZero inference data from ClickHouse by episode_id
// Returns summary with inference IDs only
async function fetchClickHouseDataByEpisode(episodeId: string) {
  // Query for all inferences in this episode - only IDs and metadata
  const query = `
    SELECT 
      id,
      function_name,
      variant_name,
      timestamp,
      processing_time_ms
    FROM ChatInference
    WHERE episode_id = '${episodeId}'
    ORDER BY timestamp ASC
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
async function fetchClickHouseInferenceById(inferenceId: string) {
  // Query for specific inference with full details including ModelInference join
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
async function fetchClickHouseFeedback(inferenceId: string) {
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

// Format system information for TensorZero prompt (same as agent.go)
function formatSystemInfoForPrompt(metrics) {
  const osInfo = metrics.os_info || {};
  const loadAvg = metrics.load_averages || {};
  const networkStats = metrics.network_stats || {};
  const filesystems = metrics.filesystem_info || [];
  const blockDevices = metrics.block_devices || [];
  let prompt = `SYSTEM INFORMATION:\n`;
  prompt += `Hostname: ${osInfo.name || 'unknown'}\n`;
  prompt += `OS: ${osInfo.platform_version || 'unknown'} (${osInfo.kernel_arch || 'unknown'})\n`;
  prompt += `Kernel: ${metrics.kernel_version || 'unknown'}\n`;
  prompt += `CPU Usage: ${metrics.cpu_percent || 0}%\n`;
  prompt += `Memory: ${metrics.memory_mb || 0}MB used\n`;
  prompt += `Load Average: ${loadAvg.load1 || 0}, ${loadAvg.load5 || 0}, ${loadAvg.load15 || 0}\n`;
  prompt += `Network: ${networkStats.network_in_kbps || 0} KB/s in, ${networkStats.network_out_kbps || 0} KB/s out\n`;
  prompt += `IP: ${metrics.ip_address || 'unknown'}\n`;
  if (Array.isArray(filesystems) && filesystems.length > 0) {
    prompt += `Filesystems:\n`;
    for (const fs of filesystems){
      const usedPercent = fs.total > 0 ? Math.round(fs.used / fs.total * 100) : 0;
      prompt += `  ${fs.mountpoint}: ${usedPercent}% used (${Math.round(fs.used / 1024 / 1024 / 1024)}GB/${Math.round(fs.total / 1024 / 1024 / 1024)}GB)\n`;
    }
  }
  if (Array.isArray(blockDevices) && blockDevices.length > 0) {
    prompt += `Block Devices:\n`;
    for (const dev of blockDevices){
      prompt += `  ${dev.name}: ${Math.round(dev.size / 1024 / 1024 / 1024)}GB\n`;
    }
  }
  return prompt;
}
// Call TensorZero Core API directly
async function callTensorZeroCore(messages) {
  const tensorZeroBaseUrl = Deno.env.get('TENSORZERO_API_URL') || 'https://tensorzero-api.nannyai.dev';
  const tensorZeroUrl = `${tensorZeroBaseUrl}/openai/v1/chat/completions`;
  const payload = {
    model: 'tensorzero::function_name::diagnose_and_heal_application',
    messages: messages
  };
  console.log('Calling TensorZero Core directly:', tensorZeroUrl);
  
  const tensorZeroApiKey = Deno.env.get('TENSORZERO_API_KEY');
  const headers = {
    'Content-Type': 'application/json'
  };
  
  if (tensorZeroApiKey) {
    headers['Authorization'] = `Bearer ${tensorZeroApiKey}`;
  }
  
  const response = await fetch(tensorZeroUrl, {
    method: 'POST',
    headers: headers,
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`TensorZero Core API error: ${response.status} ${errorText}`);
  }
  return await response.json();
}
// Send task to agent via WebSocket
async function sendTaskToAgentWebSocket(agentId, task) {
  // This would typically call the WebSocket handler function
  // For now, we'll use a direct HTTP call to the WebSocket handler's task sender
  try {
    // Check if agent is connected via WebSocket
    const { data: agent } = await supabase.from('agents').select('websocket_connected').eq('id', agentId).single();
    if (!agent?.websocket_connected) {
      console.log(`Agent ${agentId} is not connected via WebSocket`);
      return false;
    }
    // Since we can't directly access the WebSocket connections from here,
    // we'll create a pending investigation and let the agent pick it up
    return true;
  } catch (error) {
    console.error(`Failed to send task to agent ${agentId}:`, error);
    return false;
  }
}
// Wait for agent to complete investigation via WebSocket
async function waitForAgentCompletion(pendingId, maxWaitSeconds = 120) {
  const startTime = Date.now();
  const pollInterval = 2000; // Poll every 2 seconds
  console.log(`Waiting for agent to complete investigation ${pendingId} via WebSocket...`);
  while(Date.now() - startTime < maxWaitSeconds * 1000){
    const { data: pending } = await supabase.from('pending_investigations').select('*').eq('id', pendingId).single();
    if (!pending) {
      throw new Error('Pending investigation not found');
    }
    if (pending.status === 'completed') {
      console.log('Agent completed investigation via WebSocket!');
      return pending;
    }
    if (pending.status === 'failed') {
      throw new Error(`Agent investigation failed: ${pending.error_message}`);
    }
    // Wait before next poll
    await new Promise((resolve)=>setTimeout(resolve, pollInterval));
  }
  throw new Error(`Investigation timed out after ${maxWaitSeconds} seconds`);
}
// Continue TensorZero conversation with command results
async function continueConversationWithResults(initialMessages, initialResponse, commandResults) {
  const conversationMessages = [
    ...initialMessages,
    {
      role: "assistant",
      content: initialResponse
    },
    {
      role: "user",
      content: `Command execution results: ${JSON.stringify(commandResults)}\n\nPlease analyze these results and provide recommendations or next steps.`
    }
  ];
  return await callTensorZeroCore(conversationMessages);
}
// Validate JWT token
async function validateToken(req: Request) {
  const authHeader = req.headers.get('Authorization');
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return { user: null, error: 'Missing or invalid authorization header' };
  }

  const token = authHeader.substring(7);
  const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';
  
  try {
    const response = await fetch(`${SUPABASE_URL}/auth/v1/user`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'apikey': Deno.env.get('SUPABASE_ANON_KEY') || ''
      }
    });

    if (!response.ok) {
      return { user: null, error: 'Invalid token' };
    }

    const user = await response.json();
    return { user, error: null };
  } catch (error) {
    return { user: null, error: 'Token validation failed' };
  }
}

// GET - List investigations for user's agents
async function handleGetInvestigations(req: Request) {
  const { user, error } = await validateToken(req);
  
  if (error || !user) {
    return new Response(JSON.stringify({ error: error || 'Unauthorized' }), {
      status: 401,
      headers: corsHeaders
    });
  }

  const url = new URL(req.url);
  const page = parseInt(url.searchParams.get('page') || '1');
  const limit = parseInt(url.searchParams.get('limit') || '10');
  const status = url.searchParams.get('status'); // Filter by status if provided
  const agentId = url.searchParams.get('agent_id'); // Filter by specific agent if provided
  const withEpisodes = url.searchParams.get('with_episodes') === 'true'; // Only show investigations with ClickHouse data
  
  const offset = (page - 1) * limit;

  try {
    // First, get all agents owned by this user
    const { data: userAgents, error: agentsError } = await supabase
      .from('agents')
      .select('id')
      .eq('owner', user.id);

    if (agentsError) {
      throw agentsError;
    }

    if (!userAgents || userAgents.length === 0) {
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
        headers: corsHeaders
      });
    }

    const agentIds = userAgents.map(a => a.id);

    // Build query for investigations
    let query = supabase
      .from('investigations')
      .select('*, agents(id, name, status)', { count: 'exact' })
      .in('agent_id', agentIds)
      .order('created_at', { ascending: false });

    // Apply filters
    if (status) {
      query = query.eq('status', status);
    }
    
    if (agentId && agentIds.includes(agentId)) {
      query = query.eq('agent_id', agentId);
    }

    // Filter by episodes if requested
    if (withEpisodes) {
      query = query.not('episode_id', 'is', null);
    }

    // Get total count
    const { count } = await query;

    // Get paginated results
    const { data: investigations, error: invError } = await query
      .range(offset, offset + limit - 1);

    if (invError) {
      throw invError;
    }

    // Structure each investigation properly
    const structuredInvestigations = await Promise.all(
      (investigations || []).map(async (inv) => {
        // Only fetch inference count if episode_id exists
        let inferenceCount = 0;
        if (inv.episode_id) {
          const rawInferences = await fetchClickHouseDataByEpisode(inv.episode_id);
          inferenceCount = rawInferences?.data?.length || 0;
        }

        return {
          id: inv.id,
          investigation_id: inv.investigation_id,
          issue: inv.issue,
          priority: inv.priority,
          status: inv.status,
          episode_id: inv.episode_id,
          inference_count: inferenceCount,
          initiated_by: inv.initiated_by,
          initiated_at: inv.initiated_at,
          completed_at: inv.completed_at,
          created_at: inv.created_at,
          updated_at: inv.updated_at,
          agent: inv.agents ? {
            id: inv.agents.id,
            name: inv.agents.name,
            status: inv.agents.status
          } : null
        };
      })
    );

    const totalPages = count ? Math.ceil(count / limit) : 0;

    return new Response(JSON.stringify({
      investigations: structuredInvestigations,
      pagination: {
        page,
        limit,
        total: count || 0,
        total_pages: totalPages,
        has_next: page < totalPages,
        has_prev: page > 1
      },
      filters: {
        status: status || 'all',
        agent_id: agentId || 'all',
        with_episodes: withEpisodes
      }
    }), {
      status: 200,
      headers: corsHeaders
    });

  } catch (error) {
    console.error('Error fetching investigations:', error);
    return new Response(JSON.stringify({
      error: 'Failed to fetch investigations',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: corsHeaders
    });
  }
}

// GET /investigation/:investigation_id - Get investigation details with ClickHouse data
async function handleGetInvestigationDetails(req: Request, investigationId: string) {
  const { user, error } = await validateToken(req);
  
  if (error || !user) {
    return new Response(JSON.stringify({ error: error || 'Unauthorized' }), {
      status: 401,
      headers: corsHeaders
    });
  }

  try {
    // Fetch investigation from PostgreSQL
    const { data: investigation, error: invError } = await supabase
      .from('investigations')
      .select('*, agents(id, name, status, owner)')
      .eq('investigation_id', investigationId)
      .single();

    if (invError || !investigation) {
      console.error('Investigation not found:', investigationId);
      return new Response(JSON.stringify({
        error: 'Investigation not found',
        details: invError?.message
      }), {
        status: 404,
        headers: corsHeaders
      });
    }

    // Verify user owns the agent
    if (investigation.agents?.owner !== user.id) {
      return new Response(JSON.stringify({
        error: 'Unauthorized to access this investigation'
      }), {
        status: 403,
        headers: corsHeaders
      });
    }

    // Fetch ClickHouse inference summary if episode_id exists (only IDs and metadata)
    let inferences = null;
    if (investigation.episode_id) {
      const rawInferences = await fetchClickHouseDataByEpisode(investigation.episode_id);
      // Structure the inference summary properly
      if (rawInferences?.data) {
        inferences = rawInferences.data.map((row: any) => ({
          id: row[0],
          function_name: row[1],
          variant_name: row[2],
          timestamp: row[3],
          processing_time_ms: row[4]
        }));
      }
    }

    // Return combined data with structured response
    return new Response(JSON.stringify({
      investigation: {
        ...investigation,
        inferences: inferences,
        inference_count: inferences?.length || 0
      }
    }), {
      status: 200,
      headers: corsHeaders
    });

  } catch (error) {
    console.error('Error fetching investigation details:', error);
    return new Response(JSON.stringify({
      error: 'Failed to fetch investigation details',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: corsHeaders
    });
  }
}

// GET /inference/:inference_id - Get individual inference details with ModelInference and Feedback
async function handleGetInferenceDetails(req: Request, inferenceId: string) {
  const { user, error } = await validateToken(req);
  
  if (error || !user) {
    return new Response(JSON.stringify({ error: error || 'Unauthorized' }), {
      status: 401,
      headers: corsHeaders
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
        headers: corsHeaders
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
    const structuredFeedback = feedbackData?.data?.map(fbRow => ({
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
      headers: corsHeaders
    });
  } catch (error) {
    console.error('Error fetching inference details:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: corsHeaders
    });
  }
}

Deno.serve(async (req)=>{
  // Handle CORS
  if (req.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: corsHeaders
    });
  }

  // Handle GET requests
  if (req.method === 'GET') {
    const url = new URL(req.url);
    
    // Parse URL parameters - check for investigation ID in query params or path
    const investigationId = url.searchParams.get('investigation_id');
    const inferenceId = url.searchParams.get('inference_id');
    
    if (investigationId) {
      // GET ?investigation_id=xxx - Get specific investigation with ClickHouse data
      return handleGetInvestigationDetails(req, investigationId);
    }
    
    if (inferenceId) {
      // GET ?inference_id=xxx - Get specific inference details
      return handleGetInferenceDetails(req, inferenceId);
    }
    
    // Try to parse from path: /investigation/xxx or /inference/xxx
    const pathParts = url.pathname.split('/').filter(p => p);
    const lastPart = pathParts[pathParts.length - 1];
    const secondLastPart = pathParts.length > 1 ? pathParts[pathParts.length - 2] : null;
    
    // Check for /inference/{uuid}
    if (secondLastPart === 'inference' || (lastPart && lastPart.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i))) {
      return handleGetInferenceDetails(req, lastPart);
    }
    
    // Check for /investigation/{INV-xxx}
    if (lastPart && lastPart.startsWith('INV-')) {
      return handleGetInvestigationDetails(req, lastPart);
    }
    
    // GET / - List investigations
    return handleGetInvestigations(req);
  }
  
  if (req.method !== 'POST') {
    return new Response(JSON.stringify({
      error: 'Method not allowed'
    }), {
      status: 405,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
  try {
    const body = await req.json();
    const { agent_id, issue, priority = 'medium', initiated_by } = body;
    if (!agent_id || !issue) {
      return new Response(JSON.stringify({
        error: 'agent_id and issue are required'
      }), {
        status: 400,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    console.log(`Starting WebSocket investigation for agent ${agent_id}: ${issue}`);
    // 1. Check if agent is connected via WebSocket
    const { data: agent } = await supabase.from('agents').select('*, websocket_connected').eq('id', agent_id).single();
    if (!agent) {
      return new Response(JSON.stringify({
        error: 'Agent not found'
      }), {
        status: 404,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    if (!agent.websocket_connected) {
      return new Response(JSON.stringify({
        error: 'Agent not connected via WebSocket',
        details: 'Agent must be online and connected to receive investigations'
      }), {
        status: 503,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // 2. Fetch agent metrics from database
    const { data: agentMetrics, error: metricsError } = await supabase.from('agent_metrics').select('*').eq('agent_id', agent_id).order('recorded_at', {
      ascending: false
    }).limit(1).single();
    if (metricsError || !agentMetrics) {
      return new Response(JSON.stringify({
        error: 'Agent metrics not found',
        details: metricsError?.message
      }), {
        status: 404,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // 3. Create investigation record
    const investigationId = crypto.randomUUID();
    const { error: insertError } = await supabase.from('investigations').insert({
      investigation_id: investigationId,
      agent_id: agent_id,
      issue: issue,
      priority: priority,
      status: 'active',
      initiated_by: initiated_by || 'backend',
      initiated_at: new Date().toISOString(),
      metadata: {
        agent_metrics_snapshot: agentMetrics,
        initial_issue: issue,
        communication_method: 'websocket'
      }
    });
    if (insertError) {
      console.error('Failed to create investigation record:', insertError);
      return new Response(JSON.stringify({
        error: 'Failed to create investigation record',
        details: insertError.message
      }), {
        status: 500,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // 4. Format system info and send to TensorZero
    const systemInfoPrompt = formatSystemInfoForPrompt(agentMetrics);
    const initialPrompt = systemInfoPrompt + "\n" + issue;
    const initialMessages = [
      {
        role: "user",
        content: initialPrompt
      }
    ];
    console.log('Sending initial prompt to TensorZero Core');
    const tensorZeroResponse = await callTensorZeroCore(initialMessages);
    if (!tensorZeroResponse.choices || tensorZeroResponse.choices.length === 0) {
      throw new Error('No response from TensorZero Core');
    }
    const aiResponse = tensorZeroResponse.choices[0].message.content;
    console.log('TensorZero Core initial response received');
    // 5. Parse diagnostic response
    let diagnosticResponse = null;
    let finalStatus = 'completed';
    let agentResults = null;
    let continuationResponse = null;
    try {
      diagnosticResponse = JSON.parse(aiResponse);
      if (diagnosticResponse.response_type === 'diagnostic') {
        console.log('Got diagnostic response, sending to agent via WebSocket...');
        // 6. Create pending investigation for WebSocket communication
        const { data: pending, error: pendingError } = await supabase.from('pending_investigations').insert({
          investigation_id: investigationId,
          agent_id: agent_id,
          diagnostic_payload: diagnosticResponse,
          episode_id: tensorZeroResponse.episode_id,
          status: 'pending'
        }).select().single();
        if (pendingError || !pending) {
          throw new Error(`Failed to create pending investigation: ${pendingError?.message}`);
        }
        console.log(`Created pending investigation ${pending.id}, agent will pick it up via WebSocket`);
        // 7. Wait for agent to complete via WebSocket
        const completedInvestigation = await waitForAgentCompletion(pending.id);
        agentResults = completedInvestigation.command_results;
        console.log('Agent completed investigation via WebSocket, continuing TensorZero conversation...');
        // 8. Continue TensorZero conversation with results
        if (agentResults) {
          continuationResponse = await continueConversationWithResults(initialMessages, aiResponse, agentResults);
          console.log('TensorZero continuation completed');
          finalStatus = 'completed_with_analysis';
        }
      }
    } catch (e) {
      console.error('Error in WebSocket diagnostic flow:', e);
      finalStatus = 'failed';
    }
    // 9. Update investigation with final results
    await supabase.from('investigations').update({
      tensorzero_response: aiResponse,
      episode_id: tensorZeroResponse.episode_id || null,
      status: finalStatus,
      updated_at: new Date().toISOString(),
      metadata: {
        agent_metrics_snapshot: agentMetrics,
        initial_issue: issue,
        agent_results: agentResults,
        continuation_response: continuationResponse?.choices?.[0]?.message?.content,
        communication_method: 'websocket',
        full_conversation: {
          initial_response: aiResponse,
          agent_execution: agentResults,
          final_analysis: continuationResponse?.choices?.[0]?.message?.content
        }
      }
    }).eq('investigation_id', investigationId);
    return new Response(JSON.stringify({
      success: true,
      investigation_id: investigationId,
      agent_id: agent_id,
      status: finalStatus,
      communication_method: 'websocket',
      agent_connected: true,
      initial_tensorzero_response: aiResponse,
      diagnostic_response: diagnosticResponse,
      agent_execution_results: agentResults,
      continuation_response: continuationResponse?.choices?.[0]?.message?.content,
      episode_id: tensorZeroResponse.episode_id,
      message: `WebSocket investigation ${investigationId} completed for agent ${agent_id}`,
      full_flow_completed: !!(diagnosticResponse && agentResults && continuationResponse)
    }), {
      status: 200,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('WebSocket investigation coordinator error:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      details: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
});