CREATE TABLE kv (
    id BIGINT(20) NOT NULL AUTO_INCREMENT,
    workspace_id VARCHAR(255) NOT NULL,
    `key` VARCHAR(255) NOT NULL,
    value BLOB NOT NULL,
    ttl BIGINT NULL,
    created_at BIGINT NOT NULL,

    PRIMARY KEY (id),
    UNIQUE KEY unique_key (`key`),
    INDEX idx_workspace_id (workspace_id),
    INDEX idx_ttl (ttl)
);
