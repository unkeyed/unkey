-- LOCAL DEVELOPMENT ONLY - DO NOT USE IN PRODUCTION
--
-- Creates admin user for ClickHouse user provisioning with a hardcoded password.
-- For production/self-hosted: create this user manually with a secure password.
CREATE USER IF NOT EXISTS unkey_user_admin IDENTIFIED WITH sha256_password BY 'C57RqT5EPZBqCJkMxN9mEZZEzMPcw9yBlwhIizk99t7kx6uLi9rYmtWObsXzdl';

-- User management permissions
GRANT CREATE USER, ALTER USER, DROP USER ON *.* TO unkey_user_admin;

-- Quota management permissions
GRANT CREATE QUOTA, ALTER QUOTA, DROP QUOTA ON *.* TO unkey_user_admin;

-- Row policy management permissions
GRANT CREATE ROW POLICY, ALTER ROW POLICY, DROP ROW POLICY ON *.* TO unkey_user_admin;

-- Settings profile management permissions
GRANT CREATE SETTINGS PROFILE, ALTER SETTINGS PROFILE, DROP SETTINGS PROFILE ON *.* TO unkey_user_admin;

-- Grant SELECT with GRANT OPTION on default database (allows granting SELECT to workspace users)
GRANT SELECT ON default.* TO unkey_user_admin WITH GRANT OPTION;
