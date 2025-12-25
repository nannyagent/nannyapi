import "jsr:@supabase/functions-js/edge-runtime.d.ts";

const RATE_LIMIT_WINDOW = 60000; // 1 minute
const RATE_LIMIT_MAX_REQUESTS = 30; // 30 requests per minute
const rateLimitMap = new Map();

// Validate JWT token from Supabase Auth
async function validateAgentToken(authHeader) {
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return {
      valid: false,
      error: 'Missing or invalid Authorization header'
    };
  }

  const token = authHeader.substring(7);
  
  try {
    // Verify JWT token using Supabase Auth
    const { data: { user }, error } = await supabaseClient.auth.getUser(token);
    
    if (error || !user) {
      return {
        valid: false,
        error: 'Invalid or expired token'
      };
    }

    // Extract agent_id from user metadata or user_id
    const agentId = user.user_metadata?.agent_id || user.id;
    
    if (!agentId) {
      return {
        valid: false,
        error: 'Agent ID not found in token'
      };
    }

    return {
      valid: true,
      agentId
    };
  } catch (err) {
    console.error('Token validation error:', err);
    return {
      valid: false,
      error: 'Token validation failed'
    };
  }
}

// Check and update rate limiting
function checkRateLimit(agentId) {
  const now = Date.now();
  const key = `agent_${agentId}`;
  const existing = rateLimitMap.get(key);

  if (!existing || now > existing.resetTime) {
    // Create new rate limit window
    rateLimitMap.set(key, {
      count: 1,
      resetTime: now + RATE_LIMIT_WINDOW
    });
    return {
      allowed: true
    };
  }

  if (existing.count >= RATE_LIMIT_MAX_REQUESTS) {
    return {
      allowed: false,
      resetTime: existing.resetTime
    };
  }

  // Increment counter
  existing.count++;
  rateLimitMap.set(key, existing);
  
  return {
    allowed: true
  };
}

// Sanitize prompts to prevent injection attacks
function sanitizePrompt(content) {
  if (typeof content !== 'string') return '';
  
  // Remove potential system prompt injections and limit length
  const maxLength = 50000;
  let sanitized = content.substring(0, maxLength);
  
  // Remove common prompt injection patterns
  const injectionPatterns = [
    /ignore\s+previous\s+instructions/gi,
    /forget\s+everything\s+above/gi,
    /you\s+are\s+now\s+a\s+different/gi,
    /system\s*:\s*you\s+are/gi
  ];
  
  injectionPatterns.forEach(pattern => {
    sanitized = sanitized.replace(pattern, '[FILTERED]');
  });
  
  return sanitized;
}

// Create Supabase client
const supabaseClient = (() => {
  const supabaseUrl = Deno.env.get('SUPABASE_URL');
  const supabaseAnonKey = Deno.env.get('SUPABASE_ANON_KEY');
  const supabaseServiceRoleKey = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY');
  
  return {
    auth: {
      getUser: async (token) => {
        const response = await fetch(`${supabaseUrl}/auth/v1/user`, {
          headers: {
            'Authorization': `Bearer ${token}`,
            'apikey': supabaseAnonKey
          }
        });
        
        if (!response.ok) {
          return {
            data: { user: null },
            error: { message: 'Invalid token' }
          };
        }
        
        const user = await response.json();
        return {
          data: { user },
          error: null
        };
      }
    },
    from: (table) => ({
      insert: async (data) => {
        const response = await fetch(`${supabaseUrl}/rest/v1/${table}`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${supabaseServiceRoleKey}`,
            'apikey': supabaseServiceRoleKey,
            'Prefer': 'return=representation'
          },
          body: JSON.stringify(data)
        });
        
        if (!response.ok) {
          const error = await response.text();
          return { data: null, error: { message: error } };
        }
        
        const result = await response.json();
        return { data: Array.isArray(result) ? result[0] : result, error: null };
      },
      update: async (data) => ({
        eq: async (column, value) => {
          const response = await fetch(`${supabaseUrl}/rest/v1/${table}?${column}=eq.${value}`, {
            method: 'PATCH',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${supabaseServiceRoleKey}`,
              'apikey': supabaseServiceRoleKey,
              'Prefer': 'return=representation'
            },
            body: JSON.stringify(data)
          });
          
          if (!response.ok) {
            const error = await response.text();
            return { data: null, error: { message: error } };
          }
          
          const result = await response.json();
          return { data: Array.isArray(result) ? result[0] : result, error: null };
        }
      }),
      select: (columns = '*') => ({
        eq: async (column, value) => {
          const response = await fetch(`${supabaseUrl}/rest/v1/${table}?${column}=eq.${value}&select=${columns}`, {
            headers: {
              'Authorization': `Bearer ${supabaseServiceRoleKey}`,
              'apikey': supabaseServiceRoleKey
            }
          });
          
          if (!response.ok) {
            const error = await response.text();
            return { data: null, error: { message: error } };
          }
          
          const result = await response.json();
          return { data: result, error: null };
        },
        single: async () => {
          const response = await fetch(`${supabaseUrl}/rest/v1/${table}?select=${columns}&limit=1`, {
            headers: {
              'Authorization': `Bearer ${supabaseServiceRoleKey}`,
              'apikey': supabaseServiceRoleKey,
              'Accept': 'application/vnd.pgrst.object+json'
            }
          });
          
          if (!response.ok) {
            const error = await response.text();
            return { data: null, error: { message: error } };
          }
          
          const result = await response.json();
          return { data: result, error: null };
        }
      })
    })
  };
})();

// Helper functions for managing investigations

async function createInvestigation(data: {
  agentId: string;
  episodeId: string;
  issue: string;
  initiatedBy: string;
  metadata?: any;
}) {
  const investigationId = `INV-${new Date().toISOString().split('T')[0].replace(/-/g, '')}-${Math.random().toString(36).substring(2, 8).toUpperCase()}`;
  
  const investigation = {
    investigation_id: investigationId,
    agent_id: data.agentId,
    episode_id: data.episodeId,
    issue: data.issue,
    initiated_by: data.initiatedBy,
    status: 'in_progress',
    initiated_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    metadata: data.metadata || {}
  };
  
  const { data: result, error } = await supabaseClient.from('investigations').insert(investigation);
  
  if (error) {
    console.error('Failed to create investigation:', error);
    return null;
  }
  
  console.log(`Created investigation ${investigationId} for episode ${data.episodeId}`);
  return result;
}

async function updateInvestigation(episodeId: string, updates: {
  status?: string;
  holistic_analysis?: string;
  completed_at?: string;
  updated_at?: string;
}) {
  const updateData = {
    ...updates,
    updated_at: new Date().toISOString()
  };
  
  const updateBuilder = await supabaseClient.from('investigations').update(updateData);
  const { data, error } = await updateBuilder.eq('episode_id', episodeId);
  
  if (error) {
    console.error(`Failed to update investigation for episode ${episodeId}:`, error);
    return null;
  }
  
  console.log(`Updated investigation for episode ${episodeId}`, updateData);
  return data;
}

async function getInvestigationByEpisode(episodeId: string) {
  const { data, error } = await supabaseClient.from('investigations').select('*').eq('episode_id', episodeId);
  
  if (error) {
    console.error(`Failed to get investigation for episode ${episodeId}:`, error);
    return null;
  }
  
  return data && data.length > 0 ? data[0] : null;
}

async function completeInvestigation(episodeId: string, resolutionPlan: string) {
  return await updateInvestigation(episodeId, {
    status: 'completed',
    holistic_analysis: resolutionPlan,
    completed_at: new Date().toISOString()
  });
}

Deno.serve(async (req) => {
  // Handle CORS for all requests
  if (req.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization, x-client-info, apikey'
      }
    });
  }

  try {
    // Extract the path from the URL
    const url = new URL(req.url);
    // Expected format: /functions/v1/tensorzero-proxy/[path]
    const pathParts = url.pathname.split('/');
    const functionIndex = pathParts.indexOf('tensorzero-proxy');
    
    if (functionIndex === -1 || functionIndex >= pathParts.length - 1) {
      return new Response(JSON.stringify({
        error: 'Invalid path format. Expected: /functions/v1/tensorzero-proxy/[endpoint]'
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // Extract the endpoint path (everything after tensorzero-proxy)
    const endpointPath = '/' + pathParts.slice(functionIndex + 1).join('/');
    
    // Handle different endpoint types
    if (endpointPath.startsWith('/api/v1/')) {
      return await handleTensorZeroProxyAPI(req, endpointPath, url);
    } else if (endpointPath.startsWith('/openai/v1/')) {
      return await handleOpenAIProxy(req, endpointPath);
    } else {
      return new Response(JSON.stringify({
        error: 'Unsupported endpoint. Supported: /api/v1/*, /openai/v1/*'
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

  } catch (error) {
    console.error('Proxy error:', error);
    return new Response(JSON.stringify({
      error: 'Internal server error',
      message: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }
});

// Handle TensorZero Proxy API endpoints (/api/v1/*)
async function handleTensorZeroProxyAPI(req: Request, endpointPath: string, url: URL) {
  // Validate authentication for all API endpoints
  const authHeader = req.headers.get('Authorization');
  const tokenValidation = await validateAgentToken(authHeader || '');
  
  if (!tokenValidation.valid) {
    return new Response(JSON.stringify({
      error: tokenValidation.error || 'Authentication failed'
    }), {
      status: 401,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }

  const agentId = tokenValidation.agentId;

  // Route to specific API handlers
  if (endpointPath.startsWith('/api/v1/inferences/')) {
    return await handleInferenceAPI(req, endpointPath, url, agentId);
  } else if (endpointPath.startsWith('/api/v1/episodes/')) {
    return await handleEpisodeAPI(req, endpointPath, url, agentId);
  } else if (endpointPath === '/api/v1/health') {
    return await handleHealthAPI(req, agentId);
  } else {
    return new Response(JSON.stringify({
      error: 'API endpoint not found',
      supportedEndpoints: [
        '/api/v1/inferences/{id}',
        '/api/v1/episodes/{id}',
        '/api/v1/episodes/{id}/inferences',
        '/api/v1/health'
      ]
    }), {
      status: 404,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

// Handle inference-related API endpoints
async function handleInferenceAPI(req: Request, endpointPath: string, url: URL, agentId: string) {
  if (req.method !== 'GET') {
    return new Response(JSON.stringify({
      error: 'Method not allowed. Only GET is supported for inference endpoints.'
    }), {
      status: 405,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  // Extract inference ID from path: /api/v1/inferences/{id}
  const pathMatch = endpointPath.match(/^\/api\/v1\/inferences\/([a-f0-9-]{36})$/);
  if (!pathMatch) {
    return new Response(JSON.stringify({
      error: 'Invalid inference ID format. Expected UUID v4/v7.'
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  const inferenceId = pathMatch[1];
  
  // Check rate limiting
  const rateLimit = checkRateLimit(agentId);
  if (!rateLimit.allowed) {
    const resetTimeSeconds = Math.ceil((rateLimit.resetTime! - Date.now()) / 1000);
    return new Response(JSON.stringify({
      error: 'Rate limit exceeded',
      resetTime: resetTimeSeconds
    }), {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*',
        'Retry-After': resetTimeSeconds.toString()
      }
    });
  }

  try {
    // Note: In a real implementation, this would query the TensorZero ClickHouse database
    // For now, we'll provide a framework/placeholder response
    const inferenceData = await retrieveInferenceById(inferenceId, agentId);
    
    return new Response(JSON.stringify(inferenceData), {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  } catch (error) {
    console.error('Error retrieving inference:', error);
    return new Response(JSON.stringify({
      error: 'Failed to retrieve inference',
      message: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }
}

// Handle episode-related API endpoints
async function handleEpisodeAPI(req: Request, endpointPath: string, url: URL, agentId: string) {
  if (req.method !== 'GET') {
    return new Response(JSON.stringify({
      error: 'Method not allowed. Only GET is supported for episode endpoints.'
    }), {
      status: 405,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  // Check for /api/v1/episodes/{id}/inferences
  const inferenceListMatch = endpointPath.match(/^\/api\/v1\/episodes\/([a-f0-9-]{36})\/inferences$/);
  if (inferenceListMatch) {
    const episodeId = inferenceListMatch[1];
    return await handleEpisodeInferences(episodeId, url, agentId);
  }

  // Check for /api/v1/episodes/{id}
  const episodeMatch = endpointPath.match(/^\/api\/v1\/episodes\/([a-f0-9-]{36})$/);
  if (!episodeMatch) {
    return new Response(JSON.stringify({
      error: 'Invalid episode endpoint. Supported: /api/v1/episodes/{id} or /api/v1/episodes/{id}/inferences'
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  const episodeId = episodeMatch[1];
  
  // Check rate limiting
  const rateLimit = checkRateLimit(agentId);
  if (!rateLimit.allowed) {
    const resetTimeSeconds = Math.ceil((rateLimit.resetTime! - Date.now()) / 1000);
    return new Response(JSON.stringify({
      error: 'Rate limit exceeded',
      resetTime: resetTimeSeconds
    }), {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*',
        'Retry-After': resetTimeSeconds.toString()
      }
    });
  }

  try {
    const episodeData = await retrieveEpisodeById(episodeId, agentId);
    
    return new Response(JSON.stringify(episodeData), {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  } catch (error) {
    console.error('Error retrieving episode:', error);
    return new Response(JSON.stringify({
      error: 'Failed to retrieve episode',
      message: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }
}

// Handle episode inferences list
async function handleEpisodeInferences(episodeId: string, url: URL, agentId: string) {
  // Check rate limiting
  const rateLimit = checkRateLimit(agentId);
  if (!rateLimit.allowed) {
    const resetTimeSeconds = Math.ceil((rateLimit.resetTime! - Date.now()) / 1000);
    return new Response(JSON.stringify({
      error: 'Rate limit exceeded',
      resetTime: resetTimeSeconds
    }), {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*',
        'Retry-After': resetTimeSeconds.toString()
      }
    });
  }

  try {
    // Parse query parameters for pagination
    const limit = Math.min(parseInt(url.searchParams.get('limit') || '10'), 100);
    const offset = Math.max(parseInt(url.searchParams.get('offset') || '0'), 0);
    
    const inferencesData = await retrieveEpisodeInferences(episodeId, agentId, limit, offset);
    
    return new Response(JSON.stringify(inferencesData), {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  } catch (error) {
    console.error('Error retrieving episode inferences:', error);
    return new Response(JSON.stringify({
      error: 'Failed to retrieve episode inferences',
      message: error instanceof Error ? error.message : 'Unknown error'
    }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }
}

// Handle health check endpoint
async function handleHealthAPI(req: Request, agentId: string) {
  if (req.method !== 'GET') {
    return new Response(JSON.stringify({
      error: 'Method not allowed. Only GET is supported for health endpoint.'
    }), {
      status: 405,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  return new Response(JSON.stringify({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    agentId: agentId,
    endpoints: {
      chat: '/openai/v1/chat/completions',
      inference: '/api/v1/inferences/{id}',
      episode: '/api/v1/episodes/{id}',
      episodeInferences: '/api/v1/episodes/{id}/inferences'
    },
    version: '1.1.0'
  }), {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*'
    }
  });
}

// Handle OpenAI proxy endpoints (/openai/v1/*)
async function handleOpenAIProxy(req: Request, openaiPath: string) {
  // Validate it's an OpenAI v1 endpoint
  if (!openaiPath.startsWith('/openai/v1/')) {
    return new Response(JSON.stringify({
      error: 'Only OpenAI v1 endpoints are supported'
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  // Validate authentication
  const authHeader = req.headers.get('Authorization');
  const tokenValidation = await validateAgentToken(authHeader || '');
  
  if (!tokenValidation.valid) {
    return new Response(JSON.stringify({
      error: tokenValidation.error || 'Authentication failed'
    }), {
      status: 401,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }

  const agentId = tokenValidation.agentId;

  // Check rate limiting
  const rateLimit = checkRateLimit(agentId);
  if (!rateLimit.allowed) {
    const resetTimeSeconds = Math.ceil((rateLimit.resetTime! - Date.now()) / 1000);
    return new Response(JSON.stringify({
      error: 'Rate limit exceeded',
      resetTime: resetTimeSeconds
    }), {
      status: 429,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*',
        'Retry-After': resetTimeSeconds.toString()
      }
    });
  }

  // Only allow POST requests for now (chat completions)
  if (req.method !== 'POST') {
    return new Response(JSON.stringify({
      error: 'Method not allowed. Only POST is supported for OpenAI endpoints.'
    }), {
      status: 405,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  // Parse and sanitize request body
  const requestBody = await req.json();
  
  // Sanitize messages if present
  if (requestBody.messages && Array.isArray(requestBody.messages)) {
    requestBody.messages = requestBody.messages.map(msg => ({
      ...msg,
      content: typeof msg.content === 'string' ? sanitizePrompt(msg.content) : msg.content
    }));
  }

  // Forward request to TensorZero API
  const tensorZeroBaseUrl = Deno.env.get('TENSORZERO_API_URL') || 'https://tensorzero-api.nannyai.dev';
  const tensorZeroUrl = `${tensorZeroBaseUrl}${openaiPath}`;
  console.log(`Forwarding ${req.method} ${openaiPath} to TensorZero for agent ${agentId}`);
  
  const tensorZeroApiKey = Deno.env.get('TENSORZERO_API_KEY');
  const headers = {
    'Content-Type': 'application/json'
  };
  
  if (tensorZeroApiKey) {
    headers['Authorization'] = `Bearer ${tensorZeroApiKey}`;
  }
  
  const tensorZeroResponse = await fetch(tensorZeroUrl, {
    method: req.method,
    headers: headers,
    body: JSON.stringify(requestBody)
  });

  if (!tensorZeroResponse.ok) {
    const errorText = await tensorZeroResponse.text();
    console.error(`TensorZero API error: ${tensorZeroResponse.status} ${errorText}`);
    
    return new Response(JSON.stringify({
      error: 'TensorZero API request failed',
      status: tensorZeroResponse.status,
      message: errorText
    }), {
      status: tensorZeroResponse.status,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }

  // Get the response from TensorZero
  const tensorZeroData = await tensorZeroResponse.json();

  // Track investigation if this is a chat completion
  try {
    if (tensorZeroData && tensorZeroData.episode_id) {
      const episodeId = tensorZeroData.episode_id;
      
      // Check if this is a new investigation (first message in episode)
      const existingInvestigation = await getInvestigationByEpisode(episodeId);
      
      if (!existingInvestigation) {
        // Create new investigation
        // Priority 1: Get issue from metadata if provided
        let issue = requestBody.metadata?.issue_description || requestBody.metadata?.issue;
        
        // Priority 2: Get from the LAST user message (actual user question, not system prompt)
        if (!issue) {
          const userMessages = requestBody.messages?.filter(m => m.role === 'user') || [];
          const lastUserMessage = userMessages[userMessages.length - 1];
          
          if (lastUserMessage?.content) {
            const content = lastUserMessage.content;
            
            // If content contains "ISSUE DESCRIPTION:", extract only that part
            if (typeof content === 'string' && content.includes('ISSUE DESCRIPTION:')) {
              const issueMatch = content.match(/ISSUE DESCRIPTION:\s*\n(.+?)(?:\n\n|$)/s);
              issue = issueMatch ? issueMatch[1].trim() : content;
            } else {
              issue = content;
            }
          }
        }
        
        // Fallback
        if (!issue) {
          issue = 'Investigation started';
        }
        
        // Determine initiated_by from request metadata or default to 'agent'
        const initiatedBy = requestBody.metadata?.initiated_by || 'agent';
        
        await createInvestigation({
          agentId,
          episodeId,
          issue: typeof issue === 'string' ? issue.substring(0, 500) : String(issue).substring(0, 500), // Limit issue text
          initiatedBy,
          metadata: {
            function_name: tensorZeroData.function_name,
            variant_name: tensorZeroData.variant_name,
            ...requestBody.metadata
          }
        });
      } else {
        // Update existing investigation
        // Check if TensorZero response indicates investigation completion
        let isComplete = false;
        let resolutionPlan = '';
        
        // Check if response_type is at the top level of tensorZeroData
        if (tensorZeroData.response_type === 'resolution' && tensorZeroData.resolution_plan) {
          isComplete = true;
          resolutionPlan = tensorZeroData.resolution_plan;
          console.log(`Investigation ${episodeId} marked as complete - resolution detected at top level`);
        } else {
          // Check in the message content (OpenAI format)
          const assistantMessage = tensorZeroData.choices?.[0]?.message?.content || 
                                   tensorZeroData.content?.[0]?.text ||
                                   '';
          
          if (assistantMessage) {
            try {
              // Try to parse the message as JSON (TensorZero structured output in content)
              const parsedResponse = JSON.parse(assistantMessage);
              if (parsedResponse.response_type === 'resolution' && parsedResponse.resolution_plan) {
                isComplete = true;
                resolutionPlan = parsedResponse.resolution_plan;
                console.log(`Investigation ${episodeId} marked as complete - resolution detected in message content`);
              }
            } catch (e) {
              // Not JSON, check metadata for explicit completion flag
              isComplete = requestBody.metadata?.investigation_complete === true ||
                          requestBody.metadata?.status === 'completed';
              if (isComplete) {
                resolutionPlan = assistantMessage;
                console.log(`Investigation ${episodeId} marked as complete - explicit flag in metadata`);
              }
            }
          }
        }
        
        if (isComplete && resolutionPlan) {
          // Mark investigation as complete with resolution plan
          await completeInvestigation(episodeId, resolutionPlan);
        } else {
          // Just update the timestamp
          await updateInvestigation(episodeId, {
            updated_at: new Date().toISOString()
          });
        }
      }
    }
  } catch (investigationError) {
    // Log error but don't fail the request
    console.error('Failed to track investigation:', investigationError);
  }

  // Return the TensorZero response
  return new Response(JSON.stringify(tensorZeroData), {
    status: 200,
    headers: {
      'Content-Type': 'application/json',
      'Access-Control-Allow-Origin': '*'
    }
  });
}

// Database retrieval functions (placeholder implementations)
// In a real implementation, these would connect to TensorZero's ClickHouse database

async function retrieveInferenceById(inferenceId: string, agentId: string) {
  // TODO: Implement actual ClickHouse query to TensorZero database
  // This is a placeholder that demonstrates the expected response structure
  
  console.log(`Retrieving inference ${inferenceId} for agent ${agentId}`);
  
  // Simulate database lookup delay
  await new Promise(resolve => setTimeout(resolve, 100));
  
  // Check if inference exists (placeholder logic)
  if (!inferenceId.match(/^[a-f0-9-]{36}$/)) {
    throw new Error('Invalid inference ID format');
  }
  
  // Placeholder response based on TensorZero ChatInference table schema
  return {
    id: inferenceId,
    function_name: "chat_assistant",
    variant_name: "gpt-4o-v1",
    episode_id: "01234567-89ab-cdef-0123-456789abcdef",
    input: {
      messages: [
        {
          role: "user",
          content: "Hello, how are you?"
        }
      ]
    },
    output: [
      {
        type: "text",
        text: "I'm doing well, thank you for asking! How can I help you today?"
      }
    ],
    processing_time_ms: 1250,
    timestamp: new Date().toISOString(),
    tags: {
      agent_id: agentId
    },
    model_inferences: [
      {
        id: "fedcba98-7654-3210-fedc-ba9876543210",
        model_name: "gpt-4o",
        model_provider_name: "openai",
        input_tokens: 15,
        output_tokens: 18,
        response_time_ms: 1200
      }
    ],
    _meta: {
      source: "tensorzero-proxy",
      note: "This is a placeholder response. In production, this would query the actual TensorZero ClickHouse database."
    }
  };
}

async function retrieveEpisodeById(episodeId: string, agentId: string) {
  // TODO: Implement actual ClickHouse query to TensorZero database
  
  console.log(`Retrieving episode ${episodeId} for agent ${agentId}`);
  
  // Simulate database lookup delay
  await new Promise(resolve => setTimeout(resolve, 150));
  
  // Check if episode exists (placeholder logic)
  if (!episodeId.match(/^[a-f0-9-]{36}$/)) {
    throw new Error('Invalid episode ID format');
  }
  
  // Placeholder response - episodes don't have their own table in TensorZero,
  // but we can aggregate data from inferences with the same episode_id
  return {
    id: episodeId,
    inference_count: 3,
    first_inference_time: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    last_inference_time: new Date().toISOString(),
    functions_used: ["chat_assistant"],
    tags: {
      agent_id: agentId,
      session_type: "conversation"
    },
    total_processing_time_ms: 3750,
    total_input_tokens: 45,
    total_output_tokens: 52,
    _meta: {
      source: "tensorzero-proxy",
      note: "Episode data aggregated from related inferences. In production, this would query the actual TensorZero ClickHouse database."
    }
  };
}

async function retrieveEpisodeInferences(episodeId: string, agentId: string, limit: number, offset: number) {
  // TODO: Implement actual ClickHouse query to TensorZero database
  
  console.log(`Retrieving inferences for episode ${episodeId}, limit=${limit}, offset=${offset}, agent=${agentId}`);
  
  // Simulate database lookup delay
  await new Promise(resolve => setTimeout(resolve, 200));
  
  // Check if episode exists (placeholder logic)
  if (!episodeId.match(/^[a-f0-9-]{36}$/)) {
    throw new Error('Invalid episode ID format');
  }
  
  // Placeholder response - list of inferences for the episode
  const totalInferences = 3;
  const inferences = [];
  
  for (let i = offset; i < Math.min(offset + limit, totalInferences); i++) {
    inferences.push({
      id: `inference-${episodeId}-${i.toString().padStart(3, '0')}`,
      function_name: "chat_assistant",
      variant_name: "gpt-4o-v1",
      episode_id: episodeId,
      processing_time_ms: 1000 + (i * 250),
      timestamp: new Date(Date.now() - (totalInferences - i) * 300000).toISOString(), // 5 min intervals
      tags: {
        agent_id: agentId
      }
    });
  }
  
  return {
    episode_id: episodeId,
    total_count: totalInferences,
    limit: limit,
    offset: offset,
    inferences: inferences,
    _meta: {
      source: "tensorzero-proxy",
      note: "This is a placeholder response. In production, this would query the actual TensorZero ClickHouse database."
    }
  };
}