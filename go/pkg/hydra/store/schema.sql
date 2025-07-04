-- Hydra Workflow Orchestration Schema for MySQL
-- Using ENUMs for type safety and removing step input_data

-- Workflow Executions Table
CREATE TABLE workflow_executions (
    id VARCHAR(255) PRIMARY KEY,
    workflow_name VARCHAR(255) NOT NULL,
    status ENUM('pending', 'running', 'sleeping', 'completed', 'failed') NOT NULL,
    input_data VARBINARY(10485760),  -- 10MB limit for workflow inputs
    output_data VARBINARY(1048576),  -- 1MB limit for workflow outputs
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

-- Workflow Steps Table (no input_data, only output_data)
CREATE TABLE workflow_steps (
    id VARCHAR(255) PRIMARY KEY,
    execution_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    step_order INT NOT NULL,
    status ENUM('pending', 'running', 'completed', 'failed') NOT NULL,
    output_data VARBINARY(1048576),  -- 1MB limit for step outputs
    error_message TEXT,
    
    started_at BIGINT,
    completed_at BIGINT,
    
    max_attempts INT NOT NULL,
    remaining_attempts INT NOT NULL,
    
    namespace VARCHAR(255) NOT NULL
);

-- Cron Jobs Table
CREATE TABLE cron_jobs (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cron_spec VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    last_run_at BIGINT,
    next_run_at BIGINT NOT NULL
);

-- Leases Table (step kind included for GORM compatibility, though unused)
CREATE TABLE leases (
    resource_id VARCHAR(255) PRIMARY KEY,
    kind ENUM('workflow', 'step', 'cron_job') NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    worker_id VARCHAR(255) NOT NULL,
    acquired_at BIGINT NOT NULL,
    expires_at BIGINT NOT NULL,
    heartbeat_at BIGINT NOT NULL
);