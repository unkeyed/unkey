CREATE DATABASE IF NOT EXISTS `partition_001`;
USE `partition_001`;

-- Partition Database Schema
-- This database is regionally replicated and optimized for high-throughput, low-latency data plane access.
-- It provides gateways with routing tables, TLS certificates, and VM status.

-- Gateway configuration per hostname
-- Contains all middleware configuration (auth, rate limiting, validation, etc.) as protobuf
CREATE TABLE gateways (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `workspace_id` varchar(255) NOT NULL,
  `hostname` varchar(255) NOT NULL,
  `config` blob NOT NULL,   -- Protobuf with all configuration including deployment_id, workspace_id
  PRIMARY KEY (`id`),
  UNIQUE KEY `gateways_pk` (`hostname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Virtual machine instances
-- Tracks complete VM lifecycle from allocation to termination
CREATE TABLE vms (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    metal_host_id VARCHAR(255),  -- NULL until assigned to a host
    -- metalhost ip and port
    address VARCHAR(255),                    -- NULL until provisioned
    cpu_millicores INT NOT NULL,
    memory_mb INT NOT NULL,
    status ENUM('allocated', 'provisioning', 'starting', 'running', 'stopping', 'stopped', 'failed') NOT NULL,


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
CREATE TABLE `certificates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `workspace_id` varchar(255) NOT NULL,
  `hostname` varchar(255) NOT NULL,
  `certificate` text NOT NULL,
  `encrypted_private_key` text NOT NULL,
  `created_at` bigint NOT NULL,
  `updated_at` bigint,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_hostname` (`hostname`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
