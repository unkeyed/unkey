# VMM Control Plane Glossary

This glossary provides definitions for virtualization, hardware, and cloud computing terms used throughout the VMM Control Plane documentation and unified API.

## A

### AMX (Advanced Matrix Extensions)
Intel's instruction set architecture extension designed for accelerating artificial intelligence and machine learning workloads. AMX introduces new 2-dimensional registers (tiles) and instructions for matrix operations, particularly useful for deep learning inference and training.

### Affinity (CPU Affinity)
The assignment of virtual CPUs (vCPUs) to specific physical CPU cores. CPU affinity can improve performance by ensuring cache locality and reducing context switching overhead. Also known as CPU pinning.

### API Abstraction
The VMM Control Plane's approach of providing a single, unified API that works across multiple hypervisor backends without exposing backend-specific implementation details.

## B

### Backend Interface
The abstract interface in VMM Control Plane that defines standard VM operations (Create, Boot, Shutdown, etc.) implemented by different hypervisor backends like Cloud Hypervisor and Firecracker.

### Backend Selection
The process of choosing which hypervisor implementation to use, controlled via the `UNKEY_VMCP_BACKEND` environment variable. Supports `cloudhypervisor` and `firecracker`.

### Balloon Device
A virtio device that allows dynamic memory management between the host and guest. The balloon driver can "inflate" to return unused guest memory to the host, or "deflate" to reclaim memory when the guest needs it.

### Boot vCPUs
The number of virtual CPUs available to a VM at boot time. This can be different from the maximum vCPUs, allowing for CPU hotplug functionality.

## C

### Cloud Hypervisor
A Virtual Machine Monitor (VMM) that runs on top of KVM and provides a minimal, security-focused hypervisor for running modern cloud workloads. Written in Rust and designed for simplicity and security.

### ConnectRPC
A protocol-buffer-first RPC framework that works over HTTP/1.1 and HTTP/2. Used for VMM Control Plane's unified API implementation, providing both gRPC and HTTP compatibility.

### Console Device
A character device that provides interactive access to a VM's console output and input. Can be configured as TTY (terminal), file-based, or null device.

### CPU Topology
The hierarchical organization of CPU resources including sockets, cores, threads, and dies. Proper topology configuration can improve performance for NUMA-aware applications.

## D

### Direct I/O
A feature that bypasses the kernel's page cache for disk operations, allowing data to be transferred directly between the disk and the application's memory. Useful for databases and applications that manage their own caching.

### Dies per Package
In modern CPU architectures, a package (physical CPU) may contain multiple dies (silicon chips). This affects cache hierarchy and inter-core communication latency.

## F

### Firecracker
AWS's open-source microVM technology optimized for serverless computing. Provides ultra-fast startup times (~50-100ms) and minimal memory overhead, making it ideal for short-lived, stateless workloads.

### Firmware
Low-level software that provides runtime services for operating systems and programs. In virtualization, this typically refers to UEFI or BIOS implementations like OVMF.

## H

### Hotplug
The ability to add or remove resources (CPU, memory, devices) from a running VM without requiring a reboot. Supported by Cloud Hypervisor but not Firecracker.

### Hugepages
Memory pages larger than the standard 4KB size (typically 2MB or 1GB on x86_64). Hugepages reduce TLB (Translation Lookaside Buffer) pressure and can significantly improve performance for memory-intensive applications.

### Hypervisor
Software that creates and manages virtual machines by abstracting the physical hardware. Also known as a Virtual Machine Monitor (VMM).

### Hypervisor Abstraction
VMM Control Plane's architectural approach of providing a common interface for multiple hypervisor backends, allowing applications to use different VMM technologies through the same API.

## I

### IGVM (Independent Guest Virtual Machine)
A file format for packaging confidential computing workloads that can run on different confidential computing platforms.

### initramfs (Initial RAM File System)
A temporary root file system loaded into memory during the Linux boot process. Contains necessary drivers and scripts to mount the real root file system.

### IOMMU (Input-Output Memory Management Unit)
Hardware that provides memory protection and address translation for DMA-capable I/O devices. Required for secure device passthrough to VMs.

## K

### KVM (Kernel-based Virtual Machine)
A virtualization infrastructure for the Linux kernel that turns it into a hypervisor. Both Cloud Hypervisor and Firecracker use KVM for hardware virtualization support.

## M

### Memory Zones
Distinct regions of memory that can have different properties (NUMA node affinity, hugepage backing, file backing). Used for fine-grained memory management in VMs.

### microVM
A lightweight virtual machine optimized for fast startup and minimal resource usage. Firecracker is the primary implementation of the microVM concept, designed for serverless workloads.

## N

### NUMA (Non-Uniform Memory Access)
A computer memory design where memory access time depends on the memory location relative to the processor. Important for optimizing performance in multi-socket systems.

### Network TAP Device
A virtual network interface that simulates a link layer device. TAP devices transport Ethernet frames and are commonly used to provide network connectivity to VMs.

## P

### Passthrough (Device Passthrough)
Assigning a physical device directly to a VM, bypassing the hypervisor. Provides near-native performance for devices like GPUs or network cards.

### PCI (Peripheral Component Interconnect)
A standard for connecting peripheral devices to a computer. Virtual machines emulate PCI buses for attaching virtual devices.

### Protobuf (Protocol Buffers)
Google's language-neutral, platform-neutral, extensible mechanism for serializing structured data. Used by VMM Control Plane for defining the unified API schema.

## Q

### Queue Size
The number of descriptors in a virtio queue. Larger queues can improve throughput but increase memory usage.

## R

### Rate Limiter
A mechanism to control the rate of I/O operations or bandwidth usage. Uses token bucket algorithms to enforce limits on disk or network throughput.

### RNG (Random Number Generator)
A device that generates random numbers. In VMs, typically provided through virtio-rng using the host's entropy sources.

## S

### SEV (Secure Encrypted Virtualization)
AMD's technology for encrypting VM memory to protect it from unauthorized access, including from the hypervisor.

### SGX (Software Guard Extensions)
Intel's set of CPU instructions that enable applications to create secure enclaves - protected areas of memory that are encrypted and isolated from the rest of the system.

### Shared Memory
Memory regions that can be accessed by both the host and guest, or shared between multiple guests. Useful for high-performance inter-VM communication.

### SR-IOV (Single Root I/O Virtualization)
A PCI Express standard that allows a physical device to appear as multiple separate virtual devices, enabling efficient device sharing among VMs.

### Startup Time
The time required to boot a VM from creation to running state. Firecracker excels with ~50-100ms startup times, while Cloud Hypervisor typically takes 300-500ms.

## T

### Threads per Core
The number of simultaneous threads that can execute on a single CPU core. Modern processors often support SMT (Simultaneous Multi-Threading) with 2 threads per core.

### Token Bucket
An algorithm for rate limiting that uses tokens to represent permission to transfer data. Tokens are added at a constant rate and consumed by operations.

### TTY (Teletypewriter)
A terminal device that provides text-based interaction with a system. In VMs, the console can be configured to use a TTY for interactive access.

## U

### Unified API
VMM Control Plane's approach of providing a single API interface that works across multiple hypervisor backends, abstracting implementation differences while maintaining consistent functionality.

### Unix Socket
A inter-process communication mechanism that allows processes on the same host to communicate. Used by VMM Control Plane to communicate with hypervisor processes.

## V

### vCPU (Virtual CPU)
A virtualized processor core assigned to a VM. Multiple vCPUs can be mapped to physical CPU cores through the hypervisor's scheduler.

### VFIO (Virtual Function I/O)
A framework for secure device passthrough that provides IOMMU-based isolation and allows userspace drivers to directly access hardware devices.

### vhost-user
A protocol for offloading virtio device emulation to a separate process. Commonly used for high-performance networking with DPDK.

### virtio
A standardized interface for virtual devices in paravirtualized environments. Provides efficient I/O by minimizing exits to the hypervisor.

### VLAN (Virtual Local Area Network)
A logical network segment that operates at Layer 2 of the OSI model. VMs can be configured to use VLAN tagging for network isolation.

### VM (Virtual Machine)
An emulated computer system that runs on physical hardware through a hypervisor. Provides isolation and resource management for running operating systems and applications.

### VMM (Virtual Machine Monitor)
Another term for hypervisor. The software layer responsible for creating and managing virtual machines.

### VMCP (VMM Control Plane)
This project - a unified control plane that provides a single API for managing VMs across multiple hypervisor backends including Cloud Hypervisor and Firecracker.

### vsock
A socket family for communication between VMs and hosts, providing a simple, efficient channel for inter-VM and host-guest communication without requiring network configuration.

## W

### Workload Pattern
The characteristic resource usage and behavior of different types of applications:
- **Serverless**: Short-lived, stateless functions (ideal for Firecracker)
- **Persistent**: Long-running, stateful applications (ideal for Cloud Hypervisor)
- **Development**: Interactive, debugging-friendly environments
- **Production**: High-performance, feature-rich deployments

## Control Plane Terms

### Backend Implementation
The concrete implementation of the Backend interface for a specific hypervisor (e.g., Cloud Hypervisor Client, Firecracker Client).

### Configuration Validation
The process of ensuring VM configurations are valid and compatible with the selected backend before attempting VM creation.

### Environment Variable Configuration
VMM Control Plane's approach to runtime configuration using environment variables with the `UNKEY_VMCP_` prefix.

### Service Layer
The business logic layer in VMM Control Plane that handles request validation, backend routing, and response formatting.

## API and Protocol Terms

### gRPC Compatibility
VMM Control Plane's ConnectRPC implementation provides full gRPC compatibility while also supporting HTTP/1.1 and standard HTTP clients.

### HTTP/2 Support
VMM Control Plane supports HTTP/2 for improved performance with multiplexed requests and server push capabilities.

### Protobuf Schema
The structured definition of API messages and services using Protocol Buffers, providing strong typing and backward compatibility.

### Request Validation
The process of verifying API requests conform to the expected schema and business rules before processing.

## Observability Terms

### OTEL (OpenTelemetry)
An observability framework for cloud-native software providing APIs, libraries, and tools to collect distributed traces, metrics, and logs.

### OTLP (OpenTelemetry Protocol)
The protocol used for transmitting telemetry data from sources to backends. Supports both gRPC and HTTP transports.

### Parent-based Sampling
A trace sampling strategy that honors the sampling decision from upstream services while applying local sampling rules to root spans.

### Span
A single operation within a distributed trace, containing timing information, attributes, and relationships to other spans.

### Trace
A collection of spans that represents the journey of a request through a distributed system.

### Metrics Cardinality
The number of unique combinations of metric labels. High cardinality can impact performance and storage.

### Prometheus
An open-source monitoring system that collects metrics by scraping HTTP endpoints at configured intervals.

### Grafana LGTM Stack
An all-in-one observability stack containing Loki (logs), Grafana (visualization), Tempo (traces), and Mimir (metrics).

## Performance Terms

### Cold Start
The time required to start a completely new VM instance. Firecracker excels at minimizing cold start times for serverless workloads.

### Memory Overhead
The additional memory required by the hypervisor and VMM infrastructure beyond the VM's allocated memory.

### Density
The number of VMs that can be efficiently run on a single host. Firecracker's low overhead enables higher VM density.

### Hot Path
Code paths that are executed frequently and impact performance. Both hypervisors optimize hot paths for VM operations.

## Security Terms

### Attack Surface
The total number of possible points where an unauthorized user can try to enter or extract data. Firecracker minimizes attack surface through minimal device emulation.

### Isolation Boundary
The security perimeter that separates different workloads or tenants. Both hypervisors provide strong isolation through hardware virtualization.

### Privilege Separation
The practice of running different components with minimal required privileges. VMM Control Plane backends run with limited privileges.

## Architecture Patterns

### Backend Abstraction Pattern
The design pattern used by VMM Control Plane to provide a common interface across different hypervisor implementations.

### Factory Pattern
Used in backend selection where the appropriate hypervisor client is instantiated based on configuration.

### Service Composition
The architectural approach of building complex functionality through composed, single-purpose services.

## See Also

- [Cloud Hypervisor Documentation](https://github.com/cloud-hypervisor/cloud-hypervisor)
- [Firecracker Documentation](https://github.com/firecracker-microvm/firecracker)
- [KVM Documentation](https://www.kernel.org/doc/html/latest/virt/kvm/)
- [Virtio Specification](https://docs.oasis-open.org/virtio/virtio/v1.2/virtio-v1.2.html)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [ConnectRPC Documentation](https://connect.build/)
- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers)