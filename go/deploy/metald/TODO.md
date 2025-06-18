# Metald TODO

## High Priority

- [ ] Implement VM migration
  - Live migration support
  - Storage migration
  - Network state preservation

- [ ] Add VM snapshot functionality
  - Memory snapshots
  - Disk snapshots
  - Snapshot scheduling

- [ ] Implement proper VM networking
  - VLAN support
  - SDN integration
  - IPv6 support

## Medium Priority

- [ ] Add VM template support
  - Template creation from running VMs
  - Template versioning
  - Template marketplace

- [ ] Implement VM clustering
  - High availability for VMs
  - Automatic failover
  - Distributed storage support

- [ ] Add VM resource hotplug
  - CPU hotplug
  - Memory hotplug
  - Disk hotplug

## Low Priority

- [ ] Add support for Cloud Hypervisor
  - Backend abstraction complete
  - Feature parity with Firecracker
  - Performance optimization

- [ ] Implement VM console access
  - VNC/SPICE support
  - Web-based console
  - Console recording

- [ ] Add VM backup integration
  - Scheduled backups
  - Incremental backups
  - Backup restoration

## Completed

- [x] Basic VM lifecycle management
- [x] Firecracker integration
- [x] Jailer security integration
- [x] Network namespace support
- [x] Asset manager integration
- [x] Billing service integration
- [x] Builder service integration
- [x] ConnectRPC API
- [x] Prometheus metrics
- [x] SPIFFE/mTLS support
- [x] Grafana dashboards
- [x] Unified health endpoint
- [x] Unified Makefile structure