-- +goose Up
-- Initial schema for metald network management
CREATE TABLE networks (
  id INTEGER PRIMARY KEY,
  base_network TEXT UNIQUE NOT NULL,  -- CIDR notation: "10.0.0.16/28"
  is_allocated INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE network_allocations (
  id INTEGER PRIMARY KEY,
  deployment_id TEXT NOT NULL UNIQUE,
  network_id INTEGER NOT NULL,
  bridge_name TEXT NOT NULL UNIQUE,
  available_ips TEXT NOT NULL,  -- JSON array: ["10.0.0.18","10.0.0.19",...]
  allocated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (network_id) REFERENCES networks(id)
);

CREATE TABLE ip_allocations (
  id INTEGER PRIMARY KEY,
  vm_id TEXT NOT NULL UNIQUE,
  ip_addr TEXT NOT NULL,
  network_allocation_id INTEGER NOT NULL,
  allocated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (network_allocation_id) REFERENCES network_allocations(id),
  UNIQUE(network_allocation_id, ip_addr)
);

-- Indexes for performance
CREATE INDEX idx_networks_unallocated
  ON networks(is_allocated, id)
  WHERE is_allocated = 0;

CREATE INDEX idx_ip_allocations_network
  ON ip_allocations(network_allocation_id);

CREATE INDEX idx_network_allocations_deployment
  ON network_allocations(deployment_id);

-- +goose Down
-- Drop all tables
DROP TABLE IF EXISTS ip_allocations;
DROP TABLE IF EXISTS network_allocations;
DROP TABLE IF EXISTS networks;
