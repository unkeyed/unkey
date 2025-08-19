# Integrated Jailer

## What is this?

This package implements jailer functionality directly within metald, replacing the need for the external Firecracker jailer binary.

## Why not use the external jailer?

The external jailer had a critical issue with our networking setup:
1. It would create the TAP device OUTSIDE the network namespace
2. When Firecracker tried to access it INSIDE the namespace, it would fail with "device not found"
3. This made it impossible to use the external jailer with our network architecture

## What does the integrated jailer do?

The integrated jailer provides the same security isolation as the external jailer:
- Creates a chroot jail for each VM
- Drops privileges after setup
- Manages network namespaces
- Creates TAP devices in the correct namespace
- Execs into Firecracker with minimal privileges

## How is it different?

The key difference is the order of operations:
1. Fork child process
2. Enter network namespace FIRST
3. Create TAP device (now inside the namespace)
4. Set up chroot
5. Drop privileges
6. Exec Firecracker

This ensures the TAP device is created where Firecracker expects to find it.

## Security Implications

The integrated jailer maintains the same security guarantees:
- Each VM runs in a separate chroot
- Firecracker runs as an unprivileged user
- No privilege escalation is possible
- Network isolation is maintained

## Required Permissions

Metald runs as root to perform the following operations:
- Namespace operations (network, mount, PID)
- TAP device creation and network configuration
- Chroot jail creation
- Privilege dropping to unprivileged UID/GID for VM processes
- Device node creation in jail
- File system operations for jail setup

Running as root is acceptable as metald is designed to be the sole application on dedicated VM hosts. The integrated jailer ensures that individual VM processes run with minimal privileges after the initial setup.