import { createClient } from 'https://esm.sh/@supabase/supabase-js@2';

const supabaseUrl = Deno.env.get('SUPABASE_URL') ?? '';
const supabaseServiceKey = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY') ?? '';

const supabase = createClient(supabaseUrl, supabaseServiceKey);

Deno.serve(async (req: Request) => {
  try {
    // Verify this is a cron job request
    const authHeader = req.headers.get('authorization');
    if (!authHeader) {
      return new Response(
        JSON.stringify({ error: 'Unauthorized' }),
        { status: 401, headers: { 'Content-Type': 'application/json' } }
      );
    }

    const currentTime = new Date();
    const expiryThreshold = new Date(currentTime.getTime() - 10 * 60 * 1000); // 10 minutes ago

    console.log('Starting device auth cleanup job at:', currentTime.toISOString());
    console.log('Deleting expired/unused codes before:', expiryThreshold.toISOString());

    // Delete expired device sessions that were never approved
    const { data: expiredSessions, error: selectError } = await supabase
      .from('device_sessions')
      .select('id, user_code, expires_at, status')
      .lt('expires_at', expiryThreshold.toISOString())
      .neq('status', 'approved');

    if (selectError) {
      console.error('Error querying expired sessions:', selectError);
      return new Response(
        JSON.stringify({ error: 'Failed to query expired sessions', details: selectError.message }),
        { status: 500, headers: { 'Content-Type': 'application/json' } }
      );
    }

    let deletedSessions = 0;
    let deletedFailedAttempts = 0;
    let deletedOldAttempts = 0;

    // Delete expired sessions
    if (expiredSessions && expiredSessions.length > 0) {
      const sessionIds = expiredSessions.map(s => s.id);
      
      const { error: deleteSessionError, count: sessionCount } = await supabase
        .from('device_sessions')
        .delete()
        .in('id', sessionIds);

      if (deleteSessionError) {
        console.error('Error deleting expired sessions:', deleteSessionError);
      } else {
        deletedSessions = sessionCount || expiredSessions.length;
        console.log(`Deleted ${deletedSessions} expired device sessions`);
      }

      // Also delete associated failed attempts for these sessions
      const userCodes = expiredSessions.map(s => s.user_code);
      const { error: deleteAttemptsError, count: attemptsCount } = await supabase
        .from('device_failed_attempts')
        .delete()
        .in('user_code', userCodes);

      if (deleteAttemptsError) {
        console.error('Error deleting related failed attempts:', deleteAttemptsError);
      } else {
        deletedFailedAttempts = attemptsCount || 0;
        console.log(`Deleted ${deletedFailedAttempts} related failed attempts`);
      }
    }

    // Also clean up old failed attempts (older than 24 hours)
    const twentyFourHoursAgo = new Date(currentTime.getTime() - 24 * 60 * 60 * 1000);
    const { error: deleteOldAttemptsError, count: oldAttemptsCount } = await supabase
      .from('device_failed_attempts')
      .delete()
      .lt('attempted_at', twentyFourHoursAgo.toISOString());

    if (deleteOldAttemptsError) {
      console.error('Error deleting old failed attempts:', deleteOldAttemptsError);
    } else {
      deletedOldAttempts = oldAttemptsCount || 0;
      console.log(`Deleted ${deletedOldAttempts} old failed attempts (> 24 hours)`);
    }

    return new Response(
      JSON.stringify({
        success: true,
        message: 'Device auth cleanup completed',
        deleted: {
          sessions: deletedSessions,
          related_failed_attempts: deletedFailedAttempts,
          old_failed_attempts: deletedOldAttempts,
          total: deletedSessions + deletedFailedAttempts + deletedOldAttempts
        }
      }),
      { 
        status: 200, 
        headers: { 'Content-Type': 'application/json' } 
      }
    );
  } catch (error) {
    console.error('Cleanup job error:', error);
    return new Response(
      JSON.stringify({ 
        error: 'Cleanup job failed', 
        message: error instanceof Error ? error.message : 'Unknown error' 
      }),
      { status: 500, headers: { 'Content-Type': 'application/json' } }
    );
  }
});
