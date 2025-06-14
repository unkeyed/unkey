#!/bin/bash
set -e

echo "=== Testing MicroVM Networking Setup ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
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

# Check dependencies
echo "1. Checking dependencies..."
MISSING_DEPS=()

if ! command_exists ip; then
    MISSING_DEPS+=("iproute2")
fi

if ! command_exists iptables; then
    MISSING_DEPS+=("iptables")
fi

if ! command_exists brctl; then
    MISSING_DEPS+=("bridge-utils")
fi

if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
    echo -e "${RED}Missing dependencies: ${MISSING_DEPS[*]}${NC}"
    echo "Install with: apt-get install ${MISSING_DEPS[*]}"
    exit 1
fi

echo -e "${GREEN}✓ All dependencies satisfied${NC}"

# Setup host networking
echo
echo "2. Setting up host networking..."

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward
echo -e "${GREEN}✓ IP forwarding enabled${NC}"

# Create bridge if it doesn't exist
if ! ip link show br-vms >/dev/null 2>&1; then
    ip link add name br-vms type bridge
    ip addr add 10.100.0.1/16 dev br-vms
    ip link set br-vms up
    echo -e "${GREEN}✓ Bridge br-vms created${NC}"
else
    echo -e "${GREEN}✓ Bridge br-vms already exists${NC}"
fi

# Get default interface
DEFAULT_IF=$(ip route | grep default | awk '{print $5}' | head -1)
echo "Default interface: $DEFAULT_IF"

# Setup NAT (check if rules already exist)
if ! iptables -t nat -C POSTROUTING -s 10.100.0.0/16 -o $DEFAULT_IF -j MASQUERADE 2>/dev/null; then
    iptables -t nat -A POSTROUTING -s 10.100.0.0/16 -o $DEFAULT_IF -j MASQUERADE
    echo -e "${GREEN}✓ NAT rule added${NC}"
else
    echo -e "${GREEN}✓ NAT rule already exists${NC}"
fi

if ! iptables -C FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT 2>/dev/null; then
    iptables -A FORWARD -i br-vms -o $DEFAULT_IF -j ACCEPT
    iptables -A FORWARD -i $DEFAULT_IF -o br-vms -m state --state RELATED,ESTABLISHED -j ACCEPT
    echo -e "${GREEN}✓ Forwarding rules added${NC}"
else
    echo -e "${GREEN}✓ Forwarding rules already exist${NC}"
fi

# Create a test VM network namespace
echo
echo "3. Creating test VM network..."

VM_ID="test-vm-$(date +%s)"
NS_NAME="vm-$VM_ID"
TAP_NAME="tap${VM_ID:0:8}"
VETH_HOST="veth${VM_ID:0:8}"
VETH_NS="veth${VM_ID:0:8}-ns"
VM_IP="10.100.1.2"
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
ip netns exec $NS_NAME ip link add $TAP_NAME type tuntap mode tap
ip netns exec $NS_NAME ip link set $TAP_NAME address $VM_MAC
ip netns exec $NS_NAME ip link add br0 type bridge
ip netns exec $NS_NAME ip link set $VETH_NS master br0
ip netns exec $NS_NAME ip link set $TAP_NAME master br0
ip netns exec $NS_NAME ip link set $VETH_NS up
ip netns exec $NS_NAME ip link set $TAP_NAME up
ip netns exec $NS_NAME ip link set br0 up
ip netns exec $NS_NAME ip addr add $VM_IP/24 dev br0
ip netns exec $NS_NAME ip route add default via 10.100.0.1
echo -e "${GREEN}✓ Configured networking inside namespace${NC}"

# Test connectivity
echo
echo "4. Testing connectivity..."

# Test namespace can reach bridge
if ip netns exec $NS_NAME ping -c 1 -W 1 10.100.0.1 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ VM can reach bridge gateway${NC}"
else
    echo -e "${RED}✗ VM cannot reach bridge gateway${NC}"
fi

# Test namespace can reach external (Google DNS)
if ip netns exec $NS_NAME ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ VM can reach internet${NC}"
else
    echo -e "${RED}✗ VM cannot reach internet${NC}"
fi

# Show network configuration
echo
echo "5. Network configuration:"
echo "   Namespace: $NS_NAME"
echo "   TAP device: $TAP_NAME"
echo "   VM IP: $VM_IP"
echo "   VM MAC: $VM_MAC"
echo "   Gateway: 10.100.0.1"
echo

# Show how to use with Firecracker
echo "6. To use with Firecracker:"
echo
echo "   Add this to your VM configuration:"
echo "   {"
echo "     \"network-interfaces\": ["
echo "       {"
echo "         \"iface_id\": \"eth0\","
echo "         \"host_dev_name\": \"$TAP_NAME\","
echo "         \"guest_mac\": \"$VM_MAC\""
echo "       }"
echo "     ]"
echo "   }"
echo
echo "   And start Firecracker with:"
echo "   ip netns exec $NS_NAME firecracker --api-sock /tmp/firecracker.socket"
echo

# Cleanup function
echo "7. To cleanup this test setup, run:"
echo "   ip link del $VETH_HOST 2>/dev/null"
echo "   ip netns del $NS_NAME 2>/dev/null"
echo

echo -e "${GREEN}=== Network setup complete ===${NC}"