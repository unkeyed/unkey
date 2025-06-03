-- Create deployments table
CREATE TABLE deployments (
    id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    current_step VARCHAR(50) NOT NULL,
    source_config JSON NOT NULL,
    build_config JSON NOT NULL,
    runtime_config JSON NOT NULL,
    resource_config JSON NOT NULL,
    metadata JSON NOT NULL,
    steps JSON NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    INDEX idx_customer_id (customer_id),
    INDEX idx_project_id (project_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_customer_status (customer_id, status),
    INDEX idx_project_status (project_id, status)
);

-- Create deployment_events table for audit trail
CREATE TABLE deployment_events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    step VARCHAR(50) NULL,
    status VARCHAR(50) NOT NULL,
    event_data JSON NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_deployment_id (deployment_id),
    INDEX idx_customer_id (customer_id),
    INDEX idx_project_id (project_id),
    INDEX idx_event_type (event_type),
    INDEX idx_created_at (created_at),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Create deployment_logs table for streaming logs
CREATE TABLE deployment_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    step VARCHAR(50) NOT NULL,
    log_level VARCHAR(20) NOT NULL DEFAULT 'INFO',
    message TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_deployment_id (deployment_id),
    INDEX idx_deployment_step (deployment_id, step),
    INDEX idx_timestamp (timestamp),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Create deployment_resources table for tracking allocated resources
CREATE TABLE deployment_resources (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL, -- 'container', 'load_balancer', 'storage', etc.
    resource_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL, -- 'aws', 'gcp', 'azure', 'kubernetes', etc.
    region VARCHAR(100) NOT NULL,
    configuration JSON NOT NULL,
    status VARCHAR(50) NOT NULL, -- 'provisioning', 'active', 'failed', 'terminated'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    terminated_at TIMESTAMP NULL,
    INDEX idx_deployment_id (deployment_id),
    INDEX idx_resource_type (resource_type),
    INDEX idx_provider (provider),
    INDEX idx_status (status),
    UNIQUE KEY uk_deployment_resource (deployment_id, resource_type, resource_id),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Create deployment_metrics table for performance tracking
CREATE TABLE deployment_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    deployment_id VARCHAR(255) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(20,6) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    tags JSON NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_deployment_id (deployment_id),
    INDEX idx_metric_name (metric_name),
    INDEX idx_timestamp (timestamp),
    INDEX idx_deployment_metric (deployment_id, metric_name),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

-- Create deployment_webhooks table for customer notifications
CREATE TABLE deployment_webhooks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NULL,
    webhook_url VARCHAR(2048) NOT NULL,
    secret_key VARCHAR(255) NOT NULL,
    events JSON NOT NULL, -- Array of event types to subscribe to
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_customer_id (customer_id),
    INDEX idx_project_id (project_id),
    INDEX idx_active (is_active)
);