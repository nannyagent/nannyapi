import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";

const supabase = createClient(
  Deno.env.get('SUPABASE_URL') ?? '',
  Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? ''
);

const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type, Authorization, apikey, x-client-info'
};

Deno.serve(async (req: Request) => {
  if (req.method === 'OPTIONS') {
    return new Response(null, { status: 200, headers: corsHeaders });
  }

  if (req.method !== 'POST') {
    return new Response(
      JSON.stringify({ error: 'Method not allowed' }),
      { status: 405, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    );
  }

  try {
    const body = await req.json();
    const agent_id = body.agent_id;
    const execution_type = body.execution_type || 'apply'; // 'dry_run' or 'apply'

    if (!agent_id) {
      return new Response(
        JSON.stringify({ error: 'agent_id required' }),
        { status: 400, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    if (!['dry_run', 'apply'].includes(execution_type)) {
      return new Response(
        JSON.stringify({ error: 'execution_type must be "dry_run" or "apply"' }),
        { status: 400, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    // Verify agent exists and is connected
    const { data: agent, error: agentError } = await supabase
      .from('agents')
      .select('id, websocket_connected')
      .eq('id', agent_id)
      .single();

    if (agentError || !agent) {
      return new Response(
        JSON.stringify({ error: 'Agent not found' }),
        { status: 404, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    if (!agent.websocket_connected) {
      return new Response(
        JSON.stringify({ error: 'Agent not connected' }),
        { status: 503, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    // Get agent's OS info from agent_metrics (use recorded_at, not created_at)
    const { data: metrics, error: metricsError } = await supabase
      .from('agent_metrics')
      .select('os_info')
      .eq('agent_id', agent_id)
      .order('recorded_at', { ascending: false })
      .limit(1)
      .single();

    if (metricsError || !metrics || !metrics.os_info) {
      return new Response(
        JSON.stringify({ error: 'Agent OS information not found' }),
        { status: 404, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    const os_family = metrics.os_info.platform_family || metrics.os_info.platform;
    if (!os_family) {
      return new Response(
        JSON.stringify({ error: 'Agent OS family not detected' }),
        { status: 400, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    // Find all patch scripts matching agent's OS family
    const { data: patchScripts, error: scriptsError, count } = await supabase
      .from('patch_scripts')
      .select('id, name, os_family, os_platform, script_storage_path', { count: 'exact' })
      .eq('os_family', os_family)
      .eq('is_active', true);

    if (scriptsError || !patchScripts || patchScripts.length === 0) {
      return new Response(
        JSON.stringify({ error: `No patch scripts found for OS family: ${os_family}` }),
        { status: 404, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    // Create patch_executions records for each matching patch
    // Agent will receive via websocket and execute
    // For dry_run, command will be '--dry-run'
    // For apply, command will be ''
    const execution_ids: string[] = [];
    const patch_names: string[] = [];

    for (const patchScript of patchScripts) {
      // Set command based on execution_type
      const command = execution_type === 'dry_run' ? '--dry-run' : '';
      
      const { data: execution, error: execError } = await supabase
        .from('patch_executions')
        .insert({
          agent_id: agent_id,
          script_id: patchScript.id,
          execution_type: execution_type,
          status: 'pending',
          command: command,
          triggered_by: 'web_ui'
        })
        .select('id')
        .single();

      if (execError || !execution) {
        console.error(`Failed to create patch execution for ${patchScript.name}:`, execError?.message);
        continue;
      }

      execution_ids.push(execution.id);
      patch_names.push(patchScript.name);
      console.log(`Patch queued: ${execution.id} - ${patchScript.name} for agent ${agent_id} (${execution_type})`);
    }

    if (execution_ids.length === 0) {
      return new Response(
        JSON.stringify({ error: 'Failed to queue any patches' }),
        { status: 500, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      );
    }

    return new Response(
      JSON.stringify({
        success: true,
        execution_id: execution_ids[0],
        execution_ids: execution_ids,
        agent_id: agent_id,
        execution_type: execution_type,
        status: 'pending',
        patch_names: patch_names,
        total_patches_queued: execution_ids.length,
        message: `Queued ${execution_ids.length} patch(es) for agent (${execution_type})`
      }),
      { status: 200, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    );
  } catch (error) {
    console.error('Error:', error);
    return new Response(
      JSON.stringify({ error: 'Internal server error' }),
      { status: 500, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    );
  }
});
