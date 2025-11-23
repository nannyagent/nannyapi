import { createClient } from "npm:@supabase/supabase-js@2.32.0";import { createClient } from "npm:@supabase/supabase-js@2.32.0";

import { randomUUID } from "node:crypto";import { randomUUID } from "node:crypto";

import { createHmac } from "node:crypto";import { createHmac } from "node:crypto";

const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';const SUPABASE_URL = Deno.env.get('SUPABASE_URL') || '';

const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || '';const SUPABASE_SERVICE_ROLE_KEY = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') || '';

const FRONTEND_URL = Deno.env.get('FRONTEND_URL') || 'http://localhost:8080';const FRONTEND_URL = Deno.env.get('FRONTEND_URL') || 'http://localhost:8080';

const AGENT_TOKEN_HMAC_SECRET = Deno.env.get('AGENT_TOKEN_HMAC_SECRET') || '';const AGENT_TOKEN_HMAC_SECRET = Deno.env.get('AGENT_TOKEN_HMAC_SECRET') || '';

if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY) {if (!SUPABASE_URL || !SUPABASE_SERVICE_ROLE_KEY) {

  console.error('Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY');  console.error('Missing SUPABASE_URL or SUPABASE_SERVICE_ROLE_KEY');

}}

if (!AGENT_TOKEN_HMAC_SECRET) {if (!AGENT_TOKEN_HMAC_SECRET) {

  console.error('Missing AGENT_TOKEN_HMAC_SECRET');  console.error('Missing AGENT_TOKEN_HMAC_SECRET');

}}

const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {const supabase = createClient(SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, {

  auth: {  auth: {

    persistSession: false    persistSession: false

  }  }

});});

function hashToken(token) {function hashToken(token) {

  return createHmac('sha256', AGENT_TOKEN_HMAC_SECRET).update(token).digest('hex');  return createHmac('sha256', AGENT_TOKEN_HMAC_SECRET).update(token).digest('hex');

}}

function generateUserCode() {function generateUserCode() {

  // Generate an 8-character alphanumeric code  // Generate an 8-character alphanumeric code

  return Math.random().toString(36).substring(2, 10).toUpperCase();  return Math.random().toString(36).substring(2, 10).toUpperCase();

}}

// POST / - Create device code for authorization flow// POST / - Create device code for authorization flow

async function handleDeviceStart(req) {async function handleDeviceStart(req) {

  try {  try {

    const body = await req.json().catch(()=>({}));    const body = await req.json().catch(()=>({}));

    const client_id = body.client_id || 'unknown';    const client_id = body.client_id || 'unknown';

    const scope = body.scope || 'agent:register';    const scope = body.scope || 'agent:register';

    const device_code = randomUUID();    const device_code = randomUUID();

    const user_code = generateUserCode();    const user_code = generateUserCode();

    const expires_in = 600; // 10 minutes    const expires_in = 600; // 10 minutes

    const expires_at = new Date(Date.now() + expires_in * 1000).toISOString();    const expires_at = new Date(Date.now() + expires_in * 1000).toISOString();

    // Store device code in the database    // Store device code in the database

    const { error } = await supabase.from('agent_device_codes').insert({    const { error } = await supabase.from('agent_device_codes').insert({

      device_code,      device_code,

      user_code,      user_code,

      client_id,      client_id,

      scope,      scope,

      expires_at,      expires_at,

      authorized: false,      authorized: false,

      created_at: new Date().toISOString()      created_at: new Date().toISOString()

    });    });

    if (error) {    if (error) {

      console.error('Database error:', error);      console.error('Database error:', error);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Failed to create device code'        error: 'Failed to create device code'

      }), {      }), {

        status: 500,        status: 500,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Create verification URL for the user    // Create verification URL for the user

    const verification_uri = `${FRONTEND_URL.replace(/\/$/, '')}/agent/authorize?user_code=${encodeURIComponent(user_code)}`;    const verification_uri = `${FRONTEND_URL.replace(/\/$/, '')}/agent/authorize?user_code=${encodeURIComponent(user_code)}`;

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      device_code,      device_code,

      user_code,      user_code,

      verification_uri,      verification_uri,

      verification_uri_complete: verification_uri,      verification_uri_complete: verification_uri,

      expires_in,      expires_in,

      interval: 5 // Polling interval in seconds      interval: 5 // Polling interval in seconds

    }), {    }), {

      status: 200,      status: 200,

      headers: {      headers: {

        'Content-Type': 'application/json',        'Content-Type': 'application/json',

        'Access-Control-Allow-Origin': '*',        'Access-Control-Allow-Origin': '*',

        'Access-Control-Allow-Methods': 'POST',        'Access-Control-Allow-Methods': 'POST',

        'Access-Control-Allow-Headers': 'Content-Type'        'Access-Control-Allow-Headers': 'Content-Type'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleDeviceStart:', error);    console.error('Error in handleDeviceStart:', error);

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

// POST /register - Agent registration (called by agent after user authorization)// POST /register - Agent registration (called by agent after user authorization)

async function handleAgentRegister(req) {async function handleAgentRegister(req) {

  try {  try {

    const body = await req.json().catch(()=>({}));    const body = await req.json().catch(()=>({}));

    const { device_code, agent_name, agent_type, hostname } = body;    const { device_code, agent_name, agent_type, hostname } = body;

    if (!device_code) {    if (!device_code) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'device_code is required'        error: 'device_code is required'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if device code exists and is authorized    // Check if device code exists and is authorized

    const { data: deviceData, error: deviceError } = await supabase.from('agent_device_codes').select('*').eq('device_code', device_code).single();    const { data: deviceData, error: deviceError } = await supabase.from('agent_device_codes').select('*').eq('device_code', device_code).single();

    if (deviceError || !deviceData) {    if (deviceError || !deviceData) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Invalid device code'        error: 'Invalid device code'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if expired    // Check if expired

    if (new Date(deviceData.expires_at) < new Date()) {    if (new Date(deviceData.expires_at) < new Date()) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Device code expired'        error: 'Device code expired'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if not authorized yet    // Check if not authorized yet

    if (!deviceData.authorized) {    if (!deviceData.authorized) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Device not authorized yet'        error: 'Device not authorized yet'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if already has a registration for this device code    // Check if already has a registration for this device code

    const { data: existingReg } = await supabase.from('agent_registrations').select('id, agent_name').eq('device_code', device_code).single();    const { data: existingReg } = await supabase.from('agent_registrations').select('id, agent_name').eq('device_code', device_code).single();

    if (existingReg) {    if (existingReg) {

      // Registration already exists - check if we have the token stored      // Registration already exists - check if we have the token stored

      const { data: tokenData } = await supabase.from('agent_device_codes').select('stored_token, stored_agent_id').eq('device_code', device_code).single();      const { data: tokenData } = await supabase.from('agent_device_codes').select('stored_token, stored_agent_id').eq('device_code', device_code).single();

      if (tokenData?.stored_token && tokenData?.stored_agent_id) {      if (tokenData?.stored_token && tokenData?.stored_agent_id) {

        // Return the stored token        // Return the stored token

        return new Response(JSON.stringify({        return new Response(JSON.stringify({

          access_token: tokenData.stored_token,          access_token: tokenData.stored_token,

          token_type: 'bearer',          token_type: 'bearer',

          expires_in: 30 * 24 * 60 * 60,          expires_in: 30 * 24 * 60 * 60,

          agent_id: tokenData.stored_agent_id,          agent_id: tokenData.stored_agent_id,

          agent_name: existingReg.agent_name          agent_name: existingReg.agent_name

        }), {        }), {

          status: 200,          status: 200,

          headers: {          headers: {

            'Content-Type': 'application/json',            'Content-Type': 'application/json',

            'Access-Control-Allow-Origin': '*'            'Access-Control-Allow-Origin': '*'

          }          }

        });        });

      } else {      } else {

        return new Response(JSON.stringify({        return new Response(JSON.stringify({

          error: 'Device code already used but token not available'          error: 'Device code already used but token not available'

        }), {        }), {

          status: 400,          status: 400,

          headers: {          headers: {

            'Content-Type': 'application/json'            'Content-Type': 'application/json'

          }          }

        });        });

      }      }

    }    }

    // All checks passed - proceed with registration    // All checks passed - proceed with registration

    const agentId = randomUUID();    const agentId = randomUUID();

    const accessToken = randomUUID();    const accessToken = randomUUID();

    const tokenHash = hashToken(accessToken);    const tokenHash = hashToken(accessToken);

    const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(); // 30 days    const expiresAt = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(); // 30 days

    // Start transaction - Insert all records    // Start transaction - Insert all records

    // 1. Insert agent registration    // 1. Insert agent registration

    const { error: registrationError } = await supabase.from('agent_registrations').insert({    const { error: registrationError } = await supabase.from('agent_registrations').insert({

      id: agentId,      id: agentId,

      user_id: deviceData.user_id,      user_id: deviceData.user_id,

      device_code: device_code,      device_code: device_code,

      agent_name: agent_name || 'nannyagent-cli',      agent_name: agent_name || 'nannyagent-cli',

      agent_type: agent_type || 'nanny',      agent_type: agent_type || 'nanny',

      metadata: {      metadata: {

        hostname: hostname || 'unknown'        hostname: hostname || 'unknown'

      },      },

      status: 'active',      status: 'active',

      created_at: new Date().toISOString()      created_at: new Date().toISOString()

    });    });

    if (registrationError) {    if (registrationError) {

      console.error('Registration error:', registrationError);      console.error('Registration error:', registrationError);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Failed to register agent'        error: 'Failed to register agent'

      }), {      }), {

        status: 500,        status: 500,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // 2. Create agent token    // 2. Create agent token

    const { error: tokenError } = await supabase.from('agent_tokens').insert({    const { error: tokenError } = await supabase.from('agent_tokens').insert({

      id: randomUUID(),      id: randomUUID(),

      agent_id: agentId,      agent_id: agentId,

      user_id: deviceData.user_id,      user_id: deviceData.user_id,

      token_hash: tokenHash,      token_hash: tokenHash,

      expires_at: expiresAt,      expires_at: expiresAt,

      revoked: false,      revoked: false,

      created_at: new Date().toISOString()      created_at: new Date().toISOString()

    });    });

    if (tokenError) {    if (tokenError) {

      console.error('Token creation error:', tokenError);      console.error('Token creation error:', tokenError);

      // Try to cleanup registration      // Try to cleanup registration

      await supabase.from('agent_registrations').delete().eq('id', agentId);      await supabase.from('agent_registrations').delete().eq('id', agentId);

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Failed to create agent token'        error: 'Failed to create agent token'

      }), {      }), {

        status: 500,        status: 500,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // 3. Create initial heartbeat record    // 3. Create initial heartbeat record

    const { error: heartbeatError } = await supabase.from('agent_heartbeats').insert({    const { error: heartbeatError } = await supabase.from('agent_heartbeats').insert({

      id: randomUUID(),      id: randomUUID(),

      agent_id: agentId,      agent_id: agentId,

      user_id: deviceData.user_id,      user_id: deviceData.user_id,

      status: 'online',      status: 'online',

      last_seen: new Date().toISOString(),      last_seen: new Date().toISOString(),

      created_at: new Date().toISOString()      created_at: new Date().toISOString()

    });    });

    if (heartbeatError) {    if (heartbeatError) {

      console.error('Heartbeat creation error:', heartbeatError);      console.error('Heartbeat creation error:', heartbeatError);

    // Don't fail registration if heartbeat fails, just log it    // Don't fail registration if heartbeat fails, just log it

    }    }

    // 4. Store the token temporarily in device_codes for retrieval    // 4. Store the token temporarily in device_codes for retrieval

    const { error: storeError } = await supabase.from('agent_device_codes').update({    const { error: storeError } = await supabase.from('agent_device_codes').update({

      consumed: true,      consumed: true,

      stored_token: accessToken,      stored_token: accessToken,

      stored_agent_id: agentId      stored_agent_id: agentId

    }).eq('device_code', device_code);    }).eq('device_code', device_code);

    if (storeError) {    if (storeError) {

      console.error('Failed to store token in device code:', storeError);      console.error('Failed to store token in device code:', storeError);

    // Don't fail the registration for this    // Don't fail the registration for this

    }    }

    // Success! Return the access token    // Success! Return the access token

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      access_token: accessToken,      access_token: accessToken,

      token_type: 'bearer',      token_type: 'bearer',

      expires_in: 30 * 24 * 60 * 60,      expires_in: 30 * 24 * 60 * 60,

      agent_id: agentId,      agent_id: agentId,

      agent_name: agent_name || 'nannyagent-cli'      agent_name: agent_name || 'nannyagent-cli'

    }), {    }), {

      status: 200,      status: 200,

      headers: {      headers: {

        'Content-Type': 'application/json',        'Content-Type': 'application/json',

        'Access-Control-Allow-Origin': '*'        'Access-Control-Allow-Origin': '*'

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

// GET /token/{device_code} - Retrieve token for a device code// GET /token/{device_code} - Retrieve token for a device code

async function handleGetToken(req) {async function handleGetToken(req) {

  try {  try {

    const url = new URL(req.url);    const url = new URL(req.url);

    const pathParts = url.pathname.split('/');    const pathParts = url.pathname.split('/');

    const device_code = pathParts[pathParts.length - 1];    const device_code = pathParts[pathParts.length - 1];

    if (!device_code) {    if (!device_code) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'device_code is required'        error: 'device_code is required'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check device code and get stored token    // Check device code and get stored token

    const { data: deviceData, error: deviceError } = await supabase.from('agent_device_codes').select('*').eq('device_code', device_code).single();    const { data: deviceData, error: deviceError } = await supabase.from('agent_device_codes').select('*').eq('device_code', device_code).single();

    if (deviceError || !deviceData) {    if (deviceError || !deviceData) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Invalid device code'        error: 'Invalid device code'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if expired    // Check if expired

    if (new Date(deviceData.expires_at) < new Date()) {    if (new Date(deviceData.expires_at) < new Date()) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Device code expired'        error: 'Device code expired'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if not authorized yet    // Check if not authorized yet

    if (!deviceData.authorized) {    if (!deviceData.authorized) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Device not authorized yet'        error: 'Device not authorized yet'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Check if we have a stored token    // Check if we have a stored token

    if (!deviceData.stored_token || !deviceData.stored_agent_id) {    if (!deviceData.stored_token || !deviceData.stored_agent_id) {

      return new Response(JSON.stringify({      return new Response(JSON.stringify({

        error: 'Token not yet available'        error: 'Token not yet available'

      }), {      }), {

        status: 400,        status: 400,

        headers: {        headers: {

          'Content-Type': 'application/json'          'Content-Type': 'application/json'

        }        }

      });      });

    }    }

    // Get agent name from registration    // Get agent name from registration

    const { data: regData } = await supabase.from('agent_registrations').select('agent_name').eq('id', deviceData.stored_agent_id).single();    const { data: regData } = await supabase.from('agent_registrations').select('agent_name').eq('id', deviceData.stored_agent_id).single();

    // Return the stored token    // Return the stored token

    return new Response(JSON.stringify({    return new Response(JSON.stringify({

      access_token: deviceData.stored_token,      access_token: deviceData.stored_token,

      token_type: 'bearer',      token_type: 'bearer',

      expires_in: 30 * 24 * 60 * 60,      expires_in: 30 * 24 * 60 * 60,

      agent_id: deviceData.stored_agent_id,      agent_id: deviceData.stored_agent_id,

      agent_name: regData?.agent_name || 'nannyagent-cli'      agent_name: regData?.agent_name || 'nannyagent-cli'

    }), {    }), {

      status: 200,      status: 200,

      headers: {      headers: {

        'Content-Type': 'application/json',        'Content-Type': 'application/json',

        'Access-Control-Allow-Origin': '*'        'Access-Control-Allow-Origin': '*'

      }      }

    });    });

  } catch (error) {  } catch (error) {

    console.error('Error in handleGetToken:', error);    console.error('Error in handleGetToken:', error);

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

function handleOptions() {function handleOptions() {

  return new Response(null, {  return new Response(null, {

    status: 204,    status: 204,

    headers: {    headers: {

      'Access-Control-Allow-Origin': '*',      'Access-Control-Allow-Origin': '*',

      'Access-Control-Allow-Methods': 'POST, GET, OPTIONS',      'Access-Control-Allow-Methods': 'POST, GET, OPTIONS',

      'Access-Control-Allow-Headers': 'Content-Type',      'Access-Control-Allow-Headers': 'Content-Type',

      'Access-Control-Max-Age': '86400'      'Access-Control-Max-Age': '86400'

    }    }

  });  });

}}

Deno.serve(async (req)=>{Deno.serve(async (req)=>{

  const url = new URL(req.url);  const url = new URL(req.url);

  console.log(`Request: ${req.method} ${url.pathname}`);  console.log(`Request: ${req.method} ${url.pathname}`);

  // Handle CORS preflight requests  // Handle CORS preflight requests

  if (req.method === 'OPTIONS') {  if (req.method === 'OPTIONS') {

    return handleOptions();    return handleOptions();

  }  }

  // Route based on path  // Route based on path

  if (req.method === 'POST') {  if (req.method === 'POST') {

    if (url.pathname.includes('register')) {    if (url.pathname.includes('register')) {

      console.log('Handling agent registration request');      console.log('Handling agent registration request');

      return handleAgentRegister(req);      return handleAgentRegister(req);

    } else {    } else {

      console.log('Handling device start request');      console.log('Handling device start request');

      return handleDeviceStart(req);      return handleDeviceStart(req);

    }    }

  }  }

  if (req.method === 'GET') {  if (req.method === 'GET') {

    if (url.pathname.includes('token/')) {    if (url.pathname.includes('token/')) {

      console.log('Handling token retrieval request');      console.log('Handling token retrieval request');

      return handleGetToken(req);      return handleGetToken(req);

    }    }

  }  }

  return new Response(JSON.stringify({  return new Response(JSON.stringify({

    error: 'Not Found',    error: 'Not Found',

    method: req.method,    method: req.method,

    path: url.pathname,    path: url.pathname,

    available_endpoints: [    available_endpoints: [

      'POST / - Create device code',      'POST / - Create device code',

      'POST /register - Register agent after authorization',      'POST /register - Register agent after authorization',

      'GET /token/{device_code} - Retrieve token for device code'      'GET /token/{device_code} - Retrieve token for device code'

    ]    ]

  }), {  }), {

    status: 404,    status: 404,

    headers: {    headers: {

      'Content-Type': 'application/json'      'Content-Type': 'application/json'

    }    }

  });  });

});});
