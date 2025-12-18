-- Remove redundant columns from agent_metrics table
-- These columns are already captured in the 'extra' or more specialized JSONB fields

ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "ip_address";
ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "location";
ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "agent_version";
ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "kernel_version";
ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "device_fingerprint";
