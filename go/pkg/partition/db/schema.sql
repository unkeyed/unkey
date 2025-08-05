CREATE DATABASE IF NOT EXISTS `partition_001`;
USE `partition_001`;

-- Partition Database Schema
-- This database is regionally replicated and optimized for high-throughput, low-latency data plane access.
-- It provides gateways with routing tables, TLS certificates, and VM status.

-- Gateway configuration per hostname
-- Contains all middleware configuration (auth, rate limiting, validation, etc.) as protobuf
CREATE TABLE gateways (
    hostname VARCHAR(255) NOT NULL PRIMARY KEY,
    gateway_config BLOB NOT NULL            -- Protobuf with all configuration including deployment_id, workspace_id
);

-- Virtual machine instances
-- Tracks complete VM lifecycle from allocation to termination
CREATE TABLE vms (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    metal_host_id VARCHAR(255),  -- NULL until assigned to a host
    region VARCHAR(255) NOT NULL,
    private_ip VARCHAR(45),      -- NULL until provisioned
    port INT,                    -- NULL until provisioned
    cpu_millicores INT NOT NULL,
    memory_mb INT NOT NULL,
    status ENUM('allocated', 'provisioning', 'starting', 'running', 'stopping', 'stopped', 'failed') NOT NULL,
    health_status ENUM('unknown', 'healthy', 'unhealthy') NOT NULL DEFAULT 'unknown',
    last_heartbeat BIGINT,       -- NULL until running

    INDEX idx_deployment_available (deployment_id, region, status),
    INDEX idx_deployment_health (deployment_id, health_status, last_heartbeat),
    INDEX idx_host_id (metal_host_id),
    INDEX idx_region (region),
    INDEX idx_status (status),
    UNIQUE KEY unique_ip_port (private_ip, port)
);

-- Metal host instances running metald
-- Tracks EC2 metal instances and their capacity for VM provisioning
CREATE TABLE metal_hosts (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    region VARCHAR(255) NOT NULL,
    availability_zone VARCHAR(255) NOT NULL,
    instance_type VARCHAR(255) NOT NULL,
    ec2_instance_id VARCHAR(255) NOT NULL,
    private_ip VARCHAR(45) NOT NULL,
    status ENUM('provisioning', 'active', 'draining', 'terminated') NOT NULL,
    capacity_cpu_millicores INT NOT NULL,
    capacity_memory_mb INT NOT NULL,
    allocated_cpu_millicores INT NOT NULL DEFAULT 0,
    allocated_memory_mb INT NOT NULL DEFAULT 0,
    last_heartbeat BIGINT NOT NULL,

    INDEX idx_region_status (region, status),
    INDEX idx_az (availability_zone),
    INDEX idx_status (status),
    INDEX idx_heartbeat (last_heartbeat),
    UNIQUE KEY unique_ec2_instance (ec2_instance_id)
);


-- TLS certificates for hostname routing
-- Stores certificates and private keys for HTTPS termination
CREATE TABLE certificates (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    certificate_pem TEXT NOT NULL,
    private_key_encrypted BLOB NOT NULL,
    expires_at BIGINT NOT NULL,

    UNIQUE KEY unique_hostname (hostname),
    INDEX idx_expires_at (expires_at)
);
