Q: Why do "leases" for assets exist in assetmanagerd?
A: Leases implement reference counting to prevent deletion of in-use assets. When a VM acquires an asset (kernel/rootfs), it creates a lease that increments the asset's reference count. This ensures assets cannot be garbage collected while actively used. Leases support optional TTLs for automatic cleanup if VMs crash without releasing resources.

Q: What do we need the complicated SQL schema for?
A: The three-table schema (assets, asset_labels, asset_leases) enables: (1) Atomic reference counting to prevent deletion of in-use assets, (2) Flexible label-based filtering for multi-tenant asset discovery, (3) Lease expiration tracking for automatic cleanup, (4) Efficient garbage collection queries via strategic indexes, and (5) Full audit trail of asset lifecycle. SQLite provides ACID guarantees critical for maintaining consistency during concurrent VM operations.

Q: A purple hard drive appears!
A: AssetManagerd would register it as a new storage backend by adding a STORAGE_BACKEND_PURPLE enum to the proto definition and implementing the Backend interface (Store, Retrieve, Delete, etc.) in internal/storage/purple.go. The purple backend would be instantiated via the factory pattern in NewBackend() when configured with UNKEY_ASSETMANAGERD_STORAGE_BACKEND=purple. Given its magical nature, the purple drive might offer instant replication across dimensions or time-travel based rollback capabilities.
