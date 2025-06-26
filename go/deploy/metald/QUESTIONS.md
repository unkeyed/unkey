Q: What is left to integrate with builderd and assetmanagerd?
A: The integration with assetmanagerd is now complete! The implementation includes:

**Completed Integration:**
1. Asset discovery and matching based on VM configuration metadata (including Docker image labels)
2. Dynamic asset preparation through assetmanagerd before VM creation
3. Asset lease acquisition after successful VM boot (to avoid holding leases for failed VMs)
4. Asset lease release during VM deletion
5. Fallback to static file copying when assetmanager is disabled (backward compatibility)

**Implementation Details:**
- `internal/backend/firecracker/sdk_client_v4.go` now uses the assetmanager client
- `prepareVMAssets()` queries assetmanagerd for matching assets and prepares them
- `BootVM()` acquires asset leases after successful boot
- `DeleteVM()` releases asset leases during cleanup
- Asset requirements are built from VM config metadata (e.g., docker_image labels)

**For builderd integration:**
The workflow is: metald → assetmanagerd → builderd
- Metald doesn't call builderd directly
- When metald requests an asset that doesn't exist, assetmanagerd could trigger builderd
- This allows for on-demand rootfs creation from Docker images

**Future Enhancements:**
1. Add support for triggering builds through assetmanagerd when assets don't exist
2. Better asset ID to component mapping (kernel vs rootfs identification)
3. Metrics for asset preparation time and cache hit rates
4. Configuration option to enable/disable automatic build triggering
