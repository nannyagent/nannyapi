-- Seed patch_scripts with sample update scripts

INSERT INTO "public"."patch_scripts" ("id", "name", "description", "script_storage_path", "os_family", "os_platform", "major_version", "minor_version", "package_manager", "script_type", "created_at", "updated_at", "created_by", "is_active") VALUES
  ('7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc', 'Debian/Ubuntu APT Update', 'System update script for Debian and Ubuntu using APT package manager. Supports dry-run mode and package exceptions.', 'debian/apt-update.sh', 'debian', 'ubuntu', NULL, NULL, 'apt', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true),
  ('7c0c1f63-0fd4-4d07-abbe-efa4bc4d9c1e', 'Raspbian APT Update', 'System update script for Raspbian using APT package manager. Supports dry-run mode and package exceptions.', 'debian/apt-update.sh', 'raspbian', 'raspbian', NULL, NULL, 'apt', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true),
  ('ac6cbc78-83fc-4fc6-919c-0d1c1a974125', 'RHEL/CentOS DNF Update', 'System update script for RHEL 8+, CentOS 8+, and Fedora using DNF package manager. Supports dry-run mode and package exceptions.', 'rhel/dnf-update.sh', 'rhel', 'centos', NULL, NULL, 'dnf', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true),
  ('b405920f-e257-494b-a8fe-0a22160e7e56', 'RHEL/CentOS YUM Update', 'System update script for RHEL 7, CentOS 7, and older versions using YUM package manager. Supports dry-run mode and package exceptions.', 'rhel/yum-update.sh', 'rhel', 'centos', NULL, NULL, 'yum', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true),
  ('e44c3c01-7fb6-4904-b816-4ff959578c8f', 'SUSE/openSUSE Zypper Update', 'System update script for SUSE and openSUSE using Zypper package manager. Supports dry-run mode and package exceptions.', 'suse/zypper-update.sh', 'suse', 'opensuse', NULL, NULL, 'zypper', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true),
  ('7e6b1453-30d6-4941-b5ed-8ef876eb30a3', 'Arch Linux Pacman Update', 'System update script for Arch Linux and derivatives using Pacman package manager. Supports dry-run mode and package exceptions.', 'arch/pacman-update.sh', 'arch', 'arch', NULL, NULL, 'pacman', 'update', '2025-12-02 18:36:18.621826+00', '2025-12-02 18:36:18.621826+00', NULL, true)
ON CONFLICT (id) DO NOTHING;

-- Create storage buckets for patch scripts and execution outputs
INSERT INTO storage.buckets (id, name, owner, created_at, updated_at, public)
VALUES 
  ('patch-scripts', 'patch-scripts', NULL, NOW(), NOW(), false),
  ('patch-execution-outputs', 'patch-execution-outputs', NULL, NOW(), NOW(), false)
ON CONFLICT (id) DO NOTHING;

-- RLS policies for patch-scripts bucket
CREATE POLICY "Allow authenticated users to read patch scripts"
ON storage.objects FOR SELECT
USING (bucket_id = 'patch-scripts' AND auth.role() = 'authenticated');

CREATE POLICY "Allow service role to manage patch scripts"
ON storage.objects FOR INSERT
WITH CHECK (bucket_id = 'patch-scripts' AND auth.role() = 'service_role');

CREATE POLICY "Allow service role to update patch scripts"
ON storage.objects FOR UPDATE
USING (bucket_id = 'patch-scripts' AND auth.role() = 'service_role');

-- RLS policies for patch-execution-outputs bucket
CREATE POLICY "Allow authenticated users to read patch execution outputs"
ON storage.objects FOR SELECT
USING (bucket_id = 'patch-execution-outputs' AND auth.role() = 'authenticated');

CREATE POLICY "Allow agents to upload patch execution outputs"
ON storage.objects FOR INSERT
WITH CHECK (bucket_id = 'patch-execution-outputs' AND auth.role() = 'authenticated');

CREATE POLICY "Allow agents to update their own patch execution outputs"
ON storage.objects FOR UPDATE
USING (bucket_id = 'patch-execution-outputs' AND auth.role() = 'authenticated');
