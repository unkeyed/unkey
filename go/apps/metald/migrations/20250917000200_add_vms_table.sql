-- +goose Up
-- Table to track VM instances and their configuration
CREATE TABLE vms (
  vm_id TEXT PRIMARY KEY,                      -- The VM identifier (e.g., "ud-xxx")
  deployment_id TEXT NOT NULL,                 -- References the deployment

  -- VM Configuration from VmConfig
  vcpu_count INTEGER NOT NULL,                 -- Number of vCPUs
  memory_size_mib INTEGER NOT NULL,            -- Memory size in MiB
  boot TEXT NOT NULL,                          -- Boot configuration/image
  network_config TEXT,                         -- Network configuration (JSON or plain text)
  console_config TEXT,                         -- Console configuration (JSON)
  storage_config TEXT,                         -- Storage configuration (JSON)
  metadata TEXT,                                -- Metadata key-value pairs (JSON)

  -- Network allocation
  ip_address TEXT,                             -- Assigned IP address
  bridge_name TEXT,                            -- Bridge name from network allocation

  -- Status tracking
  status INTEGER NOT NULL DEFAULT 0,           -- VM status: 0=created, 1=running, 2=stopped, 3=failed, 4=terminated
  error_message TEXT,                          -- Error message if status is failed

  -- Timestamps (Unix timestamps)
  created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
  started_at INTEGER,                          -- When VM was started
  stopped_at INTEGER,                          -- When VM was stopped

  FOREIGN KEY (deployment_id) REFERENCES network_allocations(deployment_id)
);

-- Indexes for performance
CREATE INDEX idx_vms_deployment
  ON vms(deployment_id);

CREATE INDEX idx_vms_status
  ON vms(status);

-- +goose Down
DROP TABLE IF EXISTS vms;