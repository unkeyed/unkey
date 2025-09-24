#!/bin/bash
set -e

# Subnet Manager for Multi-tenant Platform
# Manages /96 IPv6 subnet allocation with recycling

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base configuration
BASE_IPV6_PREFIX="fd00::"
BASE_PREFIX_LENGTH=48
DEPLOYMENT_PREFIX_LENGTH=64

# ConfigMap for subnet tracking
SUBNET_TRACKER_CM="ipv6-subnet-allocator"
SUBNET_TRACKER_NS="kube-system"

# Initialize subnet allocator
init_allocator() {
    echo -e "${BLUE}Initializing IPv6 subnet allocator...${NC}"

    # Create configmap if it doesn't exist
    if ! kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS &>/dev/null; then
        kubectl create configmap $SUBNET_TRACKER_CM \
            --from-literal=base_prefix="${BASE_IPV6_PREFIX}/${BASE_PREFIX_LENGTH}" \
            --from-literal=deployment_prefix_length="${DEPLOYMENT_PREFIX_LENGTH}" \
            --from-literal=allocated_subnets="{}" \
            --from-literal=free_subnets="[]" \
            --from-literal=next_subnet_id="1" \
            -n $SUBNET_TRACKER_NS

        echo -e "${GREEN}Subnet allocator initialized${NC}"
        echo -e "${YELLOW}Available /96 subnets: 281,474,976,710,656 (281 trillion)${NC}"
    else
        echo -e "${YELLOW}Subnet allocator already initialized${NC}"
    fi
}

# Allocate a /96 subnet for a deployment
allocate_subnet() {
    local tenant_name=$1
    local deployment_name=$2
    local key="${tenant_name}/${deployment_name}"

    # Check if already allocated
    local existing=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.allocated_subnets}" | \
        jq -r --arg key "$key" '.[$key] // "none"')

    if [ "$existing" != "none" ]; then
        echo -e "${YELLOW}Subnet already allocated: $existing${NC}"
        echo "$existing"
        return
    fi

    # Check for recycled subnets first
    local free_subnets=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.free_subnets}")

    local subnet=""
    if [ "$(echo $free_subnets | jq '. | length')" -gt 0 ]; then
        # Use recycled subnet
        subnet=$(echo $free_subnets | jq -r '.[0]')
        echo -e "${GREEN}Reusing recycled subnet: $subnet${NC}"

        # Remove from free list
        free_subnets=$(echo $free_subnets | jq 'del(.[0])')
    else
        # Allocate new subnet
        local next_id=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
            -o jsonpath="{.data.next_subnet_id}")

        # Generate /64 subnet
        # With 16 bits available (64-48), we can represent the ID in hex
        local hex_id=$(printf '%04x' $next_id)  # 4 hex chars = 16 bits

        # Split hex_id into groups for IPv6 notation
        # fd00:XXXX::/64
        subnet="fd00:${hex_id}::/64"

        # Increment next_id
        next_id=$((next_id + 1))
        kubectl patch configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
            --type merge \
            -p "{\"data\":{\"next_subnet_id\":\"$next_id\"}}"
    fi

    # Update allocated subnets
    local allocated=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.allocated_subnets}")

    allocated=$(echo $allocated | jq --arg key "$key" --arg subnet "$subnet" \
        '.[$key] = {subnet: $subnet, allocated_at: now | todate}')

    kubectl patch configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        --type merge \
        -p "{\"data\":{\"allocated_subnets\":$(echo $allocated | jq -Rs '.'),\"free_subnets\":$(echo $free_subnets | jq -Rs '.')}}"

    echo -e "${GREEN}Allocated subnet: $subnet${NC}"
    echo "$subnet"
}

# Free a subnet (for recycling)
free_subnet() {
    local tenant_name=$1
    local deployment_name=$2
    local key="${tenant_name}/${deployment_name}"

    # Get current allocation
    local allocated=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.allocated_subnets}")

    local subnet=$(echo $allocated | jq -r --arg key "$key" '.[$key].subnet // "none"')

    if [ "$subnet" == "none" ]; then
        echo -e "${YELLOW}No subnet allocated for $key${NC}"
        return
    fi

    # Remove from allocated
    allocated=$(echo $allocated | jq --arg key "$key" 'del(.[$key])')

    # Add to free list
    local free_subnets=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.free_subnets}")

    free_subnets=$(echo $free_subnets | jq --arg subnet "$subnet" '. += [$subnet]')

    kubectl patch configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        --type merge \
        -p "{\"data\":{\"allocated_subnets\":$(echo $allocated | jq -Rs '.'),\"free_subnets\":$(echo $free_subnets | jq -Rs '.')}}"

    echo -e "${GREEN}Freed subnet: $subnet (available for reuse)${NC}"
}

# Show allocation statistics
show_stats() {
    echo -e "${BLUE}=== Subnet Allocation Statistics ===${NC}"

    local allocated=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.allocated_subnets}" | jq -r)

    local free_subnets=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.free_subnets}" | jq -r)

    local next_id=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.next_subnet_id}")

    local allocated_count=$(echo $allocated | jq '. | length')
    local recycled_count=$(echo $free_subnets | jq '. | length')
    local total_used=$((next_id - 1))

    echo -e "${YELLOW}Currently Allocated:${NC} $allocated_count deployments"
    echo -e "${YELLOW}Recycled (available):${NC} $recycled_count subnets"
    echo -e "${YELLOW}Total Used (lifetime):${NC} $total_used subnets"
    echo -e "${YELLOW}Available:${NC} ~281 trillion subnets"

    if [ $allocated_count -gt 0 ]; then
        echo ""
        echo -e "${BLUE}Active Allocations:${NC}"
        echo "$allocated" | jq -r 'to_entries[] | "  \(.key): \(.value.subnet) (allocated: \(.value.allocated_at))"'
    fi

    if [ $recycled_count -gt 0 ]; then
        echo ""
        echo -e "${BLUE}Recycled Subnets (ready for reuse):${NC}"
        echo "$free_subnets" | jq -r '.[] | "  \(.)"'
    fi
}

# List all allocated subnets
list_allocated() {
    echo -e "${BLUE}=== Allocated Subnets ===${NC}"

    kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        -o jsonpath="{.data.allocated_subnets}" | \
        jq -r 'to_entries[] | "\(.key):\t\(.value.subnet)\t(Since: \(.value.allocated_at))"' | \
        column -t -s $'\t'
}

# Create deployment with subnet
create_deployment() {
    local tenant_name=$1
    local deployment_name=$2
    local image=$3
    local replicas=${4:-3}

    # Allocate subnet
    local subnet=$(allocate_subnet "$tenant_name" "$deployment_name" | tail -1)

    echo -e "${BLUE}Creating deployment with /64 subnet${NC}"
    echo -e "  Tenant: $tenant_name"
    echo -e "  Deployment: $deployment_name"
    echo -e "  Subnet: $subnet (18.4 quintillion IPs)"
    echo -e "  Image: $image"

    # Create namespace if needed
    kubectl create namespace "tenant-$tenant_name" --dry-run=client -o yaml | kubectl apply -f -

    # Create deployment
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $deployment_name
  namespace: tenant-$tenant_name
  annotations:
    platform.io/ipv6-subnet: "$subnet"
    cilium.io/ipv6-pool: "$subnet"
  labels:
    tenant: $tenant_name
    deployment: $deployment_name
spec:
  replicas: $replicas
  selector:
    matchLabels:
      tenant: $tenant_name
      deployment: $deployment_name
  template:
    metadata:
      labels:
        tenant: $tenant_name
        deployment: $deployment_name
      annotations:
        # Use Cilium IP pool annotation for subnet assignment
        cilium.io/ipv6-pool: "$subnet"
    spec:
      containers:
      - name: app
        image: $image
        env:
        - name: IPV6_SUBNET
          value: "$subnet"
        - name: TENANT
          value: "$tenant_name"
        - name: DEPLOYMENT
          value: "$deployment_name"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 1000m
            memory: 1Gi
---
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: ${deployment_name}-subnet-isolation
  namespace: tenant-$tenant_name
spec:
  endpointSelector:
    matchLabels:
      deployment: $deployment_name
  ingress:
    - fromEndpoints:
        - matchLabels:
            deployment: $deployment_name
  egress:
    - toEndpoints:
        - matchLabels:
            deployment: $deployment_name
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
EOF

    echo -e "${GREEN}Deployment created successfully${NC}"
}

# Delete deployment and free subnet
delete_deployment() {
    local tenant_name=$1
    local deployment_name=$2

    echo -e "${BLUE}Deleting deployment and freeing subnet...${NC}"

    # Delete Kubernetes resources
    kubectl delete deployment $deployment_name -n tenant-$tenant_name --ignore-not-found=true
    kubectl delete ciliumnetworkpolicy ${deployment_name}-subnet-isolation -n tenant-$tenant_name --ignore-not-found=true

    # Free the subnet for recycling
    free_subnet "$tenant_name" "$deployment_name"

    echo -e "${GREEN}Deployment deleted and subnet recycled${NC}"
}

# Main
case "$1" in
    init)
        init_allocator
        ;;
    allocate)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 allocate <tenant> <deployment>"
            exit 1
        fi
        allocate_subnet "$2" "$3"
        ;;
    free)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 free <tenant> <deployment>"
            exit 1
        fi
        free_subnet "$2" "$3"
        ;;
    create)
        if [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ]; then
            echo "Usage: $0 create <tenant> <deployment> <image> [replicas]"
            exit 1
        fi
        create_deployment "$2" "$3" "$4" "$5"
        ;;
    delete)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 delete <tenant> <deployment>"
            exit 1
        fi
        delete_deployment "$2" "$3"
        ;;
    list)
        list_allocated
        ;;
    stats)
        show_stats
        ;;
    *)
        echo "IPv6 Subnet Manager for Multi-tenant Platform"
        echo ""
        echo "Configuration:"
        echo "  Base prefix: fd00::/48"
        echo "  Deployment subnets: /64 (18.4 quintillion IPs each)"
        echo "  Max concurrent deployments: 16,777,216"
        echo ""
        echo "Usage: $0 {command} [arguments]"
        echo ""
        echo "Commands:"
        echo "  init                        Initialize subnet allocator"
        echo "  allocate <tenant> <deploy>  Allocate subnet for deployment"
        echo "  free <tenant> <deploy>      Free subnet (make available for reuse)"
        echo "  create <tenant> <deploy>    Create deployment with auto-allocated subnet"
        echo "         <image> [replicas]"
        echo "  delete <tenant> <deploy>    Delete deployment and recycle subnet"
        echo "  list                        List all allocated subnets"
        echo "  stats                       Show allocation statistics"
        echo ""
        echo "Examples:"
        echo "  $0 init"
        echo "  $0 create acme web nginx:alpine 3"
        echo "  $0 delete acme web  # Subnet becomes available for reuse"
        echo "  $0 stats"
        exit 1
        ;;
esac