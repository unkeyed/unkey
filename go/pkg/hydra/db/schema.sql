-- Hydra workflow orchestration database schema

-- Workflow executions table
CREATE TABLE `workflow_executions` (
    `id` varchar(255) NOT NULL,
    `workflow_name` varchar(255) NOT NULL,
    `status` enum('pending','running','sleeping','completed','failed') NOT NULL,
    `input_data` longblob,
    `output_data` longblob,
    `error_message` text,
    `created_at` bigint NOT NULL,
    `started_at` bigint,
    `completed_at` bigint,
    `max_attempts` int NOT NULL,
    `remaining_attempts` int NOT NULL,
    `next_retry_at` bigint,
    `namespace` varchar(255) NOT NULL,
    `trigger_type` enum('manual','cron','event','api') NOT NULL,
    `trigger_source` varchar(255),
    `sleep_until` bigint,
    `trace_id` varchar(255) NOT NULL DEFAULT '',
    PRIMARY KEY (`id`),
    INDEX `idx_workflow_namespace_status` (`namespace`, `status`, `created_at`),
    INDEX `idx_workflow_namespace_name` (`namespace`, `workflow_name`),
    INDEX `idx_workflow_status_retry` (`status`, `namespace`, `next_retry_at`),
    INDEX `idx_workflow_status_sleep` (`status`, `namespace`, `sleep_until`)
);

-- Workflow steps table
CREATE TABLE `workflow_steps` (
    `id` varchar(255) NOT NULL,
    `execution_id` varchar(255) NOT NULL,
    `step_name` varchar(255) NOT NULL,
    `step_order` int NOT NULL,
    `status` enum('pending','running','completed','failed') NOT NULL,
    `output_data` longblob,
    `error_message` text,
    `started_at` bigint,
    `completed_at` bigint,
    `max_attempts` int NOT NULL,
    `remaining_attempts` int NOT NULL,
    `namespace` varchar(255) NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_workflow_step_unique` (`namespace`, `execution_id`, `step_name`),
    INDEX `idx_step_execution_status` (`namespace`, `execution_id`, `status`)
);

-- Leases table for distributed coordination
CREATE TABLE `leases` (
    `resource_id` varchar(255) NOT NULL,
    `kind` enum('workflow','step','cron_job') NOT NULL,
    `namespace` varchar(255) NOT NULL,
    `worker_id` varchar(255) NOT NULL,
    `acquired_at` bigint NOT NULL,
    `expires_at` bigint NOT NULL,
    `heartbeat_at` bigint NOT NULL,
    PRIMARY KEY (`resource_id`),
    INDEX `idx_lease_resource_kind` (`resource_id`, `kind`),
    INDEX `idx_lease_namespace_expires` (`namespace`, `expires_at`)
);

-- Cron jobs table
CREATE TABLE `cron_jobs` (
    `id` varchar(255) NOT NULL,
    `name` varchar(255) NOT NULL,
    `cron_spec` varchar(255) NOT NULL,
    `namespace` varchar(255) NOT NULL,
    `workflow_name` varchar(255) NOT NULL,
    `enabled` boolean NOT NULL DEFAULT true,
    `created_at` bigint NOT NULL,
    `updated_at` bigint NOT NULL,
    `last_run_at` bigint,
    `next_run_at` bigint NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_namespace_name` (`namespace`, `name`),
    INDEX `idx_cron_namespace_enabled_next` (`namespace`, `enabled`, `next_run_at`)
);