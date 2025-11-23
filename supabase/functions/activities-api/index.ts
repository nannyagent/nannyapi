import "jsr:@supabase/functions-js/edge-runtime.d.ts";
const SUPABASE_URL = Deno.env.get('SUPABASE_URL');
const SUPABASE_ANON_KEY = Deno.env.get('SUPABASE_ANON_KEY');
// Validate JWT token from Supabase Auth
async function validateToken(authHeader) {
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return {
      valid: false,
      error: 'Missing or invalid Authorization header'
    };
  }
  const token = authHeader.substring(7);
  try {
    const response = await fetch(`${SUPABASE_URL}/auth/v1/user`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'apikey': SUPABASE_ANON_KEY
      }
    });
    if (!response.ok) {
      return {
        valid: false,
        error: 'Invalid or expired token'
      };
    }
    const user = await response.json();
    return {
      valid: true,
      userId: user.id
    };
  } catch (err) {
    console.error('Token validation error:', err);
    return {
      valid: false,
      error: 'Token validation failed'
    };
  }
}
Deno.serve(async (req)=>{
  // Handle CORS for all requests
  if (req.method === 'OPTIONS') {
    return new Response(null, {
      status: 200,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type, Authorization, x-client-info, apikey'
      }
    });
  }
  try {
    // Validate authentication
    const authHeader = req.headers.get('Authorization');
    const tokenValidation = await validateToken(authHeader);
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
    const userId = tokenValidation.userId;
    const url = new URL(req.url);
    // Parse query parameters
    const filterType = url.searchParams.get('type'); // investigations, agents, health, user
    const page = Math.max(1, parseInt(url.searchParams.get('page') || '1'));
    const limit = 5; // Fixed limit as per requirements
    const offset = (page - 1) * limit;
    // Build query based on filter type
    const typePatterns = {
      'investigations': [
        'investigation_created',
        'investigation_updated',
        'investigation_completed',
        'investigation_failed'
      ],
      'agents': [
        'agent_login',
        'agent_logout',
        'device_registered',
        'websocket_connected',
        'websocket_disconnected'
      ],
      'health': [
        'health_check',
        'performance_alert',
        'error_occurred'
      ],
      'user': [
        'user_login',
        'user_logout',
        'settings_viewed',
        'settings_updated',
        'dashboard_viewed',
        'profile_updated'
      ]
    };
    // Build PostgREST query URL
    const queryUrl = new URL(`${SUPABASE_URL}/rest/v1/activities`);
    queryUrl.searchParams.set('select', '*');
    queryUrl.searchParams.set('user_id', `eq.${userId}`);
    queryUrl.searchParams.set('order', 'created_at.desc');
    queryUrl.searchParams.set('limit', limit.toString());
    queryUrl.searchParams.set('offset', offset.toString());
    if (filterType) {
      const types = typePatterns[filterType];
      if (!types) {
        return new Response(JSON.stringify({
          error: 'Invalid filter type. Supported: investigations, agents, health, user'
        }), {
          status: 400,
          headers: {
            'Content-Type': 'application/json',
            'Access-Control-Allow-Origin': '*'
          }
        });
      }
      queryUrl.searchParams.set('activity_type', `in.(${types.join(',')})`);
    }
    // Fetch activities
    const response = await fetch(queryUrl.toString(), {
      headers: {
        'apikey': SUPABASE_ANON_KEY,
        'Authorization': authHeader,
        'Prefer': 'count=exact'
      }
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch activities: ${response.statusText}`);
    }
    const activities = await response.json();
    const totalCount = parseInt(response.headers.get('content-range')?.split('/')[1] || '0');
    const totalPages = Math.ceil(totalCount / limit);
    return new Response(JSON.stringify({
      activities: activities,
      pagination: {
        page,
        limit,
        total: totalCount,
        totalPages,
        hasNext: page < totalPages,
        hasPrev: page > 1
      },
      filter: filterType || 'all'
    }), {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  } catch (error) {
    console.error('Activities API error:', error);
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
