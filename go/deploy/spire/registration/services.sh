#!/bin/bash
# SPIRE Registration Entries for Unkey Services
# AIDEV-NOTE: Defines how each service gets its SPIFFE ID

SPIRE_SERVER="spire-server.unkey.internal:8081"

# Register node agents first (one per host)
spire-server entry create \
  -node \
  -spiffeID spiffe://unkey.prod/agent/node1 \
  -selector join_token:xxxx-node1-token

# metald service
spire-server entry create \
  -parentID spiffe://unkey.prod/agent/node1 \
  -spiffeID spiffe://unkey.prod/service/metald \
  -selector unix:path:/usr/bin/metald \
  -selector unix:user:unkey-metald \
  -selector systemd:unit:metald.service \
  -ttl 3600

# billaged service  
spire-server entry create \
  -parentID spiffe://unkey.prod/agent/node1 \
  -spiffeID spiffe://unkey.prod/service/billaged \
  -selector unix:path:/usr/bin/billaged \
  -selector unix:user:unkey-billaged \
  -selector systemd:unit:billaged.service \
  -ttl 3600

# builderd service
spire-server entry create \
  -parentID spiffe://unkey.prod/agent/node1 \
  -spiffeID spiffe://unkey.prod/service/builderd \
  -selector unix:path:/usr/bin/builderd \
  -selector unix:user:unkey-builderd \
  -selector systemd:unit:builderd.service \
  -ttl 3600

# assetmanagerd service
spire-server entry create \
  -parentID spiffe://unkey.prod/agent/node1 \
  -spiffeID spiffe://unkey.prod/service/assetmanagerd \
  -selector unix:path:/usr/bin/assetmanagerd \
  -selector unix:user:unkey-assetmanagerd \
  -selector systemd:unit:assetmanagerd.service \
  -ttl 3600

# Dynamic VM workloads spawned by metald
# Uses path prefix for Firecracker VMs
spire-server entry create \
  -parentID spiffe://unkey.prod/agent/node1 \
  -spiffeID spiffe://unkey.prod/service/metald/customer/${CUSTOMER_ID} \
  -selector unix:path:/var/lib/metald/vms/*/firecracker \
  -selector unix:user:unkey-metald \
  -ttl 3600