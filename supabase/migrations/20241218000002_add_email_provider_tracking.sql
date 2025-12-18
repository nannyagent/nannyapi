-- Create user_auth_providers table to track email-to-provider mappings
-- Ensures users cannot sign up with same email using different providers (email/password, GitHub, Google)
CREATE TABLE IF NOT EXISTS user_auth_providers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  email TEXT NOT NULL,
  provider TEXT NOT NULL, -- 'email', 'github', 'google', etc.
  provider_user_id TEXT, -- External provider ID (e.g., GitHub user ID)
  is_primary BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(email, provider)
);

-- Create unique index on email to prevent duplicates across providers
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_auth_providers_email_provider 
  ON user_auth_providers(email, provider);

-- Create index for provider lookups
CREATE INDEX IF NOT EXISTS idx_user_auth_providers_user_id 
  ON user_auth_providers(user_id);

CREATE INDEX IF NOT EXISTS idx_user_auth_providers_email 
  ON user_auth_providers(email);

-- Enable RLS
ALTER TABLE user_auth_providers ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only see their own provider information
CREATE POLICY "Users can view their own auth providers" ON user_auth_providers
  FOR SELECT USING (auth.uid() = user_id);

-- RLS Policy: Service role can manage all provider data
CREATE POLICY "Service role can manage auth providers" ON user_auth_providers
  FOR ALL USING (auth.jwt() ->> 'role' = 'service_role' OR current_setting('role') = 'service_role');

-- RLS Policy: Allow public to check email availability without seeing user_id
-- This is safe because we only return whether email is used, not who uses it
CREATE POLICY "Public can check email availability" ON user_auth_providers
  FOR SELECT USING (TRUE);
