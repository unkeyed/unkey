# Network and System Resource Naming Restrictions

This document outlines the exact naming restrictions for various system resources used in the metald codebase.

## 1. Network Interface Names (TAP, veth, bridge)

### Maximum Length
- **15 characters** (defined by `IFNAMSIZ` in Linux kernel)
- This includes the null terminator, so effectively 15 visible characters

### Allowed Characters
- Letters: `a-z`, `A-Z`
- Numbers: `0-9`
- Hyphen: `-` (preferred over underscore)
- Underscore: `_` (allowed but not recommended)
- Period: `.` (allowed but can cause issues with some tools)

### Restrictions
- Cannot start with a number
- Cannot contain spaces
- Cannot contain special characters like `@`, `#`, `$`, etc.
- Cannot be empty
- Cannot use reserved names (e.g., `lo`, `eth0`, etc.)

### Best Practices
- Use lowercase letters
- Use hyphens instead of underscores (kernel convention)
- Keep names short and descriptive
- Avoid dots as they can conflict with VLAN naming

### Examples in metald
```go
// From internal/network/implementation.go
tapName := fmt.Sprintf("tap%s", vmID[:8])      // e.g., "tap12345678"
vethHost := fmt.Sprintf("vh%s", suffix)         // e.g., "vh12345678"
vethNS := fmt.Sprintf("vn%s", suffix)           // e.g., "vn12345678"
bridgeName := "br-vms"                          // Default bridge name
```

## 2. Network Namespace Names

### Maximum Length
- **255 characters** (NAME_MAX on most filesystems)
- Stored as files in `/var/run/netns/`

### Allowed Characters
- Letters: `a-z`, `A-Z`
- Numbers: `0-9`
- Hyphen: `-`
- Underscore: `_`
- Period: `.`

### Restrictions
- Cannot contain `/` (path separator)
- Cannot be `.` or `..`
- Cannot contain null bytes
- Must be valid filesystem names

### Best Practices
- Use consistent prefixing (e.g., `vm-<id>`)
- Avoid special characters
- Keep reasonably short for readability

### Examples in metald
```go
// From internal/network/implementation.go
nsName := fmt.Sprintf("vm-%s", vmID)  // e.g., "vm-550e8400-e29b-41d4-a716"
```

## 3. Unix Domain Socket Paths

### Maximum Length
- **108 characters** total path length (defined by `UNIX_PATH_MAX`)
- This includes the null terminator, so effectively 107 visible characters
- For abstract sockets (starting with null byte): 108 bytes

### Structure
- The limit applies to the complete path, not just the filename
- Common directories:
  - `/var/run/`: 9 characters
  - `/tmp/`: 5 characters
  - `/run/`: 5 characters

### Allowed Characters
- Any valid filesystem path characters
- Letters, numbers, hyphens, underscores, periods, slashes

### Restrictions
- Total path must not exceed 108 characters
- Must be accessible (permissions)
- Directory must exist
- For regular sockets, follows filesystem naming rules

### Best Practices
- Keep socket names short
- Use shallow directory structures
- Consider using `/run/` instead of `/var/run/` to save 4 characters
- Account for dynamic parts (IDs, timestamps) in length calculations

### Examples in metald
```go
// From internal/process/manager.go
// Without jailer:
socketPath = filepath.Join(m.socketDir, processID+".sock")
// e.g., "/var/run/metald/fc-1234567890.sock" (36 chars)

// With jailer (longer paths):
socketPath = filepath.Join(chrootPath, "root", "run", "firecracker.socket")
// e.g., "/srv/jailer/firecracker/vm-1234567890/root/run/firecracker.socket" (66 chars)
```

## 4. Process Names

### Maximum Length
- **15 characters** visible in most tools (defined by kernel's `TASK_COMM_LEN`)
- **4096 characters** for full command line (`ARG_MAX` related)

### Display Limitations
- `ps`, `top`, and similar tools typically show only 15 characters
- `/proc/PID/comm` is limited to 15 characters
- `/proc/PID/cmdline` shows full command line

### Allowed Characters
- Any printable ASCII characters
- Spaces are allowed in arguments but not in the binary name

### Best Practices
- Keep binary names under 15 characters for visibility
- Use descriptive names that are meaningful when truncated
- Consider how the name appears in monitoring tools

### Examples in metald
```go
// Process names used:
"firecracker"     // 11 chars - good
"jailer"          // 6 chars - good
"metald"          // 6 chars - good
"assetmanagerd"   // 14 chars - good
"billaged"        // 8 chars - good
```

## Summary Table

| Resource Type | Max Length | Key Restrictions | Example |
|--------------|------------|------------------|---------|
| Network Interface | 15 chars | No spaces, alphanumeric + `-_` | `tap12345678` |
| Network Namespace | 255 chars | Valid filename | `vm-550e8400` |
| Unix Socket Path | 108 chars | Full path limit | `/run/fc-123.sock` |
| Process Name | 15 chars visible | Truncated in tools | `firecracker` |

## AIDEV-NOTE: Critical Naming Considerations

1. **Network Interfaces**: The 15-character limit is strict. Always validate interface names before creation.

2. **Socket Paths**: When using jailer, the chroot path can be long. Always calculate total path length:
   - Base: `/srv/jailer/firecracker/` (24 chars)
   - Jailer ID: `vm-1234567890` (13 chars typical)
   - Socket: `/root/run/firecracker.socket` (28 chars)
   - Total: ~65 characters (well within limit but be careful with longer IDs)

3. **Consistency**: The codebase uses hyphens (`-`) in most places (tap-, vm-, vh-, vn-). Maintain this convention.

4. **ID Truncation**: When using VM IDs in network names, the code truncates to 8 characters to ensure the total name fits within limits.