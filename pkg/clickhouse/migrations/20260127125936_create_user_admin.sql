-- Creates a dedicated admin user for workspace ClickHouse user provisioning.
--
-- This user has minimal permissions to manage other users without access to data:
-- - CREATE/ALTER/DROP USER, QUOTA, ROW POLICY, SETTINGS PROFILE
-- - GRANT OPTION on analytics tables (to grant SELECT to workspace users)
--
-- This migration runs as part of the ClickHouse image build.

-- Create the admin user with SHA256 password authentication
CREATE USER IF NOT EXISTS unkey_user_admin IDENTIFIED WITH sha256_password BY 'C57RqT5EPZBqCJkMxN9mEZZEzMPcw9yBlwhIizk99t7kx6uLi9rYmtWObsXzdl';

-- User management permissions
GRANT CREATE USER, ALTER USER, DROP USER ON *.* TO unkey_user_admin;

-- Quota management permissions
GRANT CREATE QUOTA, ALTER QUOTA, DROP QUOTA ON *.* TO unkey_user_admin;

-- Row policy management permissions
GRANT CREATE ROW POLICY, ALTER ROW POLICY, DROP ROW POLICY ON *.* TO unkey_user_admin;

-- Settings profile management permissions
GRANT CREATE SETTINGS PROFILE, ALTER SETTINGS PROFILE, DROP SETTINGS PROFILE ON *.* TO unkey_user_admin;

-- Grant OPTION on analytics tables (allows granting SELECT to workspace users)
-- Key verifications tables
GRANT SELECT ON default.key_verifications_raw_v2 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_minute_v2 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_minute_v3 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_hour_v2 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_hour_v3 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_day_v2 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_day_v3 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_month_v2 TO unkey_user_admin WITH GRANT OPTION;
GRANT SELECT ON default.key_verifications_per_month_v3 TO unkey_user_admin WITH GRANT OPTION;
