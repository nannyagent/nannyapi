-- Create user_mfa_failed_attempts table to track MFA validation failures
-- Used for locking accounts after N failed MFA attempts
CREATE TABLE IF NOT EXISTS user_mfa_failed_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  action TEXT NOT NULL, -- 'verify_totp', 'verify_backup_code'
  ip_address TEXT NOT NULL,
  user_agent TEXT,
  failed_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_user_mfa_failed_attempts_user_id 
  ON user_mfa_failed_attempts(user_id);

CREATE INDEX IF NOT EXISTS idx_user_mfa_failed_attempts_failed_at 
  ON user_mfa_failed_attempts(failed_at);

-- Create table for MFA lockout tracking (separate from account_lockout for MFA-specific lockouts)
CREATE TABLE IF NOT EXISTS mfa_lockout (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  locked_until TIMESTAMPTZ NOT NULL,
  reason TEXT,
  ip_address TEXT,
  failed_attempts_count INTEGER,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for performance
CREATE INDEX IF NOT EXISTS idx_mfa_lockout_user_id 
  ON mfa_lockout(user_id);

CREATE INDEX IF NOT EXISTS idx_mfa_lockout_locked_until 
  ON mfa_lockout(locked_until);

-- Enable RLS
ALTER TABLE user_mfa_failed_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_lockout ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can view their own MFA failed attempts
CREATE POLICY "Users can view their own MFA failed attempts" ON user_mfa_failed_attempts
  FOR SELECT USING (auth.uid() = user_id);

-- RLS Policy: Users can view their own MFA lockout status
CREATE POLICY "Users can view their own MFA lockout" ON mfa_lockout
  FOR SELECT USING (auth.uid() = user_id);

-- RLS Policy: Service role can manage MFA attempts and lockouts
CREATE POLICY "Service role can manage MFA failed attempts" ON user_mfa_failed_attempts
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');

CREATE POLICY "Service role can manage MFA lockouts" ON mfa_lockout
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');

-- Add MFA-specific configuration values to system_config
INSERT INTO system_config (key, value, description) VALUES
  ('security.mfa_failed_attempts_limit', '5', 'Maximum failed MFA attempts before MFA lockout'),
  ('security.mfa_lockout_duration_hours', '1', 'Hours to lock MFA after max failed attempts'),
  ('security.mfa_check_window_hours', '24', 'Hours to count failed MFA attempts within')
ON CONFLICT (key) DO NOTHING;
