# SPIFFE/SPIRE Architecture for Unkey Services

## Overview

SPIFFE/SPIRE provides automatic, cryptographically-verifiable service identities without manual certificate management.

## Components

### SPIRE Server
- Central trust root
- Issues SVIDs (SPIFFE Verifiable Identity Documents)
- Manages registration entries
- Runs on dedicated host or container

### SPIRE Agents  
- One per host/node
- Attests workload identity
- Delivers SVIDs to workloads
- Handles automatic rotation

### Workloads (Your Services)
- metald, billaged, builderd, assetmanagerd
- Use Workload API to get SVIDs
- Automatic mTLS with no certificate files

## Deployment Topology

```
┌─────────────────────┐
│   SPIRE Server      │
│  (Trust Authority)  │
└──────────┬──────────┘
           │
    ┌──────┴──────┐
    │             │
┌───▼───┐    ┌───▼───┐
│ Host 1 │    │ Host 2 │
├────────┤    ├────────┤
│ Agent  │    │ Agent  │
├────────┤    ├────────┤
│metald  │    │builderd│
│billaged│    │assetmgr│
└────────┘    └────────┘
```

## Identity Scheme

### Service Identities
- Path: `/service/{name}`
- Example: `spiffe://unkey.prod/service/metald`

### Customer-Scoped Identities  
- Path: `/service/{name}/customer/{id}`
- Example: `spiffe://unkey.prod/service/metald/customer/cust-123`
- Used for VM-specific processes

### Tenant-Scoped Identities
- Path: `/service/{name}/tenant/{id}`  
- Example: `spiffe://unkey.prod/service/builderd/tenant/acme-corp`
- Used for multi-tenant isolation

## Workload Attestation

### Linux Process Attestation
- Binary path: `/usr/bin/metald`
- User/Group: `unkey-metald:unkey-metald`
- Systemd cgroup matching

### Kubernetes Attestation (Future)
- Namespace + Service Account
- Pod labels/annotations

## Benefits Over Traditional PKI

1. **Zero Certificate Management**
   - No files to distribute
   - No manual rotation
   - No passphrase management

2. **Dynamic Authorization**
   - Policy-based access control
   - Runtime identity verification
   - Automatic revocation

3. **Observability**
   - Full audit trail
   - Metrics on all mTLS connections
   - Identity-based tracing

4. **Security**
   - Short-lived credentials (1 hour)
   - Hardware-backed attestation
   - No long-lived secrets