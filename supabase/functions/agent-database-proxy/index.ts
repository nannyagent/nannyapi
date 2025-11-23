import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";
const supabase = createClient(Deno.env.get('SUPABASE_URL') ?? '', Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '');
// Validate agent JWT token
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
    // Extract agent_id from JWT subject
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
  console.log(`${req.method} ${url.pathname} - Full URL: ${req.url}`);
  // Handle CORS
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
  // Validate authentication
  const authHeader = req.headers.get('Authorization') || '';
  console.log(`Auth header present: ${authHeader ? 'Yes' : 'No'}`);
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
    // Check if it's the pending-investigations endpoint
    if (req.method === 'GET' && url.pathname.endsWith('/pending-investigations')) {
      console.log(`Fetching pending investigations for agent ${agentId}`);
      // Get pending investigations for this agent
      const { data, error } = await supabase.from('pending_investigations').select('*').eq('agent_id', agentId).eq('status', 'pending').order('created_at', {
        ascending: false
      });
      if (error) {
        console.error('Database error:', error);
        throw new Error(`Database error: ${error.message}`);
      }
      console.log(`Found ${data?.length || 0} pending investigations`);
      return new Response(JSON.stringify(data || []), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    // Check if it's a PATCH to update investigation status
    if (req.method === 'PATCH' && url.pathname.includes('/pending-investigations/')) {
      const pathSegments = url.pathname.split('/');
      const investigationId = pathSegments[pathSegments.length - 1];
      console.log(`Updating investigation ${investigationId} for agent ${agentId}`);
      const updates = await req.json();
      console.log('Updates:', updates);
      // Verify this investigation belongs to this agent
      const { data: investigation } = await supabase.from('pending_investigations').select('agent_id').eq('id', investigationId).single();
      if (!investigation || investigation.agent_id !== agentId) {
        console.log('Investigation not found or access denied');
        return new Response(JSON.stringify({
          error: 'Investigation not found or access denied'
        }), {
          status: 404,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
      // Update the investigation
      const { error } = await supabase.from('pending_investigations').update(updates).eq('id', investigationId);
      if (error) {
        console.error('Update error:', error);
        throw new Error(`Update failed: ${error.message}`);
      }
      console.log(`Successfully updated investigation ${investigationId}`);
      return new Response(JSON.stringify({
        success: true
      }), {
        status: 200,
        headers: {
          'Content-Type': 'application/json'
        }
      });
    }
    console.log(`Route not found. Method: ${req.method}, Path: ${url.pathname}`);
    return new Response(JSON.stringify({
      error: 'Not found',
      method: req.method,
      path: url.pathname,
      fullUrl: req.url
    }), {
      status: 404,
      headers: {
        'Content-Type': 'application/json'
      }
    });
  } catch (error) {
    console.error('Agent database proxy error:', error);
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
