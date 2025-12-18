import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";
const supabase = createClient(Deno.env.get('SUPABASE_URL') ?? '', Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '');
async function validateAgentToken(authHeader) {
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return {
      valid: false,
      error: 'Missing or invalid Authorization header'
    };
  }
  const token = authHeader.substring(7);
  try {
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (error || !user) {
      return {
        valid: false,
        error: 'Invalid or expired token'
      };
    }
    const agentId = user.id;
    if (!agentId) {
      return {
        valid: false,
        error: 'Agent ID not found in token'
      };
    }
    return {
      valid: true,
      agentId,
      user
    };
  } catch (err) {
    return {
      valid: false,
      error: `Token validation failed: ${err.message}`
    };
  }
}
Deno.serve(async (req)=>{
  const url = new URL(req.url);
  console.log(`${req.method} ${url.pathname}`);
  if (req.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, PATCH, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization'
      }
    });
  }
  const authHeader = req.headers.get('Authorization') || '';
  const tokenValidation = await validateAgentToken(authHeader);
  if (!tokenValidation.valid || !tokenValidation.agentId) {
    console.log(`Auth failed: ${tokenValidation.error}`);
    return new Response(JSON.stringify({
      error: tokenValidation.error
    }), {
      status: 401,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  }
  const agentId = tokenValidation.agentId;
  console.log(`Authenticated agent: ${agentId}`);
  try {
    // POST /connection-status - Update websocket connection status
    if (req.method === 'POST' && url.pathname.endsWith('/connection-status')) {
      const body = await req.json();
      const connected = body.connected === true;
      const updateData = {
        websocket_connected: connected
      };
      if (connected) {
        updateData.websocket_connected_at = new Date().toISOString();
      } else {
        updateData.websocket_disconnected_at = new Date().toISOString();
      }
      const { error } = await supabase.from('agents').update(updateData).eq('id', agentId);
      if (error) throw new Error(`Failed to update connection status: ${error.message}`);
      console.log(`Updated agent ${agentId} connection status to ${connected}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // PATCH /update-investigation - Update investigation status and episode_id
    if (req.method === 'PATCH' && url.pathname.endsWith('/update-investigation')) {
      const body = await req.json();
      const investigationId = body.investigation_id;
      const status = body.status;
      const episodeId = body.episode_id;
      if (!investigationId || !status) {
        return new Response(JSON.stringify({
          error: 'Missing investigation_id or status'
        }), {
          status: 400,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      // Verify the investigation belongs to this agent
      const { data: investigation } = await supabase.from('investigations').select('agent_id, initiated_by').eq('investigation_id', investigationId).single();
      if (!investigation || investigation.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Investigation not found or access denied'
        }), {
          status: 403,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const updateData = {
        status,
        updated_at: new Date().toISOString()
      };
      if (episodeId) updateData.episode_id = episodeId;
      const { error } = await supabase.from('investigations').update(updateData).eq('investigation_id', investigationId);
      if (error) throw new Error(`Failed to update investigation: ${error.message}`);
      console.log(`Updated investigation ${investigationId} status to ${status}${episodeId ? ` with episode_id ${episodeId}` : ''}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // POST /patch-execution-status - Update patch execution status
    if (req.method === 'POST' && url.pathname.endsWith('/patch-execution-status')) {
      const body = await req.json();
      const executionId = body.execution_id;
      const status = body.status;
      const errorMessage = body.error_message;
      const startedAt = body.started_at;
      const completedAt = body.completed_at;
      if (!executionId || !status) {
        return new Response(JSON.stringify({
          error: 'Missing execution_id or status'
        }), {
          status: 400,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      // Verify the execution belongs to this agent
      const { data: execution } = await supabase.from('patch_executions').select('agent_id').eq('id', executionId).single();
      if (!execution || execution.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Patch execution not found or access denied'
        }), {
          status: 403,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const updateData = {
        status
      };
      if (startedAt) updateData.started_at = startedAt;
      if (completedAt) updateData.completed_at = completedAt;
      if (errorMessage) updateData.error_message = errorMessage;
      const { error } = await supabase.from('patch_executions').update(updateData).eq('id', executionId);
      if (error) throw new Error(`Failed to update patch status: ${error.message}`);
      console.log(`Updated patch execution ${executionId} status to ${status}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // POST /patch-execution-logs - Update patch execution logs reference
    if (req.method === 'POST' && url.pathname.endsWith('/patch-execution-logs')) {
      const body = await req.json();
      const executionId = body.execution_id;
      const logsPath = body.logs_path;
      if (!executionId || !logsPath) {
        return new Response(JSON.stringify({
          error: 'Missing execution_id or logs_path'
        }), {
          status: 400,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      // Verify the execution belongs to this agent
      const { data: execution } = await supabase.from('patch_executions').select('agent_id').eq('id', executionId).single();
      if (!execution || execution.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Patch execution not found or access denied'
        }), {
          status: 403,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const { error } = await supabase.from('patch_executions').update({
        logs_path: logsPath
      }).eq('id', executionId);
      if (error) throw new Error(`Failed to update logs path: ${error.message}`);
      console.log(`Updated patch execution ${executionId} logs path to ${logsPath}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // GET /patch-scripts/:id - Get patch script info
    if (req.method === 'GET' && url.pathname.includes('/patch-scripts/')) {
      const pathSegments = url.pathname.split('/');
      const scriptId = pathSegments[pathSegments.length - 1];
      const { data: script, error } = await supabase.from('patch_scripts').select('id, name, script_storage_path, os_family, package_manager').eq('id', scriptId).single();
      if (error || !script) {
        return new Response(JSON.stringify({
          error: 'Script not found'
        }), {
          status: 404,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      return new Response(JSON.stringify(script), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // GET /pending-investigations
    if (req.method === 'GET' && url.pathname.endsWith('/pending-investigations')) {
      const { data, error } = await supabase.from('pending_investigations').select('*').eq('agent_id', agentId).eq('status', 'pending').order('created_at', {
        ascending: false
      });
      if (error) throw new Error(`Database error: ${error.message}`);
      console.log(`Found ${data?.length || 0} pending investigations`);
      return new Response(JSON.stringify(data || []), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // GET /patch-tasks
    if (req.method === 'GET' && url.pathname.endsWith('/patch-tasks')) {
      const { data, error } = await supabase.from('patch_tasks').select('*').eq('agent_id', agentId).eq('status', 'pending').order('created_at', {
        ascending: false
      });
      if (error) throw new Error(`Database error: ${error.message}`);
      console.log(`Found ${data?.length || 0} pending patch tasks`);
      return new Response(JSON.stringify(data || []), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // GET /patch-executions - Get pending patch executions
    if (req.method === 'GET' && url.pathname.endsWith('/patch-executions')) {
      const { data, error } = await supabase.from('patch_executions').select('*').eq('agent_id', agentId).eq('status', 'pending').order('started_at', {
        ascending: false
      });
      if (error) throw new Error(`Database error: ${error.message}`);
      console.log(`Found ${data?.length || 0} pending patch executions`);
      return new Response(JSON.stringify(data || []), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // PATCH /pending-investigations/:id
    if (req.method === 'PATCH' && url.pathname.includes('/pending-investigations/')) {
      const pathSegments = url.pathname.split('/');
      const investigationId = pathSegments[pathSegments.length - 1];
      const updates = await req.json();
      const { data: investigation } = await supabase.from('pending_investigations').select('agent_id').eq('id', investigationId).single();
      if (!investigation || investigation.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Investigation not found or access denied'
        }), {
          status: 404,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const { error } = await supabase.from('pending_investigations').update(updates).eq('id', investigationId);
      if (error) throw new Error(`Update failed: ${error.message}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // PATCH /patch-tasks/:id
    if (req.method === 'PATCH' && url.pathname.includes('/patch-tasks/')) {
      const pathSegments = url.pathname.split('/');
      const taskId = pathSegments[pathSegments.length - 1];
      const updates = await req.json();
      const { data: task } = await supabase.from('patch_tasks').select('agent_id').eq('id', taskId).single();
      if (!task || task.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Patch task not found or access denied'
        }), {
          status: 404,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const { error } = await supabase.from('patch_tasks').update(updates).eq('id', taskId);
      if (error) throw new Error(`Update failed: ${error.message}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // PATCH /patch-executions/:id - Update patch execution status
    if (req.method === 'PATCH' && url.pathname.includes('/patch-executions/')) {
      const pathSegments = url.pathname.split('/');
      const executionId = pathSegments[pathSegments.length - 1];
      const updates = await req.json();
      console.log(`Updating patch execution ${executionId}`);
      const { data: execution } = await supabase.from('patch_executions').select('agent_id').eq('id', executionId).single();
      if (!execution || execution.agent_id !== agentId) {
        return new Response(JSON.stringify({
          error: 'Patch execution not found or access denied'
        }), {
          status: 404,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      const { error } = await supabase.from('patch_executions').update(updates).eq('id', executionId);
      if (error) throw new Error(`Update failed: ${error.message}`);
      console.log(`Successfully updated patch execution ${executionId}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    return new Response(JSON.stringify({
      error: 'Not found',
      path: url.pathname
    }), {
      status: 404,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Proxy error:', error);
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
