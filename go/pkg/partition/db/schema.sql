CREATE DATABASE IF NOT EXISTS `partition_001`;
USE `partition_001`;

-- Partition Database Schema
-- This database is regionally replicated and optimized for high-throughput, low-latency data plane access.
-- It provides gateways with routing tables, TLS certificates, and VM status.

-- Gateway configuration per hostname
-- Contains all middleware configuration (auth, rate limiting, validation, etc.) as protobuf
CREATE TABLE `gateways` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `workspace_id` varchar(255) NOT NULL,
  `deployment_id` varchar(255) NOT NULL,
  `hostname` varchar(255) NOT NULL,
  `config` longblob NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gateways_pk` (`hostname`),
  KEY `idx_deployment_id` (`deployment_id`)
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Instances tracking
CREATE TABLE `instance` (
  `id` varchar(255) NOT NULL,
  `deployment_id` varchar(255) NOT NULL,
  `status` enum('allocated','provisioning','starting','running','stopping','stopped','failed') NOT NULL,
  `config` longblob NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_deployment_id` (`deployment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


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
