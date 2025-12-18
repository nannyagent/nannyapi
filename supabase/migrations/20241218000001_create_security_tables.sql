-- Create password_change_attempts table for tracking failed login attempts
CREATE TABLE IF NOT EXISTS password_change_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  ip_address TEXT NOT NULL,
  success BOOLEAN DEFAULT FALSE,
  attempted_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create password_change_history table for tracking password changes
CREATE TABLE IF NOT EXISTS password_change_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  password_hash TEXT NOT NULL,
  ip_address TEXT NOT NULL,
  user_agent TEXT,
  changed_by_agent BOOLEAN DEFAULT FALSE,
  changed_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create account_lockout table for tracking locked accounts
CREATE TABLE IF NOT EXISTS account_lockout (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  locked_until TIMESTAMPTZ NOT NULL,
  reason TEXT,
  ip_address TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create user_mfa_settings table for storing MFA configuration
CREATE TABLE IF NOT EXISTS user_mfa_settings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL UNIQUE REFERENCES auth.users(id) ON DELETE CASCADE,
  totp_secret TEXT,
  backup_codes_hash TEXT[] DEFAULT ARRAY[]::TEXT[],
  mfa_enabled BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create user_mfa_backup_codes_used table for tracking backup code usage
CREATE TABLE IF NOT EXISTS user_mfa_backup_codes_used (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  backup_code_index INTEGER,
  ip_address TEXT,
  user_agent TEXT,
  used_for_login BOOLEAN DEFAULT FALSE,
  used_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create system_config table for storing configurable security settings
CREATE TABLE IF NOT EXISTS system_config (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key TEXT NOT NULL UNIQUE,
  value TEXT NOT NULL,
  description TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default security configuration values
INSERT INTO system_config (key, value, description) VALUES
  ('security.password_change_limit_per_24h', '5', 'Maximum password changes allowed per 24 hours'),
  ('security.password_history_window_hours', '24', 'Hours to check password history to prevent reuse'),
  ('security.account_lockout_duration_hours', '24', 'Hours to lock account after threshold exceeded'),
  ('security.failed_login_attempts_limit', '10', 'Maximum failed login attempts before lockout'),
  ('security.backup_codes_count', '8', 'Number of backup codes to generate for MFA')
ON CONFLICT (key) DO NOTHING;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_password_change_attempts_user_id ON password_change_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_password_change_attempts_attempted_at ON password_change_attempts(attempted_at);
CREATE INDEX IF NOT EXISTS idx_password_change_history_user_id ON password_change_history(user_id);
CREATE INDEX IF NOT EXISTS idx_account_lockout_user_id ON account_lockout(user_id);
CREATE INDEX IF NOT EXISTS idx_account_lockout_locked_until ON account_lockout(locked_until);
CREATE INDEX IF NOT EXISTS idx_user_mfa_settings_user_id ON user_mfa_settings(user_id);
CREATE INDEX IF NOT EXISTS idx_user_mfa_backup_codes_used_user_id ON user_mfa_backup_codes_used(user_id);

-- Enable RLS (Row Level Security)
ALTER TABLE password_change_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_change_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE account_lockout ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_mfa_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_mfa_backup_codes_used ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_config ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only see their own data
CREATE POLICY "Users can view their own password attempts" ON password_change_attempts
  FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY "Users can view their own password history" ON password_change_history
  FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY "Users can view their own lockout status" ON account_lockout
  FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY "Users can view their own MFA settings" ON user_mfa_settings
  FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY "Users can view their own backup code usage" ON user_mfa_backup_codes_used
  FOR SELECT USING (auth.uid() = user_id);

-- RLS Policy: Anyone can read system config (public)
CREATE POLICY "Public can read system config" ON system_config
  FOR SELECT USING (TRUE);

-- Service role can insert/update all tables
CREATE POLICY "Service role can manage password attempts" ON password_change_attempts
  FOR ALL USING (current_setting('role') = 'authenticated' OR auth.jwt() ->> 'role' = 'service_role');

CREATE POLICY "Service role can manage password history" ON password_change_history
  FOR ALL USING (current_setting('role') = 'authenticated' OR auth.jwt() ->> 'role' = 'service_role');

CREATE POLICY "Service role can manage lockouts" ON account_lockout
  FOR ALL USING (current_setting('role') = 'authenticated' OR auth.jwt() ->> 'role' = 'service_role');

CREATE POLICY "Service role can manage MFA settings" ON user_mfa_settings
  FOR ALL USING (current_setting('role') = 'authenticated' OR auth.jwt() ->> 'role' = 'service_role');

CREATE POLICY "Service role can manage backup code usage" ON user_mfa_backup_codes_used
  FOR ALL USING (current_setting('role') = 'authenticated' OR auth.jwt() ->> 'role' = 'service_role');
