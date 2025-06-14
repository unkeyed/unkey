#!/bin/bash
set -e

echo "=== Testing Dual-Stack (IPv4 + IPv6) MicroVM Networking ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root or with sudo"
   exit 1
fi

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

echo "1. Setting up dual-stack host networking..."

# Enable IP forwarding for both IPv4 and IPv6
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding
echo -e "${GREEN}✓ IP forwarding enabled (IPv4 + IPv6)${NC}"

# Create bridge if it doesn't exist
if ! ip link show br-vms >/dev/null 2>&1; then
    ip link add name br-vms type bridge
    echo -e "${GREEN}✓ Bridge br-vms created${NC}"
else
    echo -e "${GREEN}✓ Bridge br-vms already exists${NC}"
fi

# Configure IPv4 on bridge
if ! ip addr show br-vms | grep -q "10.100.0.1/16"; then
    ip addr add 10.100.0.1/16 dev br-vms
    echo -e "${GREEN}✓ IPv4 address configured on bridge${NC}"
fi

# Configure IPv6 on bridge
if ! ip addr show br-vms | grep -q "fd00::1/64"; then
    ip addr add fd00::1/64 dev br-vms
    echo -e "${GREEN}✓ IPv6 address configured on bridge${NC}"
fi

# Bring bridge up
ip link set br-vms up

# Get default interface
DEFAULT_IF=$(ip route | grep default | awk '{print $5}' | head -1)
echo "Default interface: $DEFAULT_IF"

# Setup IPv4 NAT
if ! iptables -t nat -C POSTROUTING -s 10.100.0.0/16 -o $DEFAULT_IF -j MASQUERADE 2>/dev/null; then
    iptables -t nat -A POSTROUTING -s 10.100.0.0/16 -o $DEFAULT_IF -j MASQUERADE
    echo -e "${GREEN}✓ IPv4 NAT rule added${NC}"
else
    echo -e "${GREEN}✓ IPv4 NAT rule already exists${NC}"
fi

# Setup IPv6 NAT (using masquerade for simplicity)
if ! ip6tables -t nat -C POSTROUTING -s fd00::/64 -o $DEFAULT_IF -j MASQUERADE 2>/dev/null; then
    ip6tables -t nat -A POSTROUTING -s fd00::/64 -o $DEFAULT_IF -j MASQUERADE
    echo -e "${GREEN}✓ IPv6 NAT rule added${NC}"
else
    echo -e "${GREEN}✓ IPv6 NAT rule already exists${NC}"
fi

# Setup IPv4 forwarding rules
if ! iptables -C FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT 2>/dev/null; then
    iptables -A FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT
    iptables -A FORWARD -i $DEFAULT_IF -o br-vms -m state --state RELATED,ESTABLISHED -j ACCEPT
    echo -e "${GREEN}✓ IPv4 forwarding rules added${NC}"
else
    echo -e "${GREEN}✓ IPv4 forwarding rules already exist${NC}"
fi

# Setup IPv6 forwarding rules
if ! ip6tables -C FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT 2>/dev/null; then
    ip6tables -A FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT
    ip6tables -A FORWARD -i $DEFAULT_IF -o br-vms -m state --state RELATED,ESTABLISHED -j ACCEPT
    echo -e "${GREEN}✓ IPv6 forwarding rules added${NC}"
else
    echo -e "${GREEN}✓ IPv6 forwarding rules already exist${NC}"
fi

# Create a test VM network namespace
echo
echo "2. Creating test dual-stack VM network..."

VM_ID="$(date +%s)"
NS_NAME="vm-test-$VM_ID"
TAP_NAME="tap${VM_ID: -8}"
VETH_HOST="veth${VM_ID: -8}h"
VETH_NS="veth${VM_ID: -8}n"
VM_IPV4="10.100.1.2"
VM_IPV6="fd00::1:2"
VM_MAC="02:00:00:00:00:02"

# Create namespace
ip netns add $NS_NAME
echo -e "${GREEN}✓ Created namespace: $NS_NAME${NC}"

# Create veth pair
ip link add $VETH_HOST type veth peer name $VETH_NS
echo -e "${GREEN}✓ Created veth pair: $VETH_HOST <-> $VETH_NS${NC}"

# Move one end to namespace
ip link set $VETH_NS netns $NS_NAME
echo -e "${GREEN}✓ Moved $VETH_NS to namespace${NC}"

# Attach host end to bridge
ip link set $VETH_HOST master br-vms
ip link set $VETH_HOST up
echo -e "${GREEN}✓ Attached $VETH_HOST to bridge${NC}"

# Configure inside namespace
ip netns exec $NS_NAME ip link set lo up
ip netns exec $NS_NAME ip link set $VETH_NS up

# Add IPv4 address (use /16 to match bridge subnet)
ip netns exec $NS_NAME ip addr add $VM_IPV4/16 dev $VETH_NS
echo -e "${GREEN}✓ Added IPv4 address in namespace${NC}"

# Add IPv6 address
ip netns exec $NS_NAME ip addr add $VM_IPV6/64 dev $VETH_NS
echo -e "${GREEN}✓ Added IPv6 address in namespace${NC}"

# Wait for addresses to be configured
sleep 1

# Add default routes
ip netns exec $NS_NAME ip route add default via 10.100.0.1 dev $VETH_NS
echo -e "${GREEN}✓ Added IPv4 default route${NC}"

ip netns exec $NS_NAME ip -6 route add default via fd00::1 dev $VETH_NS
echo -e "${GREEN}✓ Added IPv6 default route${NC}"

# Test connectivity
echo
echo "3. Testing dual-stack connectivity..."

# Test IPv4 to bridge
if ip netns exec $NS_NAME ping -c 1 -W 1 10.100.0.1 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ IPv4: VM can reach bridge gateway${NC}"
else
    echo -e "${RED}✗ IPv4: VM cannot reach bridge gateway${NC}"
fi

# Test IPv6 to bridge
if ip netns exec $NS_NAME ping6 -c 1 -W 1 fd00::1 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ IPv6: VM can reach bridge gateway${NC}"
else
    echo -e "${RED}✗ IPv6: VM cannot reach bridge gateway${NC}"
fi

# Test IPv4 to internet (Google DNS)
if ip netns exec $NS_NAME ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ IPv4: VM can reach internet${NC}"
else
    echo -e "${RED}✗ IPv4: VM cannot reach internet${NC}"
fi

# Test IPv6 to internet (Google DNS)
if ip netns exec $NS_NAME ping6 -c 1 -W 2 2001:4860:4860::8888 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ IPv6: VM can reach internet${NC}"
else
    echo -e "${RED}✗ IPv6: VM cannot reach internet (this is normal if host has no IPv6)${NC}"
fi

# Show network configuration
echo
echo "4. Dual-Stack network configuration:"
echo -e "${BLUE}   Namespace:${NC} $NS_NAME"
echo -e "${BLUE}   VM IPv4:${NC} $VM_IPV4/24"
echo -e "${BLUE}   VM IPv6:${NC} $VM_IPV6/64"
echo -e "${BLUE}   Gateway IPv4:${NC} 10.100.0.1"
echo -e "${BLUE}   Gateway IPv6:${NC} fd00::1"
echo -e "${BLUE}   DNS IPv4:${NC} 8.8.8.8, 8.8.4.4"
echo -e "${BLUE}   DNS IPv6:${NC} 2606:4700:4700::1111, 2606:4700:4700::1001"

# Show IP addresses
echo
echo "5. VM Network Interfaces:"
ip netns exec $NS_NAME ip -4 addr show
ip netns exec $NS_NAME ip -6 addr show

# Test the example dual-stack VM creation
echo
echo "6. Testing dual-stack VM creation via metald API..."

# Create test VM with dual-stack networking
RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_example" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 268435456},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 root=/dev/vda rw"
      },
      "storage": [{
        "id": "rootfs",
        "path": "/opt/vm-assets/rootfs.ext4",
        "is_root_device": true,
        "read_only": false
      }],
      "network": [{
        "id": "eth0",
        "interface_type": "virtio-net",
        "mode": "NETWORK_MODE_DUAL_STACK",
        "ipv4_config": {
          "dhcp": false
        },
        "ipv6_config": {
          "slaac": false,
          "privacy_extensions": false
        },
        "rx_rate_limit": {
          "bandwidth": 1073741824
        },
        "tx_rate_limit": {
          "bandwidth": 1073741824
        }
      }],
      "metadata": {
        "name": "dual-stack-test-vm",
        "network": "dual-stack"
      }
    }
  }')

if echo "$RESPONSE" | grep -q "vm_id"; then
    VM_ID=$(echo "$RESPONSE" | grep -o '"vm_id":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Created dual-stack VM: $VM_ID${NC}"
else
    echo -e "${RED}✗ Failed to create VM: $RESPONSE${NC}"
fi

# Cleanup function
echo
echo "7. To cleanup this test setup, run:"
echo "   ip link del $VETH_HOST 2>/dev/null"
echo "   ip netns del $NS_NAME 2>/dev/null"
echo

echo -e "${GREEN}=== Dual-Stack network setup complete ===${NC}"