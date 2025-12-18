import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";

const supabase = createClient(
  Deno.env.get("SUPABASE_URL") ?? "",
  Deno.env.get("SUPABASE_SERVICE_ROLE_KEY") ?? ""
);

async function validateAgentToken(authHeader) {
  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return {
      valid: false,
      error: "Missing or invalid Authorization header"
    };
  }
  const token = authHeader.substring(7);
  try {
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (error || !user) {
      return {
        valid: false,
        error: "Invalid or expired token"
      };
    }
    const agentId = user.id;
    if (!agentId) {
      return {
        valid: false,
        error: "Agent ID not found in token"
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

// Validate investigation prompt
function validatePrompt(prompt: string): { valid: boolean; message?: string } {
  if (!prompt || typeof prompt !== "string") {
    return { valid: false, message: "Prompt is required and must be a string" };
  }

  const trimmed = prompt.trim();
  if (trimmed.length < 10) {
    return { valid: false, message: "Prompt must be at least 10 characters" };
  }

  const wordCount = trimmed.split(/\s+/).length;
  if (wordCount < 3) {
    return { valid: false, message: "Prompt must contain at least 3 words" };
  }

  return { valid: true };
}

// Check rate limiting for investigations and patch executions
async function checkRateLimit(
  userId: string,
  agentId: string,
  type: "investigation" | "patch"
): Promise<{ allowed: boolean; message?: string; remaining?: number }> {
  const tenMinutesAgo = new Date(Date.now() - 10 * 60 * 1000).toISOString();
  const maxRequests = 10;

  let query;
  if (type === "investigation") {
    query = supabase
      .from("investigations")
      .select("id", { count: "exact" })
      .eq("agent_id", agentId)
      .eq("initiated_by", userId)
      .gt("created_at", tenMinutesAgo);
  } else {
    query = supabase
      .from("patch_executions")
      .select("id", { count: "exact" })
      .eq("agent_id", agentId)
      .eq("triggered_by", userId)
      .gt("started_at", tenMinutesAgo);
  }

  const { data, error, count } = await query;

  if (error) {
    console.error(`Error checking rate limit for ${type}:`, error);
    return { allowed: true }; // Allow on error to not block operations
  }

  const requestCount = count || 0;
  const remaining = Math.max(0, maxRequests - requestCount);

  if (requestCount >= maxRequests) {
    return {
      allowed: false,
      message: `Rate limit exceeded. Maximum ${maxRequests} ${type}s per agent per 10 minutes. Try again in 10 minutes.`,
      remaining: 0
    };
  }

  return {
    allowed: true,
    remaining
  };
}
Deno.serve(async (req) => {
  const url = new URL(req.url);
  console.log(`${req.method} ${url.pathname}`);

  if (req.method === "OPTIONS") {
    return new Response(null, {
      status: 200,
      headers: {
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, PATCH, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization"
      }
    });
  }

  const authHeader = req.headers.get("Authorization") || "";
  const tokenValidation = await validateAgentToken(authHeader);
  if (!tokenValidation.valid || !tokenValidation.agentId) {
    console.log(`Auth failed: ${tokenValidation.error}`);
    return new Response(
      JSON.stringify({
        error: tokenValidation.error
      }),
      {
        status: 401,
        headers: {
          "Content-Type": "application/json"
        }
      }
    );
  }

  const agentId = tokenValidation.agentId;
  console.log(`Authenticated agent: ${agentId}`);

  try {
    // POST /connection-status - Update websocket connection status
    if (req.method === "POST" && url.pathname.endsWith("/connection-status")) {
      const body = await req.json();
      const connected = body.connected === true;
      const updateData: any = {
        websocket_connected: connected
      };
      if (connected) {
        updateData.websocket_connected_at = new Date().toISOString();
      } else {
        updateData.websocket_disconnected_at = new Date().toISOString();
      }
      const { error } = await supabase
        .from("agents")
        .update(updateData)
        .eq("id", agentId);
      if (error) throw new Error(`Failed to update connection status: ${error.message}`);
      console.log(`Updated agent ${agentId} connection status to ${connected}`);
      return new Response(
        JSON.stringify({
          success: true
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // POST /investigations - Create a new investigation
    if (req.method === "POST" && url.pathname.endsWith("/investigations")) {
      const body = await req.json();
      const prompt = body.prompt || body.issue;
      const userId = body.user_id || body.initiated_by;

      if (!prompt) {
        return new Response(
          JSON.stringify({
            error: "invalid_request",
            message: "Prompt/issue is required"
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Validate prompt format
      const promptValidation = validatePrompt(prompt);
      if (!promptValidation.valid) {
        return new Response(
          JSON.stringify({
            error: "invalid_prompt",
            message: promptValidation.message
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Check rate limiting
      const rateLimit = await checkRateLimit(userId, agentId, "investigation");
      if (!rateLimit.allowed) {
        return new Response(
          JSON.stringify({
            error: "rate_limit_exceeded",
            message: rateLimit.message,
            remaining: rateLimit.remaining
          }),
          {
            status: 429,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Create investigation record
      const investigationId = `inv_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const { data: investigation, error: insertError } = await supabase
        .from("investigations")
        .insert({
          investigation_id: investigationId,
          issue: prompt,
          agent_id: agentId,
          initiated_by: userId,
          status: "initiated",
          priority: body.priority || "medium",
          metadata: body.metadata || {}
        })
        .select();

      if (insertError) {
        throw new Error(`Failed to create investigation: ${insertError.message}`);
      }

      console.log(`Created investigation ${investigationId} for agent ${agentId}`);

      return new Response(
        JSON.stringify({
          success: true,
          investigation_id: investigationId,
          data: investigation
        }),
        {
          status: 201,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // PATCH /update-investigation - Update investigation status and episode_id
    if (req.method === "PATCH" && url.pathname.endsWith("/update-investigation")) {
      const body = await req.json();
      const investigationId = body.investigation_id;
      const status = body.status;
      const episodeId = body.episode_id;

      if (!investigationId || !status) {
        return new Response(
          JSON.stringify({
            error: "Missing investigation_id or status"
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Verify the investigation belongs to this agent
      const { data: investigation } = await supabase
        .from("investigations")
        .select("agent_id, initiated_by")
        .eq("investigation_id", investigationId)
        .single();

      if (!investigation || investigation.agent_id !== agentId) {
        return new Response(
          JSON.stringify({
            error: "Investigation not found or access denied"
          }),
          {
            status: 403,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      const updateData: any = {
        status,
        updated_at: new Date().toISOString()
      };
      if (episodeId) updateData.episode_id = episodeId;

      const { error } = await supabase
        .from("investigations")
        .update(updateData)
        .eq("investigation_id", investigationId);

      if (error) throw new Error(`Failed to update investigation: ${error.message}`);
      console.log(
        `Updated investigation ${investigationId} status to ${status}${
          episodeId ? ` with episode_id ${episodeId}` : ""
        }`
      );

      return new Response(
        JSON.stringify({
          success: true
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // POST /patch-executions - Create a new patch execution
    if (req.method === "POST" && url.pathname.endsWith("/patch-executions")) {
      const body = await req.json();
      const userId = body.user_id || body.triggered_by;
      const scriptId = body.script_id;
      const executionType = body.execution_type || "manual";
      const command = body.command;

      if (!command) {
        return new Response(
          JSON.stringify({
            error: "invalid_request",
            message: "Command is required"
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Check rate limiting
      const rateLimit = await checkRateLimit(userId, agentId, "patch");
      if (!rateLimit.allowed) {
        return new Response(
          JSON.stringify({
            error: "rate_limit_exceeded",
            message: rateLimit.message,
            remaining: rateLimit.remaining
          }),
          {
            status: 429,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Create patch execution record
      const { data: execution, error: insertError } = await supabase
        .from("patch_executions")
        .insert({
          agent_id: agentId,
          script_id: scriptId,
          execution_type: executionType,
          command: command,
          status: "pending",
          triggered_by: userId,
          should_reboot: body.should_reboot || false,
          metadata: {
            dry_run: body.dry_run || false,
            apply: body.apply !== false,
            package_exceptions: body.package_exceptions || [],
            schedule: body.schedule || null,
            ...body.metadata
          }
        })
        .select();

      if (insertError) {
        throw new Error(`Failed to create patch execution: ${insertError.message}`);
      }

      console.log(`Created patch execution for agent ${agentId}`);

      return new Response(
        JSON.stringify({
          success: true,
          execution_id: execution[0]?.id,
          data: execution
        }),
        {
          status: 201,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // PATCH /patch-execution-status - Update patch execution status
    if (req.method === "PATCH" && url.pathname.endsWith("/patch-execution-status")) {
      const body = await req.json();
      const executionId = body.execution_id;
      const status = body.status;
      const errorMessage = body.error_message;
      const startedAt = body.started_at;
      const completedAt = body.completed_at;

      if (!executionId || !status) {
        return new Response(
          JSON.stringify({
            error: "Missing execution_id or status"
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Verify the execution belongs to this agent
      const { data: execution } = await supabase
        .from("patch_executions")
        .select("agent_id")
        .eq("id", executionId)
        .single();

      if (!execution || execution.agent_id !== agentId) {
        return new Response(
          JSON.stringify({
            error: "Patch execution not found or access denied"
          }),
          {
            status: 403,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      const updateData: any = {
        status
      };
      if (startedAt) updateData.started_at = startedAt;
      if (completedAt) updateData.completed_at = completedAt;
      if (errorMessage) updateData.error_message = errorMessage;

      const { error } = await supabase
        .from("patch_executions")
        .update(updateData)
        .eq("id", executionId);

      if (error) throw new Error(`Failed to update patch status: ${error.message}`);
      console.log(`Updated patch execution ${executionId} status to ${status}`);

      return new Response(
        JSON.stringify({
          success: true
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // POST /patch-execution-logs - Update patch execution logs reference
    if (req.method === "POST" && url.pathname.endsWith("/patch-execution-logs")) {
      const body = await req.json();
      const executionId = body.execution_id;
      const logsPath = body.logs_path;
      const exitCode = body.exit_code;
      const stdout = body.stdout_storage_path;
      const stderr = body.stderr_storage_path;

      if (!executionId || !logsPath) {
        return new Response(
          JSON.stringify({
            error: "Missing execution_id or logs_path"
          }),
          {
            status: 400,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      // Verify the execution belongs to this agent
      const { data: execution } = await supabase
        .from("patch_executions")
        .select("agent_id")
        .eq("id", executionId)
        .single();

      if (!execution || execution.agent_id !== agentId) {
        return new Response(
          JSON.stringify({
            error: "Patch execution not found or access denied"
          }),
          {
            status: 403,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      const updateData: any = {
        logs_path: logsPath
      };
      if (exitCode !== undefined) updateData.exit_code = exitCode;
      if (stdout) updateData.stdout_storage_path = stdout;
      if (stderr) updateData.stderr_storage_path = stderr;

      const { error } = await supabase
        .from("patch_executions")
        .update(updateData)
        .eq("id", executionId);

      if (error) throw new Error(`Failed to update logs path: ${error.message}`);
      console.log(`Updated patch execution ${executionId} logs path to ${logsPath}`);

      return new Response(
        JSON.stringify({
          success: true
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    // GET /patch-scripts/:id - Get patch script info
    if (req.method === "GET" && url.pathname.includes("/patch-scripts/")) {
      const pathSegments = url.pathname.split("/");
      const scriptId = pathSegments[pathSegments.length - 1];

      const { data: script, error } = await supabase
        .from("patch_scripts")
        .select("id, name, script_storage_path, os_family, package_manager")
        .eq("id", scriptId)
        .single();

      if (error || !script) {
        return new Response(
          JSON.stringify({
            error: "Script not found"
          }),
          {
            status: 404,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      return new Response(JSON.stringify(script), {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      });
    }

    // GET /patch-executions - Get pending patch executions
    if (req.method === "GET" && url.pathname.endsWith("/patch-executions")) {
      const { data, error } = await supabase
        .from("patch_executions")
        .select("*")
        .eq("agent_id", agentId)
        .eq("status", "pending")
        .order("started_at", {
          ascending: false
        });

      if (error) throw new Error(`Database error: ${error.message}`);
      console.log(`Found ${data?.length || 0} pending patch executions`);

      return new Response(JSON.stringify(data || []), {
        status: 200,
        headers: {
          "Content-Type": "application/json"
        }
      });
    }

    // PATCH /patch-executions/:id - Update patch execution
    if (req.method === "PATCH" && url.pathname.includes("/patch-executions/")) {
      const pathSegments = url.pathname.split("/");
      const executionId = pathSegments[pathSegments.length - 1];
      const updates = await req.json();

      console.log(`Updating patch execution ${executionId}`);

      const { data: execution } = await supabase
        .from("patch_executions")
        .select("agent_id")
        .eq("id", executionId)
        .single();

      if (!execution || execution.agent_id !== agentId) {
        return new Response(
          JSON.stringify({
            error: "Patch execution not found or access denied"
          }),
          {
            status: 404,
            headers: {
              "Content-Type": "application/json"
            }
          }
        );
      }

      const { error } = await supabase
        .from("patch_executions")
        .update(updates)
        .eq("id", executionId);

      if (error) throw new Error(`Update failed: ${error.message}`);
      console.log(`Successfully updated patch execution ${executionId}`);

      return new Response(
        JSON.stringify({
          success: true
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json"
          }
        }
      );
    }

    return new Response(
      JSON.stringify({
        error: "Not found",
        path: url.pathname
      }),
      {
        status: 404,
        headers: {
          "Content-Type": "application/json"
        }
      }
    );
  } catch (error) {
    console.error("Proxy error:", error);
    return new Response(
      JSON.stringify({
        error: "Internal server error",
        details: error instanceof Error ? error.message : "Unknown error"
      }),
      {
        status: 500,
        headers: {
          "Content-Type": "application/json"
        }
      }
    );
  }
});
