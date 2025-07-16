# Unkey Services Local Development Environment Setup

This guide provides detailed instructions for setting up a complete Unkey development environment on Linux (Fedora 42 or Ubuntu 22.04+).

## Prerequisites

### 1. System Requirements Check

Before beginning installation, verify your system meets all requirements:

```bash
cd /path/to/unkey/go/deploy
./scripts/check-system-readiness.sh
```

This script verifies:
- Operating system compatibility (Fedora 42+ or Ubuntu 24.04+)
- Required tools: Go 1.24+, Make, Git, systemd
- Container runtime: Docker or Podman
- Virtualization: Firecracker/Cloud Hypervisor, KVM support
- Port availability: 8080-8085, 9464-9467
- Disk space: minimum 5GB free
- Network connectivity

### 2. Fix Any Missing Prerequisites

If the readiness check reports missing dependencies:

<details>
<summary><b>For Fedora</b></summary>

```bash
# Install development tools
sudo dnf group install -y development-tools
sudo dnf install -y git make curl wget iptables-legacy

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh
```

#### Install Docker (Official Method)

Follow the official Docker installation for Fedora:

```bash
# Remove old versions
sudo dnf remove docker \
                docker-client \
                docker-client-latest \
                docker-common \
                docker-latest \
                docker-latest-logrotate \
                docker-logrotate \
                docker-selinux \
                docker-engine-selinux \
                docker-engine

# Set up the Docker repository
sudo dnf -y install dnf-plugins-core
sudo dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo

# Install Docker Engine
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker

# Add your user to the docker group
sudo usermod -aG docker $USER

# Verify installation
sudo docker run hello-world
```

#### Complete Setup

```bash
# Install KVM support
sudo dnf install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# Log out and back in for group changes to take effect
```

</details>

### 3. Firecracker Setup (REQUIRED)

Metald uses an integrated jailer approach that handles VM isolation automatically:

```bash
# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh
```

**Notes**:
- Metald v0.2.0+ includes integrated jailer functionality
- No manual jailer user or cgroup configuration needed
- The system automatically handles VM isolation and security

<details>
<summary><b>For Ubuntu</b></summary>

```bash
# Install development tools
sudo apt update
sudo apt install -y build-essential git make golang curl wget

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh
```

#### Install Docker (Official Method)

Follow the official Docker installation for Ubuntu:

```bash
# Remove old versions
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
  sudo apt-get remove $pkg;
done

# Update package index and install prerequisites
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# Add Docker's official GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Set up the repository
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker Engine
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

#### Complete Setup

```bash
# Install KVM support
sudo apt install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker
sudo ./scripts/install-firecracker.sh
```

</details>

## Service Installation Order

1. **Observability Stack** (optional but recommended) - Grafana, Loki, Tempo, Mimir
2. **SPIRE** (REQUIRED) - Service identity and mTLS for secure inter-service communication
3. **assetmanagerd** - VM asset management
4. **billaged** - Usage billing service
5. **builderd** - Container build service
6. **metald** - VM management service (depends on assetmanagerd and billaged)

## Quick Installation

Run from the `/path/to/unkey/go/deploy` directory:

### Step 1: Observability Stack

```bash
# Start observability stack
make o11y
```

Access Grafana at `http://localhost:3000` (admin/admin)

### Step 2: SPIRE Installation and Setup

```bash
# Install and start SPIRE with all services registered
make -C spire install
make -C spire service-start-server
make -C spire register-agent
make -C spire service-start-agent
make -C spire register-services
```

### Step 3: Install Services/Clients

```bash
# Install all services
make assetmanagerd-install
make builderd-install
make billaged-install
make metald-install

# Install all Clients
make -C assetmanagerd/client install
make -C builderd/client install
make -C billaged/client install
make -C metaldd/client install
```

### Step 4: Launch a MicroVM

```bash
metald-cli -docker-image=ghcr.io/unkeyed/best-api:v1.1.0 create-and-boot
```

You should see output similar to:

```bash
