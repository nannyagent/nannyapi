create extension if not exists "pg_cron" with schema "pg_catalog";

drop extension if exists "pg_net";

create extension if not exists "pg_net" with schema "public";

create sequence "public"."investigations_id_seq";

drop policy "Service role can manage lockouts" on "public"."account_lockout";

drop policy "Users can view their own lockout status" on "public"."account_lockout";

drop policy "Service role can manage device code consumption" on "public"."device_code_consumption";

drop policy "Service role can manage device failed attempts" on "public"."device_failed_attempts";

drop policy "Authenticated users can view device sessions they approved" on "public"."device_sessions";

drop policy "Public can insert device sessions" on "public"."device_sessions";

drop policy "Service role can manage device sessions" on "public"."device_sessions";

drop policy "Service role can manage MFA lockouts" on "public"."mfa_lockout";

drop policy "Users can view their own MFA lockout" on "public"."mfa_lockout";

drop policy "Service role can manage password attempts" on "public"."password_change_attempts";

drop policy "Users can view their own password attempts" on "public"."password_change_attempts";

drop policy "Service role can manage password history" on "public"."password_change_history";

drop policy "Users can view their own password history" on "public"."password_change_history";

drop policy "Public can read system config" on "public"."system_config";

drop policy "Public can check email availability" on "public"."user_auth_providers";

drop policy "Service role can manage auth providers" on "public"."user_auth_providers";

drop policy "Users can view their own auth providers" on "public"."user_auth_providers";

drop policy "Service role can manage backup code usage" on "public"."user_mfa_backup_codes_used";

drop policy "Users can view their own backup code usage" on "public"."user_mfa_backup_codes_used";

drop policy "Service role can manage MFA failed attempts" on "public"."user_mfa_failed_attempts";

drop policy "Users can view their own MFA failed attempts" on "public"."user_mfa_failed_attempts";

drop policy "Service role can manage MFA settings" on "public"."user_mfa_settings";

drop policy "Users can view their own MFA settings" on "public"."user_mfa_settings";

revoke delete on table "public"."device_code_consumption" from "anon";

revoke insert on table "public"."device_code_consumption" from "anon";

revoke references on table "public"."device_code_consumption" from "anon";

revoke select on table "public"."device_code_consumption" from "anon";

revoke trigger on table "public"."device_code_consumption" from "anon";

revoke truncate on table "public"."device_code_consumption" from "anon";

revoke update on table "public"."device_code_consumption" from "anon";

revoke delete on table "public"."device_code_consumption" from "authenticated";

revoke insert on table "public"."device_code_consumption" from "authenticated";

revoke references on table "public"."device_code_consumption" from "authenticated";

revoke select on table "public"."device_code_consumption" from "authenticated";

revoke trigger on table "public"."device_code_consumption" from "authenticated";

revoke truncate on table "public"."device_code_consumption" from "authenticated";

revoke update on table "public"."device_code_consumption" from "authenticated";

revoke delete on table "public"."device_code_consumption" from "service_role";

revoke insert on table "public"."device_code_consumption" from "service_role";

revoke references on table "public"."device_code_consumption" from "service_role";

revoke select on table "public"."device_code_consumption" from "service_role";

revoke trigger on table "public"."device_code_consumption" from "service_role";

revoke truncate on table "public"."device_code_consumption" from "service_role";

revoke update on table "public"."device_code_consumption" from "service_role";

revoke delete on table "public"."device_failed_attempts" from "anon";

revoke insert on table "public"."device_failed_attempts" from "anon";

revoke references on table "public"."device_failed_attempts" from "anon";

revoke select on table "public"."device_failed_attempts" from "anon";

revoke trigger on table "public"."device_failed_attempts" from "anon";

revoke truncate on table "public"."device_failed_attempts" from "anon";

revoke update on table "public"."device_failed_attempts" from "anon";

revoke delete on table "public"."device_failed_attempts" from "authenticated";

revoke insert on table "public"."device_failed_attempts" from "authenticated";

revoke references on table "public"."device_failed_attempts" from "authenticated";

revoke select on table "public"."device_failed_attempts" from "authenticated";

revoke trigger on table "public"."device_failed_attempts" from "authenticated";

revoke truncate on table "public"."device_failed_attempts" from "authenticated";

revoke update on table "public"."device_failed_attempts" from "authenticated";

revoke delete on table "public"."device_failed_attempts" from "service_role";

revoke insert on table "public"."device_failed_attempts" from "service_role";

revoke references on table "public"."device_failed_attempts" from "service_role";

revoke select on table "public"."device_failed_attempts" from "service_role";

revoke trigger on table "public"."device_failed_attempts" from "service_role";

revoke truncate on table "public"."device_failed_attempts" from "service_role";

revoke update on table "public"."device_failed_attempts" from "service_role";

revoke delete on table "public"."mfa_lockout" from "anon";

revoke insert on table "public"."mfa_lockout" from "anon";

revoke references on table "public"."mfa_lockout" from "anon";

revoke select on table "public"."mfa_lockout" from "anon";

revoke trigger on table "public"."mfa_lockout" from "anon";

revoke truncate on table "public"."mfa_lockout" from "anon";

revoke update on table "public"."mfa_lockout" from "anon";

revoke delete on table "public"."mfa_lockout" from "authenticated";

revoke insert on table "public"."mfa_lockout" from "authenticated";

revoke references on table "public"."mfa_lockout" from "authenticated";

revoke select on table "public"."mfa_lockout" from "authenticated";

revoke trigger on table "public"."mfa_lockout" from "authenticated";

revoke truncate on table "public"."mfa_lockout" from "authenticated";

revoke update on table "public"."mfa_lockout" from "authenticated";

revoke delete on table "public"."mfa_lockout" from "service_role";

revoke insert on table "public"."mfa_lockout" from "service_role";

revoke references on table "public"."mfa_lockout" from "service_role";

revoke select on table "public"."mfa_lockout" from "service_role";

revoke trigger on table "public"."mfa_lockout" from "service_role";

revoke truncate on table "public"."mfa_lockout" from "service_role";

revoke update on table "public"."mfa_lockout" from "service_role";

revoke delete on table "public"."user_auth_providers" from "anon";

revoke insert on table "public"."user_auth_providers" from "anon";

revoke references on table "public"."user_auth_providers" from "anon";

revoke select on table "public"."user_auth_providers" from "anon";

revoke trigger on table "public"."user_auth_providers" from "anon";

revoke truncate on table "public"."user_auth_providers" from "anon";

revoke update on table "public"."user_auth_providers" from "anon";

revoke delete on table "public"."user_auth_providers" from "authenticated";

revoke insert on table "public"."user_auth_providers" from "authenticated";

revoke references on table "public"."user_auth_providers" from "authenticated";

revoke select on table "public"."user_auth_providers" from "authenticated";

revoke trigger on table "public"."user_auth_providers" from "authenticated";

revoke truncate on table "public"."user_auth_providers" from "authenticated";

revoke update on table "public"."user_auth_providers" from "authenticated";

revoke delete on table "public"."user_auth_providers" from "service_role";

revoke insert on table "public"."user_auth_providers" from "service_role";

revoke references on table "public"."user_auth_providers" from "service_role";

revoke select on table "public"."user_auth_providers" from "service_role";

revoke trigger on table "public"."user_auth_providers" from "service_role";

revoke truncate on table "public"."user_auth_providers" from "service_role";

revoke update on table "public"."user_auth_providers" from "service_role";

revoke delete on table "public"."user_mfa_failed_attempts" from "anon";

revoke insert on table "public"."user_mfa_failed_attempts" from "anon";

revoke references on table "public"."user_mfa_failed_attempts" from "anon";

revoke select on table "public"."user_mfa_failed_attempts" from "anon";

revoke trigger on table "public"."user_mfa_failed_attempts" from "anon";

revoke truncate on table "public"."user_mfa_failed_attempts" from "anon";

revoke update on table "public"."user_mfa_failed_attempts" from "anon";

revoke delete on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke insert on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke references on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke select on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke trigger on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke truncate on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke update on table "public"."user_mfa_failed_attempts" from "authenticated";

revoke delete on table "public"."user_mfa_failed_attempts" from "service_role";

revoke insert on table "public"."user_mfa_failed_attempts" from "service_role";

revoke references on table "public"."user_mfa_failed_attempts" from "service_role";

revoke select on table "public"."user_mfa_failed_attempts" from "service_role";

revoke trigger on table "public"."user_mfa_failed_attempts" from "service_role";

revoke truncate on table "public"."user_mfa_failed_attempts" from "service_role";

revoke update on table "public"."user_mfa_failed_attempts" from "service_role";

alter table "public"."device_code_consumption" drop constraint "device_code_consumption_agent_id_fkey";

alter table "public"."device_code_consumption" drop constraint "device_code_consumption_user_code_key";

alter table "public"."device_sessions" drop constraint "device_sessions_device_id_key";

alter table "public"."mfa_lockout" drop constraint "mfa_lockout_user_id_fkey";

alter table "public"."user_auth_providers" drop constraint "user_auth_providers_email_provider_key";

alter table "public"."user_auth_providers" drop constraint "user_auth_providers_user_id_fkey";

alter table "public"."user_mfa_failed_attempts" drop constraint "user_mfa_failed_attempts_user_id_fkey";

alter table "public"."device_sessions" drop constraint "device_sessions_approved_by_fkey";

alter table "public"."password_change_attempts" drop constraint "password_change_attempts_user_id_fkey";

alter table "public"."device_code_consumption" drop constraint "device_code_consumption_pkey";

alter table "public"."device_failed_attempts" drop constraint "device_failed_attempts_pkey";

alter table "public"."mfa_lockout" drop constraint "mfa_lockout_pkey";

alter table "public"."user_auth_providers" drop constraint "user_auth_providers_pkey";

alter table "public"."user_mfa_failed_attempts" drop constraint "user_mfa_failed_attempts_pkey";

drop index if exists "public"."device_code_consumption_pkey";

drop index if exists "public"."device_code_consumption_user_code_key";

drop index if exists "public"."device_failed_attempts_pkey";

drop index if exists "public"."device_sessions_device_id_key";

drop index if exists "public"."idx_device_code_consumption_user_code";

drop index if exists "public"."idx_device_failed_attempts_attempted_at";

drop index if exists "public"."idx_device_failed_attempts_client_id";

drop index if exists "public"."idx_device_sessions_expires_at";

drop index if exists "public"."idx_device_sessions_status";

drop index if exists "public"."idx_device_sessions_user_code";

drop index if exists "public"."idx_mfa_lockout_locked_until";

drop index if exists "public"."idx_mfa_lockout_user_id";

drop index if exists "public"."idx_user_auth_providers_email";

drop index if exists "public"."idx_user_auth_providers_email_provider";

drop index if exists "public"."idx_user_auth_providers_user_id";

drop index if exists "public"."idx_user_mfa_backup_codes_used_user_id";

drop index if exists "public"."idx_user_mfa_failed_attempts_failed_at";

drop index if exists "public"."idx_user_mfa_failed_attempts_user_id";

drop index if exists "public"."mfa_lockout_pkey";

drop index if exists "public"."user_auth_providers_email_provider_key";

drop index if exists "public"."user_auth_providers_pkey";

drop index if exists "public"."user_mfa_failed_attempts_pkey";

drop table "public"."device_code_consumption";

drop table "public"."device_failed_attempts";

drop table "public"."mfa_lockout";

drop table "public"."user_auth_providers";

drop table "public"."user_mfa_failed_attempts";


  create table "public"."activities" (
    "id" uuid not null default gen_random_uuid(),
    "user_id" uuid not null,
    "agent_id" uuid,
    "activity_type" text not null,
    "summary" text not null,
    "metadata" jsonb default '{}'::jsonb,
    "created_at" timestamp with time zone not null default now()
      );



  create table "public"."agent_device_codes" (
    "device_code" uuid not null default gen_random_uuid(),
    "agent_id" uuid,
    "created_at" timestamp with time zone default now(),
    "expires_at" timestamp with time zone default (now() + '00:15:00'::interval),
    "metadata" jsonb default '{}'::jsonb,
    "authorized" boolean default false,
    "authorized_by" uuid,
    "id" uuid default gen_random_uuid(),
    "user_code" character varying(8),
    "client_id" text default 'unknown'::text,
    "scope" text default 'agent:register'::text,
    "authorized_at" timestamp with time zone,
    "user_id" uuid,
    "consumed" boolean default false,
    "stored_agent_id" uuid,
    "stored_access_token" text,
    "stored_refresh_token" text
      );


alter table "public"."agent_device_codes" enable row level security;


  create table "public"."agent_metrics" (
    "agent_id" uuid not null,
    "recorded_at" timestamp with time zone not null default now(),
    "cpu_percent" numeric(5,2),
    "memory_mb" integer,
    "disk_percent" numeric(5,2),
    "network_in_kbps" numeric(10,2),
    "network_out_kbps" numeric(10,2),
    "extra" jsonb default '{}'::jsonb,
    "ip_address" inet,
    "location" text default 'Global'::text,
    "agent_version" text default 'dev'::text,
    "os_info" jsonb default '{}'::jsonb,
    "kernel_version" text,
    "filesystem_info" jsonb default '{}'::jsonb,
    "block_devices" jsonb default '{}'::jsonb,
    "device_fingerprint" text,
    "load_averages" jsonb default '{}'::jsonb,
    "network_stats" jsonb
      );



  create table "public"."agent_packages" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "storage_path" text not null,
    "distro" text not null,
    "package_count" integer,
    "file_size_bytes" integer,
    "collected_at" timestamp with time zone not null default now(),
    "created_at" timestamp with time zone not null default now(),
    "updated_at" timestamp with time zone not null default now()
      );


alter table "public"."agent_packages" enable row level security;


  create table "public"."agent_registrations" (
    "id" uuid not null,
    "user_id" uuid not null,
    "device_code" uuid,
    "agent_name" text not null,
    "agent_type" text not null default 'generic'::text,
    "metadata" jsonb default '{}'::jsonb,
    "status" text not null default 'active'::text,
    "created_at" timestamp with time zone not null default now(),
    "updated_at" timestamp with time zone not null default now()
      );


alter table "public"."agent_registrations" enable row level security;


  create table "public"."agent_tokens" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "user_id" uuid not null,
    "token_hash" text not null,
    "expires_at" timestamp with time zone not null,
    "revoked" boolean not null default false,
    "revoked_at" timestamp with time zone,
    "created_at" timestamp with time zone not null default now(),
    "issued_at" timestamp with time zone default now(),
    "refresh_token_hash" text
      );


alter table "public"."agent_tokens" enable row level security;


  create table "public"."agents" (
    "id" uuid not null default gen_random_uuid(),
    "owner" uuid not null,
    "name" text,
    "fingerprint" text,
    "status" text not null default 'pending'::text,
    "last_seen" timestamp with time zone,
    "created_at" timestamp with time zone default now(),
    "metadata" jsonb default '{}'::jsonb,
    "websocket_connected" boolean default false,
    "websocket_connected_at" timestamp with time zone,
    "websocket_disconnected_at" timestamp with time zone,
    "oauth_client_id" text,
    "oauth_token_expires_at" timestamp with time zone,
    "public_key" text,
    "registered_ip" text,
    "ip_address" text,
    "location" text,
    "version" text,
    "os_version" text,
    "kernel_version" text,
    "timeline" jsonb
      );


alter table "public"."agents" enable row level security;


  create table "public"."installed_packages" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "package_name" text not null,
    "package_version" text not null,
    "package_manager" text not null,
    "architecture" text,
    "recorded_at" timestamp with time zone not null default now(),
    "metadata" jsonb default '{}'::jsonb
      );


alter table "public"."installed_packages" enable row level security;


  create table "public"."investigations" (
    "id" integer not null default nextval('public.investigations_id_seq'::regclass),
    "investigation_id" character varying(255) not null,
    "issue" text not null,
    "application_group" character varying(255),
    "priority" character varying(20),
    "initiated_by" character varying(255),
    "target_agents" text[],
    "status" character varying(50) default 'initiated'::character varying,
    "holistic_analysis" text,
    "created_at" timestamp with time zone default now(),
    "completed_at" timestamp with time zone,
    "agent_id" uuid,
    "initiated_at" timestamp with time zone default now(),
    "updated_at" timestamp with time zone default now(),
    "tensorzero_response" text,
    "episode_id" text,
    "metadata" jsonb default '{}'::jsonb
      );


alter table "public"."investigations" enable row level security;


  create table "public"."package_exceptions" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "package_name" text not null,
    "reason" text,
    "created_at" timestamp with time zone default now(),
    "created_by" uuid,
    "expires_at" timestamp with time zone,
    "is_active" boolean default true
      );


alter table "public"."package_exceptions" enable row level security;


  create table "public"."package_fetch_commands" (
    "id" uuid not null default gen_random_uuid(),
    "distro" text not null,
    "command" text not null,
    "package_manager" text not null,
    "description" text,
    "created_at" timestamp with time zone not null default now(),
    "updated_at" timestamp with time zone not null default now()
      );



  create table "public"."patch_executions" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "script_id" uuid,
    "execution_type" text not null,
    "status" text not null default 'pending'::text,
    "command" text not null,
    "exit_code" integer,
    "stdout_storage_path" text,
    "stderr_storage_path" text,
    "started_at" timestamp with time zone default now(),
    "completed_at" timestamp with time zone,
    "error_message" text,
    "triggered_by" text,
    "should_reboot" boolean default false,
    "rebooted_at" timestamp with time zone,
    "logs_path" text
      );


alter table "public"."patch_executions" enable row level security;


  create table "public"."patch_management_schedules" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "user_id" uuid not null,
    "cron_schedule" text not null default '0 0 * * *'::text,
    "enabled" boolean not null default true,
    "last_run_at" timestamp with time zone,
    "next_run_at" timestamp with time zone,
    "created_at" timestamp with time zone not null default now(),
    "updated_at" timestamp with time zone not null default now()
      );


alter table "public"."patch_management_schedules" enable row level security;


  create table "public"."patch_scripts" (
    "id" uuid not null default gen_random_uuid(),
    "name" text not null,
    "description" text,
    "script_storage_path" text not null,
    "os_family" text not null,
    "os_platform" text,
    "major_version" text,
    "minor_version" text,
    "package_manager" text not null,
    "script_type" text not null default 'update'::text,
    "created_at" timestamp with time zone default now(),
    "updated_at" timestamp with time zone default now(),
    "created_by" uuid,
    "is_active" boolean default true
      );


alter table "public"."patch_scripts" enable row level security;


  create table "public"."patch_tasks" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "command" text not null,
    "status" text not null default 'pending'::text,
    "exit_code" integer,
    "error_message" text,
    "created_at" timestamp with time zone default now(),
    "started_at" timestamp with time zone,
    "completed_at" timestamp with time zone,
    "stdout_storage_path" text,
    "stderr_storage_path" text
      );


alter table "public"."patch_tasks" enable row level security;


  create table "public"."pending_investigations" (
    "id" uuid not null default gen_random_uuid(),
    "investigation_id" character varying not null,
    "agent_id" uuid not null,
    "diagnostic_payload" jsonb not null,
    "episode_id" text,
    "status" text default 'pending'::text,
    "created_at" timestamp with time zone default now(),
    "started_at" timestamp with time zone,
    "completed_at" timestamp with time zone,
    "command_results" jsonb,
    "error_message" text,
    "task_type" text default 'investigation'::text
      );


alter table "public"."pending_investigations" enable row level security;


  create table "public"."pricing_plans" (
    "plan_id" text not null,
    "name" text,
    "agent_limit" integer default 1,
    "monthly_price_cents" integer default 0,
    "api_calls_per_min" integer default 60,
    "created_at" timestamp with time zone default now(),
    "slug" text,
    "max_agents" integer,
    "core_api_access" boolean default false,
    "monitoring" text,
    "support" text,
    "api_calls_per_day" bigint,
    "tokens_per_day" bigint,
    "data_retention_days" integer,
    "advanced_security" boolean default false,
    "priority_support" boolean default false,
    "custom_agents" boolean default false,
    "features" jsonb default '{}'::jsonb
      );


alter table "public"."pricing_plans" enable row level security;


  create table "public"."scheduled_patches" (
    "id" uuid not null default gen_random_uuid(),
    "agent_id" uuid not null,
    "script_id" uuid,
    "cron_expression" text not null,
    "execution_type" text not null,
    "is_active" boolean default true,
    "last_run_at" timestamp with time zone,
    "next_run_at" timestamp with time zone,
    "created_at" timestamp with time zone default now(),
    "created_by" uuid
      );


alter table "public"."scheduled_patches" enable row level security;

alter table "public"."account_lockout" alter column "created_at" set not null;

alter table "public"."account_lockout" disable row level security;

alter table "public"."device_sessions" add column "agent_email" text;

alter table "public"."device_sessions" add column "agent_password" text;

alter table "public"."device_sessions" add column "agent_user_id" uuid;

alter table "public"."device_sessions" add column "authorized" boolean default false;

alter table "public"."device_sessions" add column "authorized_at" timestamp with time zone;

alter table "public"."device_sessions" add column "authorized_by" uuid;

alter table "public"."device_sessions" add column "consume_ip" inet;

alter table "public"."device_sessions" add column "consumed_at" timestamp with time zone;

alter table "public"."device_sessions" add column "device_name" text;

alter table "public"."device_sessions" add column "last_polled_at" timestamp with time zone;

alter table "public"."device_sessions" add column "poll_count" integer not null default 0;

alter table "public"."device_sessions" add column "request_ip" inet;

alter table "public"."device_sessions" add column "stored_agent_id" uuid;

alter table "public"."device_sessions" alter column "created_at" set not null;

alter table "public"."device_sessions" alter column "device_id" drop not null;

alter table "public"."device_sessions" alter column "interval_seconds" set not null;

alter table "public"."device_sessions" alter column "requested_scopes" drop default;

alter table "public"."device_sessions" alter column "requested_scopes" set data type jsonb using to_jsonb("requested_scopes");

alter table "public"."device_sessions" alter column "status" set not null;

alter table "public"."device_sessions" disable row level security;

alter table "public"."password_change_attempts" drop column "created_at";

alter table "public"."password_change_attempts" alter column "attempted_at" set not null;

alter table "public"."password_change_attempts" alter column "user_id" drop not null;

alter table "public"."password_change_attempts" disable row level security;

alter table "public"."password_change_history" drop column "created_at";

alter table "public"."password_change_history" alter column "changed_at" set not null;

alter table "public"."password_change_history" alter column "ip_address" drop not null;

alter table "public"."password_change_history" disable row level security;

alter table "public"."system_config" add column "category" text;

alter table "public"."system_config" add column "data_type" text;

alter table "public"."system_config" add column "is_sensitive" boolean default false;

alter table "public"."system_config" alter column "created_at" set not null;

alter table "public"."system_config" alter column "updated_at" set not null;

alter table "public"."system_config" disable row level security;

alter table "public"."user_mfa_backup_codes_used" alter column "backup_code_index" set not null;

alter table "public"."user_mfa_backup_codes_used" alter column "code_hash" drop not null;

alter table "public"."user_mfa_backup_codes_used" alter column "code_hash" set data type character varying(255) using "code_hash"::character varying(255);

alter table "public"."user_mfa_backup_codes_used" alter column "created_at" set not null;

alter table "public"."user_mfa_backup_codes_used" alter column "ip_address" set data type inet using "ip_address"::inet;

alter table "public"."user_mfa_backup_codes_used" alter column "used_at" set not null;

alter table "public"."user_mfa_backup_codes_used" alter column "used_for_login" set default true;

alter table "public"."user_mfa_backup_codes_used" alter column "used_for_login" set not null;

alter table "public"."user_mfa_settings" drop column "backup_codes_hash";

alter table "public"."user_mfa_settings" add column "backup_codes" text[] not null;

alter table "public"."user_mfa_settings" add column "mfa_enabled_at" timestamp with time zone;

alter table "public"."user_mfa_settings" alter column "created_at" set not null;

alter table "public"."user_mfa_settings" alter column "mfa_enabled" set not null;

alter table "public"."user_mfa_settings" alter column "totp_secret" set not null;

alter table "public"."user_mfa_settings" alter column "updated_at" set not null;

alter sequence "public"."investigations_id_seq" owned by "public"."investigations"."id";

CREATE UNIQUE INDEX activities_pkey ON public.activities USING btree (id);

CREATE UNIQUE INDEX agent_device_codes_pkey ON public.agent_device_codes USING btree (device_code);

CREATE UNIQUE INDEX agent_device_codes_user_code_key ON public.agent_device_codes USING btree (user_code);

CREATE UNIQUE INDEX agent_metrics_pkey ON public.agent_metrics USING btree (agent_id);

CREATE UNIQUE INDEX agent_packages_pkey ON public.agent_packages USING btree (id);

CREATE UNIQUE INDEX agent_registrations_pkey ON public.agent_registrations USING btree (id);

CREATE UNIQUE INDEX agent_tokens_pkey ON public.agent_tokens USING btree (id);

CREATE UNIQUE INDEX agent_tokens_token_hash_key ON public.agent_tokens USING btree (token_hash);

CREATE UNIQUE INDEX agents_fingerprint_key ON public.agents USING btree (fingerprint);

CREATE UNIQUE INDEX agents_pkey ON public.agents USING btree (id);

CREATE UNIQUE INDEX device_sessions_user_code_unique ON public.device_sessions USING btree (user_code);

CREATE INDEX idx_agent_device_codes_user_code ON public.agent_device_codes USING btree (user_code);

CREATE INDEX idx_agent_metrics_agent_recorded_at ON public.agent_metrics USING btree (agent_id, recorded_at DESC);

CREATE INDEX idx_agent_packages_agent_id ON public.agent_packages USING btree (agent_id);

CREATE INDEX idx_agent_packages_collected_at ON public.agent_packages USING btree (collected_at DESC);

CREATE INDEX idx_agent_registrations_user_id ON public.agent_registrations USING btree (user_id);

CREATE INDEX idx_agent_tokens_agent_id ON public.agent_tokens USING btree (agent_id);

CREATE INDEX idx_agent_tokens_token_hash ON public.agent_tokens USING btree (token_hash);

CREATE INDEX idx_agents_last_seen ON public.agents USING btree (last_seen);

CREATE INDEX idx_agents_owner ON public.agents USING btree (owner);

CREATE INDEX idx_agents_user_id ON public.agents USING btree (owner);

CREATE INDEX idx_backup_codes_used_code_index ON public.user_mfa_backup_codes_used USING btree (user_id, backup_code_index);

CREATE INDEX idx_backup_codes_used_user_id ON public.user_mfa_backup_codes_used USING btree (user_id);

CREATE INDEX idx_device_sessions_approved_by ON public.device_sessions USING btree (approved_by);

CREATE INDEX idx_device_sessions_device_code_hash ON public.device_sessions USING btree (device_code_hash);

CREATE INDEX idx_device_sessions_status_expires ON public.device_sessions USING btree (status, expires_at);

CREATE INDEX idx_installed_packages_agent_id ON public.installed_packages USING btree (agent_id);

CREATE INDEX idx_installed_packages_recorded_at ON public.installed_packages USING btree (recorded_at DESC);

CREATE INDEX idx_investigations_created_at ON public.investigations USING btree (created_at);

CREATE INDEX idx_investigations_status ON public.investigations USING btree (status);

CREATE INDEX idx_package_exceptions_agent ON public.package_exceptions USING btree (agent_id, is_active);

CREATE INDEX idx_password_change_attempts_ip_address ON public.password_change_attempts USING btree (ip_address);

CREATE INDEX idx_password_change_history_changed_at ON public.password_change_history USING btree (changed_at);

CREATE INDEX idx_patch_executions_agent_status ON public.patch_executions USING btree (agent_id, status, started_at DESC);

CREATE INDEX idx_patch_schedules_agent_id ON public.patch_management_schedules USING btree (agent_id);

CREATE INDEX idx_patch_schedules_enabled ON public.patch_management_schedules USING btree (enabled) WHERE (enabled = true);

CREATE INDEX idx_patch_scripts_os_family ON public.patch_scripts USING btree (os_family, is_active);

CREATE INDEX idx_patch_scripts_os_platform ON public.patch_scripts USING btree (os_platform, is_active);

CREATE INDEX idx_patch_tasks_agent_status ON public.patch_tasks USING btree (agent_id, status);

CREATE INDEX idx_patch_tasks_created_at ON public.patch_tasks USING btree (created_at DESC);

CREATE INDEX idx_pending_investigations_agent_id ON public.pending_investigations USING btree (agent_id);

CREATE INDEX idx_pending_investigations_created_at ON public.pending_investigations USING btree (created_at DESC);

CREATE INDEX idx_pending_investigations_status ON public.pending_investigations USING btree (status);

CREATE INDEX idx_scheduled_patches_agent_active ON public.scheduled_patches USING btree (agent_id, is_active);

CREATE INDEX idx_system_config_category ON public.system_config USING btree (category);

CREATE INDEX idx_system_config_key ON public.system_config USING btree (key);

CREATE UNIQUE INDEX installed_packages_pkey ON public.installed_packages USING btree (id);

CREATE UNIQUE INDEX investigations_investigation_id_key ON public.investigations USING btree (investigation_id);

CREATE UNIQUE INDEX investigations_pkey ON public.investigations USING btree (id);

CREATE UNIQUE INDEX package_exceptions_agent_id_package_name_key ON public.package_exceptions USING btree (agent_id, package_name);

CREATE UNIQUE INDEX package_exceptions_pkey ON public.package_exceptions USING btree (id);

CREATE UNIQUE INDEX package_fetch_commands_distro_key ON public.package_fetch_commands USING btree (distro);

CREATE UNIQUE INDEX package_fetch_commands_pkey ON public.package_fetch_commands USING btree (id);

CREATE UNIQUE INDEX patch_executions_pkey ON public.patch_executions USING btree (id);

CREATE UNIQUE INDEX patch_management_schedules_pkey ON public.patch_management_schedules USING btree (id);

CREATE UNIQUE INDEX patch_scripts_pkey ON public.patch_scripts USING btree (id);

CREATE UNIQUE INDEX patch_tasks_pkey ON public.patch_tasks USING btree (id);

CREATE UNIQUE INDEX pending_investigations_pkey ON public.pending_investigations USING btree (id);

CREATE UNIQUE INDEX pricing_plans_pkey ON public.pricing_plans USING btree (plan_id);

CREATE UNIQUE INDEX pricing_plans_slug_key ON public.pricing_plans USING btree (slug);

CREATE UNIQUE INDEX scheduled_patches_pkey ON public.scheduled_patches USING btree (id);

CREATE UNIQUE INDEX unique_agent_package ON public.installed_packages USING btree (agent_id, package_name);

CREATE UNIQUE INDEX unique_agent_packages ON public.agent_packages USING btree (agent_id);

CREATE UNIQUE INDEX unique_agent_schedule ON public.patch_management_schedules USING btree (agent_id);

alter table "public"."activities" add constraint "activities_pkey" PRIMARY KEY using index "activities_pkey";

alter table "public"."agent_device_codes" add constraint "agent_device_codes_pkey" PRIMARY KEY using index "agent_device_codes_pkey";

alter table "public"."agent_metrics" add constraint "agent_metrics_pkey" PRIMARY KEY using index "agent_metrics_pkey";

alter table "public"."agent_packages" add constraint "agent_packages_pkey" PRIMARY KEY using index "agent_packages_pkey";

alter table "public"."agent_registrations" add constraint "agent_registrations_pkey" PRIMARY KEY using index "agent_registrations_pkey";

alter table "public"."agent_tokens" add constraint "agent_tokens_pkey" PRIMARY KEY using index "agent_tokens_pkey";

alter table "public"."agents" add constraint "agents_pkey" PRIMARY KEY using index "agents_pkey";

alter table "public"."installed_packages" add constraint "installed_packages_pkey" PRIMARY KEY using index "installed_packages_pkey";

alter table "public"."investigations" add constraint "investigations_pkey" PRIMARY KEY using index "investigations_pkey";

alter table "public"."package_exceptions" add constraint "package_exceptions_pkey" PRIMARY KEY using index "package_exceptions_pkey";

alter table "public"."package_fetch_commands" add constraint "package_fetch_commands_pkey" PRIMARY KEY using index "package_fetch_commands_pkey";

alter table "public"."patch_executions" add constraint "patch_executions_pkey" PRIMARY KEY using index "patch_executions_pkey";

alter table "public"."patch_management_schedules" add constraint "patch_management_schedules_pkey" PRIMARY KEY using index "patch_management_schedules_pkey";

alter table "public"."patch_scripts" add constraint "patch_scripts_pkey" PRIMARY KEY using index "patch_scripts_pkey";

alter table "public"."patch_tasks" add constraint "patch_tasks_pkey" PRIMARY KEY using index "patch_tasks_pkey";

alter table "public"."pending_investigations" add constraint "pending_investigations_pkey" PRIMARY KEY using index "pending_investigations_pkey";

alter table "public"."pricing_plans" add constraint "pricing_plans_pkey" PRIMARY KEY using index "pricing_plans_pkey";

alter table "public"."scheduled_patches" add constraint "scheduled_patches_pkey" PRIMARY KEY using index "scheduled_patches_pkey";

alter table "public"."agent_device_codes" add constraint "agent_device_codes_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."agent_device_codes" validate constraint "agent_device_codes_agent_id_fkey";

alter table "public"."agent_device_codes" add constraint "agent_device_codes_user_code_key" UNIQUE using index "agent_device_codes_user_code_key";

alter table "public"."agent_device_codes" add constraint "agent_device_codes_user_id_fkey" FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE not valid;

alter table "public"."agent_device_codes" validate constraint "agent_device_codes_user_id_fkey";

alter table "public"."agent_metrics" add constraint "agent_metrics_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."agent_metrics" validate constraint "agent_metrics_agent_id_fkey";

alter table "public"."agent_packages" add constraint "fk_agent_packages" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."agent_packages" validate constraint "fk_agent_packages";

alter table "public"."agent_packages" add constraint "unique_agent_packages" UNIQUE using index "unique_agent_packages";

alter table "public"."agent_registrations" add constraint "agent_registrations_device_code_fkey" FOREIGN KEY (device_code) REFERENCES public.agent_device_codes(device_code) ON DELETE SET NULL not valid;

alter table "public"."agent_registrations" validate constraint "agent_registrations_device_code_fkey";

alter table "public"."agent_registrations" add constraint "agent_registrations_status_check" CHECK ((status = ANY (ARRAY['active'::text, 'inactive'::text, 'suspended'::text]))) not valid;

alter table "public"."agent_registrations" validate constraint "agent_registrations_status_check";

alter table "public"."agent_registrations" add constraint "agent_registrations_user_id_fkey" FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE not valid;

alter table "public"."agent_registrations" validate constraint "agent_registrations_user_id_fkey";

alter table "public"."agent_tokens" add constraint "agent_tokens_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agent_registrations(id) ON DELETE CASCADE not valid;

alter table "public"."agent_tokens" validate constraint "agent_tokens_agent_id_fkey";

alter table "public"."agent_tokens" add constraint "agent_tokens_token_hash_key" UNIQUE using index "agent_tokens_token_hash_key";

alter table "public"."agent_tokens" add constraint "agent_tokens_user_id_fkey" FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE not valid;

alter table "public"."agent_tokens" validate constraint "agent_tokens_user_id_fkey";

alter table "public"."agents" add constraint "agents_fingerprint_key" UNIQUE using index "agents_fingerprint_key";

alter table "public"."device_sessions" add constraint "device_sessions_agent_user_id_fkey" FOREIGN KEY (agent_user_id) REFERENCES auth.users(id) not valid;

alter table "public"."device_sessions" validate constraint "device_sessions_agent_user_id_fkey";

alter table "public"."device_sessions" add constraint "device_sessions_authorized_by_fkey" FOREIGN KEY (authorized_by) REFERENCES auth.users(id) not valid;

alter table "public"."device_sessions" validate constraint "device_sessions_authorized_by_fkey";

alter table "public"."device_sessions" add constraint "device_sessions_stored_agent_id_fkey" FOREIGN KEY (stored_agent_id) REFERENCES public.agents(id) not valid;

alter table "public"."device_sessions" validate constraint "device_sessions_stored_agent_id_fkey";

alter table "public"."device_sessions" add constraint "device_sessions_user_code_unique" UNIQUE using index "device_sessions_user_code_unique";

alter table "public"."installed_packages" add constraint "fk_agent_installed_packages" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."installed_packages" validate constraint "fk_agent_installed_packages";

alter table "public"."installed_packages" add constraint "unique_agent_package" UNIQUE using index "unique_agent_package";

alter table "public"."investigations" add constraint "investigations_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."investigations" validate constraint "investigations_agent_id_fkey";

alter table "public"."investigations" add constraint "investigations_investigation_id_key" UNIQUE using index "investigations_investigation_id_key";

alter table "public"."investigations" add constraint "investigations_priority_check" CHECK (((priority)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[]))) not valid;

alter table "public"."investigations" validate constraint "investigations_priority_check";

alter table "public"."package_exceptions" add constraint "package_exceptions_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."package_exceptions" validate constraint "package_exceptions_agent_id_fkey";

alter table "public"."package_exceptions" add constraint "package_exceptions_agent_id_package_name_key" UNIQUE using index "package_exceptions_agent_id_package_name_key";

alter table "public"."package_fetch_commands" add constraint "package_fetch_commands_distro_key" UNIQUE using index "package_fetch_commands_distro_key";

alter table "public"."patch_executions" add constraint "patch_executions_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."patch_executions" validate constraint "patch_executions_agent_id_fkey";

alter table "public"."patch_executions" add constraint "patch_executions_script_id_fkey" FOREIGN KEY (script_id) REFERENCES public.patch_scripts(id) ON DELETE SET NULL not valid;

alter table "public"."patch_executions" validate constraint "patch_executions_script_id_fkey";

alter table "public"."patch_management_schedules" add constraint "fk_agent_patch_schedule" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."patch_management_schedules" validate constraint "fk_agent_patch_schedule";

alter table "public"."patch_management_schedules" add constraint "fk_user_patch_schedule" FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE not valid;

alter table "public"."patch_management_schedules" validate constraint "fk_user_patch_schedule";

alter table "public"."patch_management_schedules" add constraint "unique_agent_schedule" UNIQUE using index "unique_agent_schedule";

alter table "public"."patch_tasks" add constraint "fk_patch_tasks_agent" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."patch_tasks" validate constraint "fk_patch_tasks_agent";

alter table "public"."patch_tasks" add constraint "patch_tasks_status_check" CHECK ((status = ANY (ARRAY['pending'::text, 'executing'::text, 'completed'::text, 'failed'::text]))) not valid;

alter table "public"."patch_tasks" validate constraint "patch_tasks_status_check";

alter table "public"."pending_investigations" add constraint "pending_investigations_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."pending_investigations" validate constraint "pending_investigations_agent_id_fkey";

alter table "public"."pending_investigations" add constraint "pending_investigations_status_check" CHECK ((status = ANY (ARRAY['pending'::text, 'executing'::text, 'completed'::text, 'failed'::text]))) not valid;

alter table "public"."pending_investigations" validate constraint "pending_investigations_status_check";

alter table "public"."pending_investigations" add constraint "pending_investigations_task_type_check" CHECK ((task_type = ANY (ARRAY['investigation'::text, 'patch_management'::text]))) not valid;

alter table "public"."pending_investigations" validate constraint "pending_investigations_task_type_check";

alter table "public"."pricing_plans" add constraint "pricing_plans_slug_key" UNIQUE using index "pricing_plans_slug_key";

alter table "public"."scheduled_patches" add constraint "scheduled_patches_agent_id_fkey" FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE not valid;

alter table "public"."scheduled_patches" validate constraint "scheduled_patches_agent_id_fkey";

alter table "public"."scheduled_patches" add constraint "scheduled_patches_script_id_fkey" FOREIGN KEY (script_id) REFERENCES public.patch_scripts(id) ON DELETE SET NULL not valid;

alter table "public"."scheduled_patches" validate constraint "scheduled_patches_script_id_fkey";

alter table "public"."system_config" add constraint "system_config_data_type_check" CHECK ((data_type = ANY (ARRAY['integer'::text, 'boolean'::text, 'string'::text, 'json'::text]))) not valid;

alter table "public"."system_config" validate constraint "system_config_data_type_check";

alter table "public"."device_sessions" add constraint "device_sessions_approved_by_fkey" FOREIGN KEY (approved_by) REFERENCES auth.users(id) not valid;

alter table "public"."device_sessions" validate constraint "device_sessions_approved_by_fkey";

alter table "public"."password_change_attempts" add constraint "password_change_attempts_user_id_fkey" FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE SET NULL not valid;

alter table "public"."password_change_attempts" validate constraint "password_change_attempts_user_id_fkey";

set check_function_bodies = off;

CREATE OR REPLACE FUNCTION public.delete_agent_cascade(p_agent_id uuid)
 RETURNS void
 LANGUAGE plpgsql
 SECURITY DEFINER
AS $function$
BEGIN
  -- Delete in the correct order to respect foreign key constraints
  
  -- Delete agent heartbeats
  DELETE FROM agent_heartbeats WHERE agent_id = p_agent_id;
  
  -- Delete agent metrics
  DELETE FROM agent_metrics WHERE agent_id = p_agent_id;
  
  -- Delete agent oauth tokens
  DELETE FROM agent_oauth_tokens WHERE agent_id = p_agent_id;
  
  -- Delete agent rate limits
  DELETE FROM agent_rate_limits WHERE agent_id = p_agent_id;
  
  -- Delete agent device codes
  DELETE FROM agent_device_codes WHERE agent_id = p_agent_id;
  
  -- Update device sessions to remove the reference (don't delete the session itself)
  UPDATE device_sessions 
  SET stored_agent_id = NULL 
  WHERE stored_agent_id = p_agent_id;
  
  -- Finally delete the agent itself
  DELETE FROM agents WHERE id = p_agent_id;
  
  -- Note: We don't delete from activities table as we want to keep audit trail
  
END;
$function$
;

CREATE OR REPLACE FUNCTION public.hash_token(input_text text)
 RETURNS text
 LANGUAGE sql
 STABLE
AS $function$
  SELECT encode(digest(input_text, 'sha256'), 'hex');
$function$
;

CREATE OR REPLACE FUNCTION public.insert_agent_metrics(p_agent_id uuid, p_cpu_percent numeric, p_memory_mb integer, p_disk_percent numeric, p_network_in_kbps numeric, p_network_out_kbps numeric, p_load1 numeric, p_load5 numeric, p_load15 numeric, p_extra jsonb)
 RETURNS void
 LANGUAGE plpgsql
 SECURITY DEFINER
AS $function$
BEGIN
  INSERT INTO public.agent_metrics(
    agent_id, cpu_percent, memory_mb, disk_percent, network_in_kbps, network_out_kbps, load1, load5, load15, extra
  ) VALUES (
    p_agent_id, p_cpu_percent, p_memory_mb, p_disk_percent, p_network_in_kbps, p_network_out_kbps, p_load1, p_load5, p_load15, COALESCE(p_extra, '{}'::jsonb)
  );

  UPDATE public.agents SET last_seen = now(), timeline = jsonb_set(COALESCE(timeline, '{}'::jsonb), '{last_metric}', to_jsonb(now()), true)
  WHERE id = p_agent_id;
END;
$function$
;

CREATE OR REPLACE FUNCTION public.sign_jwt(payload jsonb, key text, ttl_seconds integer)
 RETURNS text
 LANGUAGE plpgsql
AS $function$
DECLARE
  claims jsonb := payload || jsonb_build_object('exp', (extract(epoch from now())::bigint + ttl_seconds));
  token text;
BEGIN
  -- pgjwt.sign returns text
  token := pgjwt.sign(claims::json, key);
  RETURN token;
END;
$function$
;

CREATE OR REPLACE FUNCTION public.update_updated_at_column()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$function$
;

grant delete on table "public"."activities" to "anon";

grant insert on table "public"."activities" to "anon";

grant references on table "public"."activities" to "anon";

grant select on table "public"."activities" to "anon";

grant trigger on table "public"."activities" to "anon";

grant truncate on table "public"."activities" to "anon";

grant update on table "public"."activities" to "anon";

grant delete on table "public"."activities" to "authenticated";

grant insert on table "public"."activities" to "authenticated";

grant references on table "public"."activities" to "authenticated";

grant select on table "public"."activities" to "authenticated";

grant trigger on table "public"."activities" to "authenticated";

grant truncate on table "public"."activities" to "authenticated";

grant update on table "public"."activities" to "authenticated";

grant delete on table "public"."activities" to "service_role";

grant insert on table "public"."activities" to "service_role";

grant references on table "public"."activities" to "service_role";

grant select on table "public"."activities" to "service_role";

grant trigger on table "public"."activities" to "service_role";

grant truncate on table "public"."activities" to "service_role";

grant update on table "public"."activities" to "service_role";

grant delete on table "public"."agent_device_codes" to "anon";

grant insert on table "public"."agent_device_codes" to "anon";

grant references on table "public"."agent_device_codes" to "anon";

grant select on table "public"."agent_device_codes" to "anon";

grant trigger on table "public"."agent_device_codes" to "anon";

grant truncate on table "public"."agent_device_codes" to "anon";

grant update on table "public"."agent_device_codes" to "anon";

grant delete on table "public"."agent_device_codes" to "authenticated";

grant insert on table "public"."agent_device_codes" to "authenticated";

grant references on table "public"."agent_device_codes" to "authenticated";

grant select on table "public"."agent_device_codes" to "authenticated";

grant trigger on table "public"."agent_device_codes" to "authenticated";

grant truncate on table "public"."agent_device_codes" to "authenticated";

grant update on table "public"."agent_device_codes" to "authenticated";

grant delete on table "public"."agent_device_codes" to "service_role";

grant insert on table "public"."agent_device_codes" to "service_role";

grant references on table "public"."agent_device_codes" to "service_role";

grant select on table "public"."agent_device_codes" to "service_role";

grant trigger on table "public"."agent_device_codes" to "service_role";

grant truncate on table "public"."agent_device_codes" to "service_role";

grant update on table "public"."agent_device_codes" to "service_role";

grant delete on table "public"."agent_metrics" to "anon";

grant insert on table "public"."agent_metrics" to "anon";

grant references on table "public"."agent_metrics" to "anon";

grant select on table "public"."agent_metrics" to "anon";

grant trigger on table "public"."agent_metrics" to "anon";

grant truncate on table "public"."agent_metrics" to "anon";

grant update on table "public"."agent_metrics" to "anon";

grant delete on table "public"."agent_metrics" to "authenticated";

grant insert on table "public"."agent_metrics" to "authenticated";

grant references on table "public"."agent_metrics" to "authenticated";

grant select on table "public"."agent_metrics" to "authenticated";

grant trigger on table "public"."agent_metrics" to "authenticated";

grant truncate on table "public"."agent_metrics" to "authenticated";

grant update on table "public"."agent_metrics" to "authenticated";

grant delete on table "public"."agent_metrics" to "service_role";

grant insert on table "public"."agent_metrics" to "service_role";

grant references on table "public"."agent_metrics" to "service_role";

grant select on table "public"."agent_metrics" to "service_role";

grant trigger on table "public"."agent_metrics" to "service_role";

grant truncate on table "public"."agent_metrics" to "service_role";

grant update on table "public"."agent_metrics" to "service_role";

grant delete on table "public"."agent_packages" to "anon";

grant insert on table "public"."agent_packages" to "anon";

grant references on table "public"."agent_packages" to "anon";

grant select on table "public"."agent_packages" to "anon";

grant trigger on table "public"."agent_packages" to "anon";

grant truncate on table "public"."agent_packages" to "anon";

grant update on table "public"."agent_packages" to "anon";

grant delete on table "public"."agent_packages" to "authenticated";

grant insert on table "public"."agent_packages" to "authenticated";

grant references on table "public"."agent_packages" to "authenticated";

grant select on table "public"."agent_packages" to "authenticated";

grant trigger on table "public"."agent_packages" to "authenticated";

grant truncate on table "public"."agent_packages" to "authenticated";

grant update on table "public"."agent_packages" to "authenticated";

grant delete on table "public"."agent_packages" to "service_role";

grant insert on table "public"."agent_packages" to "service_role";

grant references on table "public"."agent_packages" to "service_role";

grant select on table "public"."agent_packages" to "service_role";

grant trigger on table "public"."agent_packages" to "service_role";

grant truncate on table "public"."agent_packages" to "service_role";

grant update on table "public"."agent_packages" to "service_role";

grant delete on table "public"."agent_registrations" to "anon";

grant insert on table "public"."agent_registrations" to "anon";

grant references on table "public"."agent_registrations" to "anon";

grant select on table "public"."agent_registrations" to "anon";

grant trigger on table "public"."agent_registrations" to "anon";

grant truncate on table "public"."agent_registrations" to "anon";

grant update on table "public"."agent_registrations" to "anon";

grant delete on table "public"."agent_registrations" to "authenticated";

grant insert on table "public"."agent_registrations" to "authenticated";

grant references on table "public"."agent_registrations" to "authenticated";

grant select on table "public"."agent_registrations" to "authenticated";

grant trigger on table "public"."agent_registrations" to "authenticated";

grant truncate on table "public"."agent_registrations" to "authenticated";

grant update on table "public"."agent_registrations" to "authenticated";

grant delete on table "public"."agent_registrations" to "service_role";

grant insert on table "public"."agent_registrations" to "service_role";

grant references on table "public"."agent_registrations" to "service_role";

grant select on table "public"."agent_registrations" to "service_role";

grant trigger on table "public"."agent_registrations" to "service_role";

grant truncate on table "public"."agent_registrations" to "service_role";

grant update on table "public"."agent_registrations" to "service_role";

grant delete on table "public"."agent_tokens" to "anon";

grant insert on table "public"."agent_tokens" to "anon";

grant references on table "public"."agent_tokens" to "anon";

grant select on table "public"."agent_tokens" to "anon";

grant trigger on table "public"."agent_tokens" to "anon";

grant truncate on table "public"."agent_tokens" to "anon";

grant update on table "public"."agent_tokens" to "anon";

grant delete on table "public"."agent_tokens" to "authenticated";

grant insert on table "public"."agent_tokens" to "authenticated";

grant references on table "public"."agent_tokens" to "authenticated";

grant select on table "public"."agent_tokens" to "authenticated";

grant trigger on table "public"."agent_tokens" to "authenticated";

grant truncate on table "public"."agent_tokens" to "authenticated";

grant update on table "public"."agent_tokens" to "authenticated";

grant delete on table "public"."agent_tokens" to "service_role";

grant insert on table "public"."agent_tokens" to "service_role";

grant references on table "public"."agent_tokens" to "service_role";

grant select on table "public"."agent_tokens" to "service_role";

grant trigger on table "public"."agent_tokens" to "service_role";

grant truncate on table "public"."agent_tokens" to "service_role";

grant update on table "public"."agent_tokens" to "service_role";

grant delete on table "public"."agents" to "anon";

grant insert on table "public"."agents" to "anon";

grant references on table "public"."agents" to "anon";

grant select on table "public"."agents" to "anon";

grant trigger on table "public"."agents" to "anon";

grant truncate on table "public"."agents" to "anon";

grant update on table "public"."agents" to "anon";

grant delete on table "public"."agents" to "authenticated";

grant insert on table "public"."agents" to "authenticated";

grant references on table "public"."agents" to "authenticated";

grant select on table "public"."agents" to "authenticated";

grant trigger on table "public"."agents" to "authenticated";

grant truncate on table "public"."agents" to "authenticated";

grant update on table "public"."agents" to "authenticated";

grant delete on table "public"."agents" to "service_role";

grant insert on table "public"."agents" to "service_role";

grant references on table "public"."agents" to "service_role";

grant select on table "public"."agents" to "service_role";

grant trigger on table "public"."agents" to "service_role";

grant truncate on table "public"."agents" to "service_role";

grant update on table "public"."agents" to "service_role";

grant delete on table "public"."installed_packages" to "anon";

grant insert on table "public"."installed_packages" to "anon";

grant references on table "public"."installed_packages" to "anon";

grant select on table "public"."installed_packages" to "anon";

grant trigger on table "public"."installed_packages" to "anon";

grant truncate on table "public"."installed_packages" to "anon";

grant update on table "public"."installed_packages" to "anon";

grant delete on table "public"."installed_packages" to "authenticated";

grant insert on table "public"."installed_packages" to "authenticated";

grant references on table "public"."installed_packages" to "authenticated";

grant select on table "public"."installed_packages" to "authenticated";

grant trigger on table "public"."installed_packages" to "authenticated";

grant truncate on table "public"."installed_packages" to "authenticated";

grant update on table "public"."installed_packages" to "authenticated";

grant delete on table "public"."installed_packages" to "service_role";

grant insert on table "public"."installed_packages" to "service_role";

grant references on table "public"."installed_packages" to "service_role";

grant select on table "public"."installed_packages" to "service_role";

grant trigger on table "public"."installed_packages" to "service_role";

grant truncate on table "public"."installed_packages" to "service_role";

grant update on table "public"."installed_packages" to "service_role";

grant delete on table "public"."investigations" to "anon";

grant insert on table "public"."investigations" to "anon";

grant references on table "public"."investigations" to "anon";

grant select on table "public"."investigations" to "anon";

grant trigger on table "public"."investigations" to "anon";

grant truncate on table "public"."investigations" to "anon";

grant update on table "public"."investigations" to "anon";

grant delete on table "public"."investigations" to "authenticated";

grant insert on table "public"."investigations" to "authenticated";

grant references on table "public"."investigations" to "authenticated";

grant select on table "public"."investigations" to "authenticated";

grant trigger on table "public"."investigations" to "authenticated";

grant truncate on table "public"."investigations" to "authenticated";

grant update on table "public"."investigations" to "authenticated";

grant delete on table "public"."investigations" to "service_role";

grant insert on table "public"."investigations" to "service_role";

grant references on table "public"."investigations" to "service_role";

grant select on table "public"."investigations" to "service_role";

grant trigger on table "public"."investigations" to "service_role";

grant truncate on table "public"."investigations" to "service_role";

grant update on table "public"."investigations" to "service_role";

grant delete on table "public"."package_exceptions" to "anon";

grant insert on table "public"."package_exceptions" to "anon";

grant references on table "public"."package_exceptions" to "anon";

grant select on table "public"."package_exceptions" to "anon";

grant trigger on table "public"."package_exceptions" to "anon";

grant truncate on table "public"."package_exceptions" to "anon";

grant update on table "public"."package_exceptions" to "anon";

grant delete on table "public"."package_exceptions" to "authenticated";

grant insert on table "public"."package_exceptions" to "authenticated";

grant references on table "public"."package_exceptions" to "authenticated";

grant select on table "public"."package_exceptions" to "authenticated";

grant trigger on table "public"."package_exceptions" to "authenticated";

grant truncate on table "public"."package_exceptions" to "authenticated";

grant update on table "public"."package_exceptions" to "authenticated";

grant delete on table "public"."package_exceptions" to "service_role";

grant insert on table "public"."package_exceptions" to "service_role";

grant references on table "public"."package_exceptions" to "service_role";

grant select on table "public"."package_exceptions" to "service_role";

grant trigger on table "public"."package_exceptions" to "service_role";

grant truncate on table "public"."package_exceptions" to "service_role";

grant update on table "public"."package_exceptions" to "service_role";

grant delete on table "public"."package_fetch_commands" to "anon";

grant insert on table "public"."package_fetch_commands" to "anon";

grant references on table "public"."package_fetch_commands" to "anon";

grant select on table "public"."package_fetch_commands" to "anon";

grant trigger on table "public"."package_fetch_commands" to "anon";

grant truncate on table "public"."package_fetch_commands" to "anon";

grant update on table "public"."package_fetch_commands" to "anon";

grant delete on table "public"."package_fetch_commands" to "authenticated";

grant insert on table "public"."package_fetch_commands" to "authenticated";

grant references on table "public"."package_fetch_commands" to "authenticated";

grant select on table "public"."package_fetch_commands" to "authenticated";

grant trigger on table "public"."package_fetch_commands" to "authenticated";

grant truncate on table "public"."package_fetch_commands" to "authenticated";

grant update on table "public"."package_fetch_commands" to "authenticated";

grant delete on table "public"."package_fetch_commands" to "service_role";

grant insert on table "public"."package_fetch_commands" to "service_role";

grant references on table "public"."package_fetch_commands" to "service_role";

grant select on table "public"."package_fetch_commands" to "service_role";

grant trigger on table "public"."package_fetch_commands" to "service_role";

grant truncate on table "public"."package_fetch_commands" to "service_role";

grant update on table "public"."package_fetch_commands" to "service_role";

grant delete on table "public"."patch_executions" to "anon";

grant insert on table "public"."patch_executions" to "anon";

grant references on table "public"."patch_executions" to "anon";

grant select on table "public"."patch_executions" to "anon";

grant trigger on table "public"."patch_executions" to "anon";

grant truncate on table "public"."patch_executions" to "anon";

grant update on table "public"."patch_executions" to "anon";

grant delete on table "public"."patch_executions" to "authenticated";

grant insert on table "public"."patch_executions" to "authenticated";

grant references on table "public"."patch_executions" to "authenticated";

grant select on table "public"."patch_executions" to "authenticated";

grant trigger on table "public"."patch_executions" to "authenticated";

grant truncate on table "public"."patch_executions" to "authenticated";

grant update on table "public"."patch_executions" to "authenticated";

grant delete on table "public"."patch_executions" to "service_role";

grant insert on table "public"."patch_executions" to "service_role";

grant references on table "public"."patch_executions" to "service_role";

grant select on table "public"."patch_executions" to "service_role";

grant trigger on table "public"."patch_executions" to "service_role";

grant truncate on table "public"."patch_executions" to "service_role";

grant update on table "public"."patch_executions" to "service_role";

grant delete on table "public"."patch_management_schedules" to "anon";

grant insert on table "public"."patch_management_schedules" to "anon";

grant references on table "public"."patch_management_schedules" to "anon";

grant select on table "public"."patch_management_schedules" to "anon";

grant trigger on table "public"."patch_management_schedules" to "anon";

grant truncate on table "public"."patch_management_schedules" to "anon";

grant update on table "public"."patch_management_schedules" to "anon";

grant delete on table "public"."patch_management_schedules" to "authenticated";

grant insert on table "public"."patch_management_schedules" to "authenticated";

grant references on table "public"."patch_management_schedules" to "authenticated";

grant select on table "public"."patch_management_schedules" to "authenticated";

grant trigger on table "public"."patch_management_schedules" to "authenticated";

grant truncate on table "public"."patch_management_schedules" to "authenticated";

grant update on table "public"."patch_management_schedules" to "authenticated";

grant delete on table "public"."patch_management_schedules" to "service_role";

grant insert on table "public"."patch_management_schedules" to "service_role";

grant references on table "public"."patch_management_schedules" to "service_role";

grant select on table "public"."patch_management_schedules" to "service_role";

grant trigger on table "public"."patch_management_schedules" to "service_role";

grant truncate on table "public"."patch_management_schedules" to "service_role";

grant update on table "public"."patch_management_schedules" to "service_role";

grant delete on table "public"."patch_scripts" to "anon";

grant insert on table "public"."patch_scripts" to "anon";

grant references on table "public"."patch_scripts" to "anon";

grant select on table "public"."patch_scripts" to "anon";

grant trigger on table "public"."patch_scripts" to "anon";

grant truncate on table "public"."patch_scripts" to "anon";

grant update on table "public"."patch_scripts" to "anon";

grant delete on table "public"."patch_scripts" to "authenticated";

grant insert on table "public"."patch_scripts" to "authenticated";

grant references on table "public"."patch_scripts" to "authenticated";

grant select on table "public"."patch_scripts" to "authenticated";

grant trigger on table "public"."patch_scripts" to "authenticated";

grant truncate on table "public"."patch_scripts" to "authenticated";

grant update on table "public"."patch_scripts" to "authenticated";

grant delete on table "public"."patch_scripts" to "service_role";

grant insert on table "public"."patch_scripts" to "service_role";

grant references on table "public"."patch_scripts" to "service_role";

grant select on table "public"."patch_scripts" to "service_role";

grant trigger on table "public"."patch_scripts" to "service_role";

grant truncate on table "public"."patch_scripts" to "service_role";

grant update on table "public"."patch_scripts" to "service_role";

grant delete on table "public"."patch_tasks" to "anon";

grant insert on table "public"."patch_tasks" to "anon";

grant references on table "public"."patch_tasks" to "anon";

grant select on table "public"."patch_tasks" to "anon";

grant trigger on table "public"."patch_tasks" to "anon";

grant truncate on table "public"."patch_tasks" to "anon";

grant update on table "public"."patch_tasks" to "anon";

grant delete on table "public"."patch_tasks" to "authenticated";

grant insert on table "public"."patch_tasks" to "authenticated";

grant references on table "public"."patch_tasks" to "authenticated";

grant select on table "public"."patch_tasks" to "authenticated";

grant trigger on table "public"."patch_tasks" to "authenticated";

grant truncate on table "public"."patch_tasks" to "authenticated";

grant update on table "public"."patch_tasks" to "authenticated";

grant delete on table "public"."patch_tasks" to "service_role";

grant insert on table "public"."patch_tasks" to "service_role";

grant references on table "public"."patch_tasks" to "service_role";

grant select on table "public"."patch_tasks" to "service_role";

grant trigger on table "public"."patch_tasks" to "service_role";

grant truncate on table "public"."patch_tasks" to "service_role";

grant update on table "public"."patch_tasks" to "service_role";

grant delete on table "public"."pending_investigations" to "anon";

grant insert on table "public"."pending_investigations" to "anon";

grant references on table "public"."pending_investigations" to "anon";

grant select on table "public"."pending_investigations" to "anon";

grant trigger on table "public"."pending_investigations" to "anon";

grant truncate on table "public"."pending_investigations" to "anon";

grant update on table "public"."pending_investigations" to "anon";

grant delete on table "public"."pending_investigations" to "authenticated";

grant insert on table "public"."pending_investigations" to "authenticated";

grant references on table "public"."pending_investigations" to "authenticated";

grant select on table "public"."pending_investigations" to "authenticated";

grant trigger on table "public"."pending_investigations" to "authenticated";

grant truncate on table "public"."pending_investigations" to "authenticated";

grant update on table "public"."pending_investigations" to "authenticated";

grant delete on table "public"."pending_investigations" to "service_role";

grant insert on table "public"."pending_investigations" to "service_role";

grant references on table "public"."pending_investigations" to "service_role";

grant select on table "public"."pending_investigations" to "service_role";

grant trigger on table "public"."pending_investigations" to "service_role";

grant truncate on table "public"."pending_investigations" to "service_role";

grant update on table "public"."pending_investigations" to "service_role";

grant delete on table "public"."pricing_plans" to "anon";

grant insert on table "public"."pricing_plans" to "anon";

grant references on table "public"."pricing_plans" to "anon";

grant select on table "public"."pricing_plans" to "anon";

grant trigger on table "public"."pricing_plans" to "anon";

grant truncate on table "public"."pricing_plans" to "anon";

grant update on table "public"."pricing_plans" to "anon";

grant delete on table "public"."pricing_plans" to "authenticated";

grant insert on table "public"."pricing_plans" to "authenticated";

grant references on table "public"."pricing_plans" to "authenticated";

grant select on table "public"."pricing_plans" to "authenticated";

grant trigger on table "public"."pricing_plans" to "authenticated";

grant truncate on table "public"."pricing_plans" to "authenticated";

grant update on table "public"."pricing_plans" to "authenticated";

grant delete on table "public"."pricing_plans" to "service_role";

grant insert on table "public"."pricing_plans" to "service_role";

grant references on table "public"."pricing_plans" to "service_role";

grant select on table "public"."pricing_plans" to "service_role";

grant trigger on table "public"."pricing_plans" to "service_role";

grant truncate on table "public"."pricing_plans" to "service_role";

grant update on table "public"."pricing_plans" to "service_role";

grant delete on table "public"."scheduled_patches" to "anon";

grant insert on table "public"."scheduled_patches" to "anon";

grant references on table "public"."scheduled_patches" to "anon";

grant select on table "public"."scheduled_patches" to "anon";

grant trigger on table "public"."scheduled_patches" to "anon";

grant truncate on table "public"."scheduled_patches" to "anon";

grant update on table "public"."scheduled_patches" to "anon";

grant delete on table "public"."scheduled_patches" to "authenticated";

grant insert on table "public"."scheduled_patches" to "authenticated";

grant references on table "public"."scheduled_patches" to "authenticated";

grant select on table "public"."scheduled_patches" to "authenticated";

grant trigger on table "public"."scheduled_patches" to "authenticated";

grant truncate on table "public"."scheduled_patches" to "authenticated";

grant update on table "public"."scheduled_patches" to "authenticated";

grant delete on table "public"."scheduled_patches" to "service_role";

grant insert on table "public"."scheduled_patches" to "service_role";

grant references on table "public"."scheduled_patches" to "service_role";

grant select on table "public"."scheduled_patches" to "service_role";

grant trigger on table "public"."scheduled_patches" to "service_role";

grant truncate on table "public"."scheduled_patches" to "service_role";

grant update on table "public"."scheduled_patches" to "service_role";


  create policy "Users can claim unassigned device codes"
  on "public"."agent_device_codes"
  as permissive
  for update
  to public
using (((user_id IS NULL) OR (auth.uid() = user_id)));



  create policy "Users can view unassigned and own device codes"
  on "public"."agent_device_codes"
  as permissive
  for select
  to public
using (((user_id IS NULL) OR (auth.uid() = user_id)));



  create policy "Service role can manage all packages"
  on "public"."agent_packages"
  as permissive
  for all
  to public
using (((auth.jwt() ->> 'role'::text) = 'service_role'::text));



  create policy "Users can view their agent packages"
  on "public"."agent_packages"
  as permissive
  for select
  to public
using ((agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid()))));



  create policy "Users can create their own agents"
  on "public"."agent_registrations"
  as permissive
  for insert
  to public
with check ((auth.uid() = user_id));



  create policy "Users can update their own agents"
  on "public"."agent_registrations"
  as permissive
  for update
  to public
using ((auth.uid() = user_id));



  create policy "Users can view their own agents"
  on "public"."agent_registrations"
  as permissive
  for select
  to public
using ((auth.uid() = user_id));



  create policy "Users can view their own agent tokens"
  on "public"."agent_tokens"
  as permissive
  for select
  to public
using ((auth.uid() = user_id));



  create policy "agents_delete_owner"
  on "public"."agents"
  as permissive
  for delete
  to authenticated
using ((owner = auth.uid()));



  create policy "agents_insert_owner"
  on "public"."agents"
  as permissive
  for insert
  to authenticated
with check ((owner = auth.uid()));



  create policy "agents_select_owner"
  on "public"."agents"
  as permissive
  for select
  to authenticated
using ((owner = auth.uid()));



  create policy "agents_update_owner"
  on "public"."agents"
  as permissive
  for update
  to authenticated
using ((owner = auth.uid()))
with check ((owner = auth.uid()));



  create policy "Service role can manage all installed packages"
  on "public"."installed_packages"
  as permissive
  for all
  to public
using (((auth.jwt() ->> 'role'::text) = 'service_role'::text));



  create policy "Users can view their installed packages"
  on "public"."installed_packages"
  as permissive
  for select
  to public
using ((agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid()))));



  create policy "Agents access assigned investigations"
  on "public"."investigations"
  as permissive
  for all
  to public
using (((auth.uid() = agent_id) OR (agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid())))))
with check (((auth.uid() = agent_id) OR (agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid())))));



  create policy "Agents can select assigned investigations"
  on "public"."investigations"
  as permissive
  for select
  to public
using ((agent_id = auth.uid()));



  create policy "Agents can update their investigation status"
  on "public"."investigations"
  as permissive
  for update
  to public
using ((agent_id = auth.uid()))
with check ((agent_id = auth.uid()));



  create policy "Service role can update investigations"
  on "public"."investigations"
  as permissive
  for update
  to public
using (true)
with check (true);



  create policy "Users can insert investigations"
  on "public"."investigations"
  as permissive
  for insert
  to public
with check ((((auth.uid())::text = (initiated_by)::text) OR (initiated_by IS NULL)));



  create policy "Users can select own investigations"
  on "public"."investigations"
  as permissive
  for select
  to public
using ((((initiated_by)::text = (auth.uid())::text) OR (initiated_by IS NULL)));



  create policy "Service role can manage package_exceptions"
  on "public"."package_exceptions"
  as permissive
  for all
  to service_role
using (true);



  create policy "Users can manage their agents' package_exceptions"
  on "public"."package_exceptions"
  as permissive
  for all
  to authenticated
using ((agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid()))));



  create policy "Agents can select assigned patch executions"
  on "public"."patch_executions"
  as permissive
  for select
  to public
using ((agent_id = auth.uid()));



  create policy "Agents can update patch executions"
  on "public"."patch_executions"
  as permissive
  for update
  to public
using ((agent_id = auth.uid()))
with check ((agent_id = auth.uid()));



  create policy "Service role can manage patch_executions"
  on "public"."patch_executions"
  as permissive
  for all
  to service_role
using (true)
with check (true);



  create policy "Users can insert patch executions"
  on "public"."patch_executions"
  as permissive
  for insert
  to public
with check (true);



  create policy "Users can select own patch executions"
  on "public"."patch_executions"
  as permissive
  for select
  to public
using ((agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid()))));



  create policy "Service role can manage all schedules"
  on "public"."patch_management_schedules"
  as permissive
  for all
  to public
using (((auth.jwt() ->> 'role'::text) = 'service_role'::text));



  create policy "Users can manage their schedules"
  on "public"."patch_management_schedules"
  as permissive
  for all
  to public
using ((user_id = auth.uid()));



  create policy "Service role can manage patch_scripts"
  on "public"."patch_scripts"
  as permissive
  for all
  to service_role
using (true);



  create policy "Users can read active patch_scripts"
  on "public"."patch_scripts"
  as permissive
  for select
  to authenticated
using ((is_active = true));



  create policy "Service role can manage patch tasks"
  on "public"."patch_tasks"
  as permissive
  for all
  to service_role
using (true)
with check (true);



  create policy "Service role can manage pending_investigations"
  on "public"."pending_investigations"
  as permissive
  for all
  to service_role
using (true)
with check (true);



  create policy "pricing_public_select"
  on "public"."pricing_plans"
  as permissive
  for select
  to public
using (true);



  create policy "Service role can manage scheduled_patches"
  on "public"."scheduled_patches"
  as permissive
  for all
  to service_role
using (true);



  create policy "Users can manage their agents' scheduled_patches"
  on "public"."scheduled_patches"
  as permissive
  for all
  to authenticated
using ((agent_id IN ( SELECT agents.id
   FROM public.agents
  WHERE (agents.owner = auth.uid()))));



  create policy "Service role can insert backup code usage"
  on "public"."user_mfa_backup_codes_used"
  as permissive
  for insert
  to public
with check (((auth.uid() = user_id) OR (auth.role() = 'service_role'::text)));



  create policy "Service role can read all backup code usage"
  on "public"."user_mfa_backup_codes_used"
  as permissive
  for select
  to public
using (((auth.uid() = user_id) OR (auth.role() = 'service_role'::text)));



  create policy "Service role can read all MFA settings"
  on "public"."user_mfa_settings"
  as permissive
  for select
  to public
using (((auth.uid() = user_id) OR (auth.role() = 'service_role'::text)));



  create policy "Users can create their own MFA settings"
  on "public"."user_mfa_settings"
  as permissive
  for insert
  to public
with check ((auth.uid() = user_id));



  create policy "Users can delete their own MFA settings"
  on "public"."user_mfa_settings"
  as permissive
  for delete
  to public
using ((auth.uid() = user_id));



  create policy "Users can update their own MFA settings"
  on "public"."user_mfa_settings"
  as permissive
  for update
  to public
using ((auth.uid() = user_id))
with check ((auth.uid() = user_id));


CREATE TRIGGER update_agent_registrations_updated_at BEFORE UPDATE ON public.agent_registrations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


  create policy "Agents can upload execution outputs"
  on "storage"."objects"
  as permissive
  for insert
  to authenticated
with check (((bucket_id = 'patch-execution-outputs'::text) AND ((storage.foldername(name))[1] = (auth.jwt() ->> 'sub'::text))));



  create policy "Agents can upload their task outputs"
  on "storage"."objects"
  as permissive
  for insert
  to public
with check (((bucket_id = 'patch-task-outputs'::text) AND (auth.role() = 'authenticated'::text) AND ((storage.foldername(name))[1] = (auth.jwt() ->> 'sub'::text))));



  create policy "Authenticated users can read patch scripts"
  on "storage"."objects"
  as permissive
  for select
  to authenticated
using ((bucket_id = 'patch-scripts'::text));



  create policy "Service role can manage execution outputs in storage"
  on "storage"."objects"
  as permissive
  for all
  to service_role
using ((bucket_id = 'patch-execution-outputs'::text));



  create policy "Service role can manage patch reports"
  on "storage"."objects"
  as permissive
  for all
  to public
using (((bucket_id = 'patch-reports'::text) AND ((auth.jwt() ->> 'role'::text) = 'service_role'::text)));



  create policy "Service role can manage patch scripts in storage"
  on "storage"."objects"
  as permissive
  for all
  to service_role
using ((bucket_id = 'patch-scripts'::text));



  create policy "Service role can manage patch task outputs"
  on "storage"."objects"
  as permissive
  for all
  to public
using (((bucket_id = 'patch-task-outputs'::text) AND ((auth.jwt() ->> 'role'::text) = 'service_role'::text)));



  create policy "Users can read their agent patch reports"
  on "storage"."objects"
  as permissive
  for select
  to public
using (((bucket_id = 'patch-reports'::text) AND ((storage.foldername(name))[1] IN ( SELECT (agents.id)::text AS id
   FROM public.agents
  WHERE (agents.owner = auth.uid())))));



  create policy "Users can read their agent task outputs"
  on "storage"."objects"
  as permissive
  for select
  to public
using (((bucket_id = 'patch-task-outputs'::text) AND ((storage.foldername(name))[1] IN ( SELECT (agents.id)::text AS id
   FROM public.agents
  WHERE (agents.owner = auth.uid())))));



  create policy "Users can read their agents' execution outputs"
  on "storage"."objects"
  as permissive
  for select
  to authenticated
using (((bucket_id = 'patch-execution-outputs'::text) AND ((storage.foldername(name))[1] IN ( SELECT (agents.id)::text AS id
   FROM public.agents
  WHERE (agents.owner = auth.uid())))));



