Q: What is left to integrate with builderd and assetmanagerd?
A: The integration with assetmanagerd is partially implemented with a client interface defined in `internal/assetmanager/client.go`. The client supports listing assets, preparing assets, acquiring leases, and releasing assets. However, the actual integration points in the VM creation flow are currently stubbed. 

For builderd integration:
1. No direct client implementation exists yet - builderd would likely be called by assetmanagerd rather than directly by metald
2. The workflow would be: metald → assetmanagerd → builderd for custom image builds
3. Missing implementation includes:
   - Asset preparation calls during VM creation in `internal/service/vm.go:48-159`
   - Asset cleanup during VM deletion
   - Proper error handling for asset preparation failures
   - Configuration for custom kernel/rootfs builds via builderd

The current implementation uses static paths for kernel and rootfs rather than dynamically prepared assets from the asset management pipeline.
