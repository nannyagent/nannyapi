-- Clean up agents table: remove redundant/unused columns
-- Already removed via earlier migration: oauth_client_id, oauth_token_expires_at, public_key, registered_ip, ip_address, location, os_version, version, kernel_version, timeline
-- Remaining table should only have: id, owner, name, fingerprint, status, last_seen, created_at, metadata, websocket_connected, websocket_connected_at, websocket_disconnected_at

-- This migration is a marker that agents table cleanup is complete
-- All redundant columns have been removed in previous migrations (20251218160000_remove_unused_agent_columns.sql)
