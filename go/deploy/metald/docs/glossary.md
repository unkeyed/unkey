# Glossary

Key terms used in metald documentation.

## Backend
The underlying VMM implementation (Firecracker or Cloud Hypervisor) that metald communicates with via Unix sockets.

## ConnectRPC
HTTP/gRPC-compatible protocol used for metald's API. Supports both binary gRPC and JSON over HTTP.

## FIFO Metrics
Named pipe-based real-time metrics streaming from Firecracker VMs. Provides 100ms precision resource usage data.

## metald
The VM management service that provides a unified API across multiple VMM backends with integrated billing.

## Process Manager
Component in metald that manages dedicated Firecracker processes, one per VM for isolation.

## VM State
Current status of a VM:
- `VM_STATE_CREATED` - VM configured but not running
- `VM_STATE_RUNNING` - VM is active and running
- `VM_STATE_SHUTDOWN` - VM has been shut down
- `VM_STATE_PAUSED` - VM execution is paused

## VMM (Virtual Machine Monitor)
The hypervisor software (Firecracker, Cloud Hypervisor) that manages VM execution.

## Write-Ahead Log (WAL)
Persistence mechanism ensuring billing metrics are not lost during metald restarts or failures.