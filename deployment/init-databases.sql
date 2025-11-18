-- Initialize multiple databases for the Unkey deployment platform
CREATE DATABASE IF NOT EXISTS unkey;

-- Create the unkey user
CREATE USER IF NOT EXISTS 'unkey'@'%' IDENTIFIED BY 'password';

-- Grant permissions to unkey user for all databases
GRANT ALL PRIVILEGES ON unkey.* TO 'unkey'@'%';
FLUSH PRIVILEGES;
