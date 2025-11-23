import "jsr:@supabase/functions-js/edge-runtime.d.ts";
import { createClient } from "jsr:@supabase/supabase-js@2";
const supabase = createClient(Deno.env.get('SUPABASE_URL') ?? '', Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '');
// Store active WebSocket connections with connection start time
const activeConnections = new Map();
// Debug logging function with timestamps
function debugLog(message, data) {
  const timestamp = new Date().toISOString();
  if (data) {
    console.log(`[${timestamp}] ${message}`, JSON.stringify(data, null, 2));
  } else {
    console.log(`[${timestamp}] ${message}`);
  }
}
// Track connection duration and detect timeout patterns
function logConnectionDuration(agentId, event) {
  const connection = activeConnections.get(agentId);
  if (connection) {
    const durationMs = Date.now() - connection.startTime;
    const durationMin = Math.round(durationMs / 1000 / 60 * 10) / 10;
    debugLog(`⏱️ ${event} for agent ${agentId} after ${durationMin} minutes (${durationMs}ms)`);
    // Check for timeout patterns
    if (durationMs > 120000) {
      debugLog(`⚠️ POTENTIAL TIMEOUT PATTERN - Connection lasted ${durationMin}min`, {
        agentId,
        durationMs
      });
    }
  }
}
// Validate agent JWT token
async function validateAgentToken(authHeader) {
  debugLog('🔑 Starting token validation', {
    authHeader: authHeader?.substring(0, 50) + '...'
  });
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    debugLog('❌ Invalid auth header format');
    return {
      valid: false,
      error: 'Missing or invalid Authorization header'
    };
  }
  const token = authHeader.substring(7);
  debugLog('🔍 Extracted token', {
    tokenLength: token.length,
    tokenStart: token.substring(0, 20)
  });
  try {
    // Verify JWT token using Supabase Auth
    debugLog('📡 Calling supabase.auth.getUser...');
    const { data: { user }, error } = await supabase.auth.getUser(token);
    if (error) {
      debugLog('❌ Supabase auth error', error);
      return {
        valid: false,
        error: `Auth error: ${error.message}`
      };
    }
    if (!user) {
      debugLog('❌ No user returned from auth');
      return {
        valid: false,
        error: 'No user found for token'
      };
    }
    debugLog('✅ User authenticated', {
      userId: user.id,
      metadata: user.user_metadata
    });
    // Extract agent_id from user sub or metadata
    const agentId = user.user_metadata?.agent_id || user.id;
    if (!agentId) {
      debugLog('❌ No agent ID found');
      return {
        valid: false,
        error: 'Agent ID not found in token'
      };
    }
    debugLog('✅ Token validation successful', {
      agentId
    });
    return {
      valid: true,
      agentId
    };
  } catch (err) {
    debugLog('💥 Token validation exception', err);
    return {
      valid: false,
      error: `Token validation failed: ${err.message}`
    };
  }
}
// Update agent connection status in database
async function updateAgentConnectionStatus(agentId, connected) {
  debugLog(`📊 Updating agent connection status`, {
    agentId,
    connected
  });
  const updateData = {
    websocket_connected: connected,
    last_seen: new Date().toISOString()
  };
  if (connected) {
    updateData['websocket_connected_at'] = new Date().toISOString();
  } else {
    updateData['websocket_disconnected_at'] = new Date().toISOString();
  }
  const { error } = await supabase.from('agents').update(updateData).eq('id', agentId);
  if (error) {
    debugLog('❌ Failed to update agent status', error);
  } else {
    debugLog('✅ Agent status updated successfully');
  }
}
// Send task to specific agent
export async function sendTaskToAgent(agentId, task) {
  debugLog(`📤 Attempting to send task to agent`, {
    agentId,
    taskId: task.task_id
  });
  const connection = activeConnections.get(agentId);
  if (!connection || connection.socket.readyState !== WebSocket.OPEN) {
    debugLog(`❌ Agent not connected`, {
      agentId,
      hasConnection: !!connection,
      readyState: connection?.socket.readyState,
      connectionAge: connection ? Date.now() - connection.startTime : 0
    });
    return false;
  }
  try {
    const message = {
      type: 'investigation_task',
      data: task
    };
    debugLog(`📨 Sending task message`, {
      agentId,
      message
    });
    connection.socket.send(JSON.stringify(message));
    debugLog(`✅ Task sent successfully`, {
      agentId,
      taskId: task.task_id
    });
    return true;
  } catch (error) {
    debugLog(`💥 Failed to send task`, {
      agentId,
      error
    });
    return false;
  }
}
Deno.serve(async (req)=>{
  const requestStart = Date.now();
  debugLog('🌐 New request received', {
    method: req.method,
    url: req.url,
    timestamp: requestStart,
    headers: Object.fromEntries(req.headers.entries())
  });
  // Handle WebSocket upgrade
  if (req.headers.get("upgrade") !== "websocket") {
    debugLog('❌ Not a WebSocket upgrade request');
    return new Response("Expected WebSocket upgrade", {
      status: 400
    });
  }
  debugLog('🔄 Processing WebSocket upgrade...');
  // Validate authentication
  const authHeader = req.headers.get('Authorization') || '';
  debugLog('🔐 Starting authentication', {
    hasAuthHeader: !!authHeader
  });
  const tokenValidation = await validateAgentToken(authHeader);
  if (!tokenValidation.valid || !tokenValidation.agentId) {
    debugLog('🚫 Authentication failed', tokenValidation);
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
  debugLog('✅ Authentication successful', {
    agentId
  });
  // Check for existing connection
  const existingConnection = activeConnections.get(agentId);
  if (existingConnection) {
    debugLog('⚠️ Replacing existing connection', {
      agentId,
      existingConnectionAge: Date.now() - existingConnection.startTime
    });
    try {
      existingConnection.socket.close();
    } catch (e) {
      debugLog('🔧 Error closing existing connection', e);
    }
  }
  // Upgrade to WebSocket
  debugLog('⬆️ Upgrading to WebSocket...');
  const { socket, response } = Deno.upgradeWebSocket(req);
  // Handle WebSocket connection
  socket.onopen = async ()=>{
    const connectionTime = Date.now();
    debugLog(`🔌 WebSocket opened for agent`, {
      agentId,
      connectionTime
    });
    // Store connection with start time
    activeConnections.set(agentId, {
      socket,
      startTime: connectionTime
    });
    debugLog(`📝 Connection stored`, {
      agentId,
      totalConnections: activeConnections.size
    });
    // Update database
    await updateAgentConnectionStatus(agentId, true);
    // Send connection acknowledgment
    const ackMessage = {
      type: 'connection_ack',
      data: {
        agent_id: agentId,
        connected_at: new Date().toISOString(),
        message: 'WebSocket connection established',
        edge_function_start: requestStart,
        connection_start: connectionTime
      }
    };
    try {
      debugLog(`📤 Sending connection ack`, {
        agentId,
        ackMessage
      });
      socket.send(JSON.stringify(ackMessage));
      debugLog(`✅ Connection ack sent successfully`, {
        agentId
      });
    } catch (error) {
      debugLog(`💥 Failed to send connection ack`, {
        agentId,
        error
      });
    }
  };
  socket.onmessage = async (event)=>{
    const messageTime = Date.now();
    logConnectionDuration(agentId, 'Message received');
    debugLog(`📨 Message received from agent`, {
      agentId,
      dataLength: event.data?.length,
      messageTime
    });
    try {
      const message = JSON.parse(event.data);
      debugLog(`📋 Parsed message`, {
        agentId,
        messageType: message.type,
        message
      });
      switch(message.type){
        case 'heartbeat':
          logConnectionDuration(agentId, 'Heartbeat');
          debugLog(`💓 Processing heartbeat`, {
            agentId
          });
          // Respond to heartbeat
          const heartbeatAck = {
            type: 'heartbeat_ack',
            timestamp: new Date().toISOString(),
            connection_duration_ms: Date.now() - (activeConnections.get(agentId)?.startTime || messageTime)
          };
          try {
            socket.send(JSON.stringify(heartbeatAck));
            debugLog(`💓 Heartbeat ack sent`, {
              agentId,
              connectionAge: heartbeatAck.connection_duration_ms
            });
          } catch (error) {
            debugLog(`💥 Failed to send heartbeat ack`, {
              agentId,
              error
            });
          }
          // Update last_seen in database
          try {
            await supabase.from('agents').update({
              last_seen: new Date().toISOString()
            }).eq('id', agentId);
            debugLog(`📊 Updated last_seen for agent`, {
              agentId
            });
          } catch (error) {
            debugLog(`💥 Failed to update last_seen`, {
              agentId,
              error
            });
          }
          break;
        case 'task_result':
          const result = message.data;
          logConnectionDuration(agentId, 'Task result received');
          debugLog(`📝 Processing task result`, {
            agentId,
            taskId: result.task_id,
            success: result.success
          });
          // Update pending investigation with results
          try {
            const { error } = await supabase.from('pending_investigations').update({
              status: result.success ? 'completed' : 'failed',
              command_results: result.command_results,
              error_message: result.error,
              completed_at: new Date().toISOString()
            }).eq('id', result.task_id);
            if (error) {
              debugLog(`💥 Failed to update investigation result`, {
                agentId,
                taskId: result.task_id,
                error
              });
            } else {
              debugLog(`✅ Investigation result updated`, {
                agentId,
                taskId: result.task_id
              });
            }
          } catch (error) {
            debugLog(`💥 Exception updating investigation`, {
              agentId,
              taskId: result.task_id,
              error
            });
          }
          // Send acknowledgment
          try {
            socket.send(JSON.stringify({
              type: 'task_result_ack',
              task_id: result.task_id
            }));
            debugLog(`✅ Task result ack sent`, {
              agentId,
              taskId: result.task_id
            });
          } catch (error) {
            debugLog(`💥 Failed to send task result ack`, {
              agentId,
              taskId: result.task_id,
              error
            });
          }
          break;
        default:
          debugLog(`⚠️ Unknown message type`, {
            agentId,
            messageType: message.type,
            message
          });
      }
    } catch (error) {
      debugLog(`💥 Error processing message`, {
        agentId,
        error,
        rawData: event.data
      });
    }
  };
  socket.onclose = async (event)=>{
    logConnectionDuration(agentId, `Connection closed (code: ${event.code})`);
    debugLog(`🔌 WebSocket closed for agent`, {
      agentId,
      code: event.code,
      reason: event.reason,
      wasClean: event.wasClean,
      closeTime: Date.now()
    });
    // Analyze close code for diagnostics
    if (event.code === 1006) {
      debugLog(`🚨 CODE 1006 - ABNORMAL CLOSURE (likely Edge Function timeout/kill)`, {
        agentId
      });
    } else if (event.code === 1001) {
      debugLog(`📤 CODE 1001 - Going away (normal)`, {
        agentId
      });
    } else if (event.code === 1000) {
      debugLog(`✅ CODE 1000 - Normal closure`, {
        agentId
      });
    } else {
      debugLog(`❓ UNKNOWN CLOSE CODE: ${event.code}`, {
        agentId,
        reason: event.reason
      });
    }
    // Remove from active connections
    activeConnections.delete(agentId);
    debugLog(`🗑️ Connection removed`, {
      agentId,
      remainingConnections: activeConnections.size
    });
    // Update database
    await updateAgentConnectionStatus(agentId, false);
  };
  socket.onerror = (error)=>{
    logConnectionDuration(agentId, 'WebSocket error');
    debugLog(`💥 WebSocket error for agent`, {
      agentId,
      error
    });
  };
  debugLog(`🚀 WebSocket setup complete`, {
    agentId,
    setupTime: Date.now() - requestStart
  });
  return response;
});
// Export for use by other Edge Functions
globalThis.sendTaskToAgent = sendTaskToAgent;
debugLog('🏁 Edge function initialized');
