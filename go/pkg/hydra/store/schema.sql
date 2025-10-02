CREATE DATABASE IF NOT EXISTS `hydra`;
USE `hydra`;

CREATE TABLE IF NOT EXISTS workflow_executions (
    id VARCHAR(255) PRIMARY KEY,
    workflow_name VARCHAR(255) NOT NULL,
    status ENUM('pending', 'running', 'sleeping', 'completed', 'failed') NOT NULL,
    input_data LONGBLOB,  -- Large binary data for workflow inputs
    output_data MEDIUMBLOB,  -- Medium binary data for workflow outputs
    error_message TEXT,

    created_at BIGINT NOT NULL,
    started_at BIGINT,
    completed_at BIGINT,
    max_attempts INT NOT NULL,
    remaining_attempts INT NOT NULL,
    next_retry_at BIGINT,

    namespace VARCHAR(255) NOT NULL,

    trigger_type ENUM('manual', 'cron', 'event', 'api'),
    trigger_source VARCHAR(255),

    sleep_until BIGINT,

    trace_id VARCHAR(255),
    span_id VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id VARCHAR(255) PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') NOT NULL,
    output_data LONGBLOB,
    error_message TEXT,

    started_at BIGINT,
    completed_at BIGINT,

    max_attempts INT NOT NULL,
    remaining_attempts INT NOT NULL,

    namespace VARCHAR(255) NOT NULL
);

-- Cron Jobs Table
CREATE TABLE IF NOT EXISTS `cron_jobs` (
  `id` varchar(255) NOT NULL,
  `name` varchar(255) NOT NULL,
  `cron_spec` varchar(255) NOT NULL,
  `namespace` varchar(255) NOT NULL,
  `workflow_name` varchar(255) DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` bigint NOT NULL,
  `updated_at` bigint NOT NULL,
  `last_run_at` bigint DEFAULT NULL,
  `next_run_at` bigint NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `cron_jobs_name_namespace_idx` (`name`,`namespace`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Leases Table (step kind included for GORM compatibility, though unused)
CREATE TABLE IF NOT EXISTS leases (
    resource_id VARCHAR(255) PRIMARY KEY,
    kind ENUM('workflow', 'step', 'cron_job') NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    worker_id VARCHAR(255) NOT NULL,
    acquired_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    heartbeat_at BIGINT NOT NULL
);
