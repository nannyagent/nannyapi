-- Remove unused columns from agents table
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "oauth_client_id";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "oauth_token_expires_at";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "public_key";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "registered_ip";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "ip_address";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "location";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "os_version";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "version";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "kernel_version";
ALTER TABLE "public"."agents" DROP COLUMN IF EXISTS "timeline";
