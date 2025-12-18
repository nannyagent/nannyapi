-- Create device_sessions table for device code flow
CREATE TABLE IF NOT EXISTS device_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id TEXT NOT NULL UNIQUE,
  device_code_hash TEXT NOT NULL,
  user_code TEXT NOT NULL,
  client_metadata JSONB,
  status TEXT DEFAULT 'pending', -- pending, approved, rejected, expired
  requested_scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
  created_at TIMESTAMPTZ DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  interval_seconds INTEGER DEFAULT 5,
  approved_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
  approved_at TIMESTAMPTZ
);

-- Create device_failed_attempts table for rate limiting
CREATE TABLE IF NOT EXISTS device_failed_attempts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id TEXT NOT NULL, -- Agent identifier
  user_code TEXT, -- The code attempted
  ip_address TEXT,
  attempted_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create device_code_consumption table to prevent code reuse across agents
CREATE TABLE IF NOT EXISTS device_code_consumption (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_code TEXT NOT NULL UNIQUE,
  agent_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  consumed_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_device_sessions_user_code ON device_sessions(user_code);
CREATE INDEX IF NOT EXISTS idx_device_sessions_status ON device_sessions(status);
CREATE INDEX IF NOT EXISTS idx_device_sessions_expires_at ON device_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_device_failed_attempts_client_id ON device_failed_attempts(client_id);
CREATE INDEX IF NOT EXISTS idx_device_failed_attempts_attempted_at ON device_failed_attempts(attempted_at);
CREATE INDEX IF NOT EXISTS idx_device_code_consumption_user_code ON device_code_consumption(user_code);

-- Enable RLS
ALTER TABLE device_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_failed_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_code_consumption ENABLE ROW LEVEL SECURITY;

-- RLS Policies: Public can read/insert their own sessions for device flow
CREATE POLICY "Public can insert device sessions" ON device_sessions
  FOR INSERT WITH CHECK (TRUE);

CREATE POLICY "Authenticated users can view device sessions they approved" ON device_sessions
  FOR SELECT USING (approved_by = auth.uid() OR approved_by IS NULL);

-- Service role can manage all device data
CREATE POLICY "Service role can manage device sessions" ON device_sessions
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');

CREATE POLICY "Service role can manage device failed attempts" ON device_failed_attempts
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');

CREATE POLICY "Service role can manage device code consumption" ON device_code_consumption
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');
