-- Clean up agent_metrics table: remove all redundant columns
-- Keep only: agent_id, recorded_at, cpu_percent, memory_mb, disk_percent, network_in_kbps, network_out_kbps
-- Keep JSONB fields: load_averages, os_info, filesystem_info, block_devices, extra

ALTER TABLE "public"."agent_metrics" DROP COLUMN IF EXISTS "network_stats";
