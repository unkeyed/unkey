-- Initialize multiple databases for the Unkey deployment platform
CREATE DATABASE IF NOT EXISTS unkey;
CREATE DATABASE IF NOT EXISTS hydra;
CREATE DATABASE IF NOT EXISTS partition_001;

-- Create the unkey user
CREATE USER IF NOT EXISTS 'unkey'@'%' IDENTIFIED BY 'password';

-- Grant permissions to unkey user for all databases
GRANT ALL PRIVILEGES ON unkey.* TO 'unkey'@'%';
GRANT ALL PRIVILEGES ON hydra.* TO 'unkey'@'%';
GRANT ALL PRIVILEGES ON partition_001.* TO 'unkey'@'%';
FLUSH PRIVILEGES;
