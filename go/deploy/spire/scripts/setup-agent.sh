#!/bin/bash
# Setup script for SPIRE agent with join token
# AIDEV-NOTE: This script creates a join token and configures the agent

set -euo pipefail

# Check if server is running
if ! systemctl is-active --quiet spire-server; then
    echo "Error: SPIRE server is not running. Start it first."
    exit 1
fi

echo "Creating join token for agent..."
TOKEN=$(sudo /opt/spire/bin/spire-server token generate \
    -socketPath /run/spire/server.sock \
    -spiffeID spiffe://development.unkey.app/agent/node1 \
    -ttl 300 | awk '{print $2}')

if [ -z "$TOKEN" ]; then
    echo "Error: Failed to generate join token"
    exit 1
fi

echo "Join token created: $TOKEN"

# Create a temporary configuration with the join token
cat > /tmp/agent-join.conf <<EOF
agent {
  data_dir = "/var/lib/spire/agent"
  log_level = "INFO"
  server_address = "localhost"
  server_port = "8090"
  socket_path = "/run/spire/sockets/agent.sock"
  trust_domain = "development.unkey.app"
  trust_bundle_path = "/etc/spire/agent/bootstrap.crt"
  join_token = "$TOKEN"
}

plugins {
  NodeAttestor "join_token" {
    plugin_data {
    }
  }
  
  KeyManager "disk" {
    plugin_data {
      directory = "/var/lib/spire/agent/keys"
    }
  }
  
  WorkloadAttestor "unix" {
    plugin_data {
      discover_workload_path = true
      discover_workload_user = true
      discover_workload_group = true
    }
  }
  
  WorkloadAttestor "systemd" {
    plugin_data {
    }
  }
}

health_checks {
  listener_enabled = true
  bind_address = "127.0.0.1"
  bind_port = "8092"
}
EOF

# Copy the configuration
sudo cp /tmp/agent-join.conf /etc/spire/agent/agent.conf
sudo chown spire-agent:spire-agent /etc/spire/agent/agent.conf
sudo chmod 600 /etc/spire/agent/agent.conf

echo "Agent configuration updated with join token."
echo "You can now start the agent with: sudo systemctl start spire-agent"