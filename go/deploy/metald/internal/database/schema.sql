-- VM state storage schema
CREATE TABLE IF NOT EXISTS vms (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    config BLOB NOT NULL,
    state INTEGER NOT NULL DEFAULT 0,
    process_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Index for efficient customer queries
CREATE INDEX IF NOT EXISTS idx_vms_customer_id ON vms(customer_id);

-- Index for state queries
CREATE INDEX IF NOT EXISTS idx_vms_state ON vms(state);

-- Index for process queries
CREATE INDEX IF NOT EXISTS idx_vms_process_id ON vms(process_id);

-- Composite index for customer + state queries
CREATE INDEX IF NOT EXISTS idx_vms_customer_state ON vms(customer_id, state);