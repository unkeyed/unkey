#!/bin/bash
set -e

# Enhanced Tenant and Deployment Management Script
# Uses flat IPv6 allocation - no tenant-based limits

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base IPv6 prefix - we have 65,536 /64 subnets available
BASE_IPV6_PREFIX="fd00"

# ConfigMap to track subnet allocations (persistent across script runs)
SUBNET_TRACKER_CM="subnet-allocations"
SUBNET_TRACKER_NS="kube-system"

# Initialize subnet tracker if it doesn't exist
init_subnet_tracker() {
    if ! kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS &>/dev/null; then
        echo -e "${YELLOW}Initializing subnet tracker...${NC}"
        kubectl create configmap $SUBNET_TRACKER_CM \
            --from-literal=next_subnet_id=2 \
            --from-literal=max_subnet_id=65535 \
            -n $SUBNET_TRACKER_NS
    fi
}

# Allocate next available /64 subnet
allocate_subnet() {
    local deployment_name=$1
    local namespace=$2

    # Get next available subnet ID
    local next_id=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS -o jsonpath='{.data.next_subnet_id}')
    local max_id=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS -o jsonpath='{.data.max_subnet_id}')

    if [ $next_id -gt $max_id ]; then
        echo -e "${RED}ERROR: No more subnets available! Used all 65,536 /64 subnets${NC}"
        exit 1
    fi

    # Generate subnet
    local hex_id=$(printf '%04x' $next_id)
    local subnet="${BASE_IPV6_PREFIX}:${hex_id}::/64"

    # Update tracker
    local new_next_id=$((next_id + 1))
    kubectl patch configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS \
        --type merge \
        -p "{\"data\":{\"next_subnet_id\":\"$new_next_id\",\"deployment_${deployment_name}_${namespace}\":\"$subnet\"}}"

    echo "$subnet"
}

# Function to create a new tenant namespace (no subnet hierarchy)
create_tenant() {
    local tenant_name=$1

    echo -e "${BLUE}Creating tenant: $tenant_name${NC}"
    echo -e "${YELLOW}  Note: Tenants have no subnet limits - deployments allocated from global pool${NC}"

    # Create namespace
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-$tenant_name
  labels:
    tenant: $tenant_name
    platform: "multi-tenant"
    isolation: "strict"
EOF

    # Create default network isolation policy
    cat <<EOF | kubectl apply -f -
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: default-tenant-isolation
  namespace: tenant-$tenant_name
spec:
  description: "Default isolation for tenant $tenant_name"
  endpointSelector: {}
  ingress:
    # Allow intra-namespace by default
    - fromEndpoints:
        - matchLabels:
            io.kubernetes.pod.namespace: tenant-$tenant_name
    # Allow DNS
    - fromEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
  egress:
    # Allow intra-namespace
    - toEndpoints:
        - matchLabels:
            io.kubernetes.pod.namespace: tenant-$tenant_name
    # Allow DNS
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
    # Allow external HTTPS (for image pulls, etc)
    - toCIDRSet:
        - cidr: "0.0.0.0/0"
      toPorts:
        - ports:
            - port: "443"
              protocol: TCP
EOF

    # Create resource quota (optional)
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-quota
  namespace: tenant-$tenant_name
spec:
  hard:
    requests.cpu: "100"
    requests.memory: 200Gi
    limits.cpu: "200"
    limits.memory: 400Gi
    persistentvolumeclaims: "100"
    services.loadbalancers: "10"
EOF

    echo -e "${GREEN}Tenant $tenant_name created successfully${NC}"
}

# Create deployment with automatic subnet allocation
create_deployment() {
    local tenant_name=$1
    local deployment_name=$2
    local image=$3
    local replicas=${4:-3}

    # Initialize tracker if needed
    init_subnet_tracker

    # Allocate subnet for this deployment
    local deployment_subnet=$(allocate_subnet "$deployment_name" "tenant-$tenant_name")

    echo -e "${BLUE}Creating deployment: $deployment_name${NC}"
    echo -e "${YELLOW}  Tenant: $tenant_name${NC}"
    echo -e "${YELLOW}  Allocated Subnet: $deployment_subnet${NC}"
    echo -e "${YELLOW}  Replicas: $replicas${NC}"

    # Create deployment with its own /64 subnet
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $deployment_name
  namespace: tenant-$tenant_name
  annotations:
    cilium.io/ipv6-pool: "$deployment_subnet"
    platform.io/subnet: "$deployment_subnet"
    platform.io/tenant: "$tenant_name"
spec:
  replicas: $replicas
  selector:
    matchLabels:
      app: $deployment_name
      deployment: $deployment_name
      tenant: $tenant_name
  template:
    metadata:
      labels:
        app: $deployment_name
        deployment: $deployment_name
        tenant: $tenant_name
        has-subnet: "true"
      annotations:
        cilium.io/ipv6-pool: "$deployment_subnet"
    spec:
      runtimeClassName: gvisor
      containers:
      - name: app
        image: $image
        ports:
        - containerPort: 8080
        env:
        - name: DEPLOYMENT_NAME
          value: "$deployment_name"
        - name: TENANT_NAME
          value: "$tenant_name"
        - name: DEPLOYMENT_SUBNET
          value: "$deployment_subnet"
        - name: POD_IPS
          valueFrom:
            fieldRef:
              fieldPath: status.podIPs
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 1000m
            memory: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: $deployment_name
  namespace: tenant-$tenant_name
  annotations:
    platform.io/subnet: "$deployment_subnet"
spec:
  selector:
    app: $deployment_name
    deployment: $deployment_name
  ports:
    - port: 8080
      targetPort: 8080
      name: http
EOF

    # Create strict network policy for this deployment
    cat <<EOF | kubectl apply -f -
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: $deployment_name-subnet-isolation
  namespace: tenant-$tenant_name
spec:
  description: "Subnet isolation for $deployment_name"
  endpointSelector:
    matchLabels:
      deployment: $deployment_name
  ingress:
    # Only allow from same deployment (same subnet)
    - fromEndpoints:
        - matchLabels:
            deployment: $deployment_name
    # Allow from gateway/proxy if it exists
    - fromEndpoints:
        - matchLabels:
            role: gateway
  egress:
    # Allow to same deployment
    - toEndpoints:
        - matchLabels:
            deployment: $deployment_name
    # Allow to gateway if needed
    - toEndpoints:
        - matchLabels:
            role: gateway
    # Allow DNS
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
    # Block RFC1918 (private networks) but allow internet
    - toCIDRSet:
        - cidr: "0.0.0.0/0"
          except:
            - "10.0.0.0/8"
            - "172.16.0.0/12"
            - "192.168.0.0/16"
            - "${BASE_IPV6_PREFIX}::/48"  # Block other tenant subnets
EOF

    echo -e "${GREEN}Deployment $deployment_name created with isolated subnet${NC}"
}

# Gateway Architecture Options
create_gateway() {
    local mode=$1  # "central" or "per-tenant" or "per-deployment"
    local tenant_name=$2
    local deployment_name=$3

    case "$mode" in
        central)
            echo -e "${BLUE}Creating central gateway for all tenants${NC}"
            create_central_gateway
            ;;
        per-tenant)
            echo -e "${BLUE}Creating gateway for tenant: $tenant_name${NC}"
            create_tenant_gateway "$tenant_name"
            ;;
        per-deployment)
            echo -e "${BLUE}Creating gateway for deployment: $deployment_name in tenant: $tenant_name${NC}"
            create_deployment_gateway "$tenant_name" "$deployment_name"
            ;;
        *)
            echo -e "${RED}Invalid gateway mode. Use: central, per-tenant, or per-deployment${NC}"
            exit 1
            ;;
    esac
}

# Central gateway (shared by all tenants)
create_central_gateway() {
    # Create gateway namespace
    kubectl create namespace gateway-system --dry-run=client -o yaml | kubectl apply -f -

    # Allocate subnet for gateway
    init_subnet_tracker
    local gateway_subnet=$(allocate_subnet "central-gateway" "gateway-system")

    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: central-gateway
  namespace: gateway-system
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-ip-address-type: "dualstack"
spec:
  type: LoadBalancer
  selector:
    app: central-gateway
  ports:
    - port: 80
      name: http
    - port: 443
      name: https
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: central-gateway
  namespace: gateway-system
  annotations:
    cilium.io/ipv6-pool: "$gateway_subnet"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: central-gateway
  template:
    metadata:
      labels:
        app: central-gateway
        role: gateway
      annotations:
        cilium.io/ipv6-pool: "$gateway_subnet"
    spec:
      containers:
      - name: gateway
        image: envoyproxy/envoy:v1.28-latest
        ports:
        - containerPort: 80
          name: http
        - containerPort: 443
          name: https
        - containerPort: 9901
          name: admin
        env:
        - name: GATEWAY_MODE
          value: "central"
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
---
# Allow all namespaces to reach the central gateway
apiVersion: cilium.io/v2
kind: CiliumClusterwideNetworkPolicy
metadata:
  name: allow-to-central-gateway
spec:
  description: "Allow all pods to reach central gateway"
  endpointSelector: {}
  egress:
    - toEndpoints:
        - matchLabels:
            app: central-gateway
            io.kubernetes.pod.namespace: gateway-system
EOF

    echo -e "${GREEN}Central gateway created with subnet: $gateway_subnet${NC}"
}

# Per-tenant gateway
create_tenant_gateway() {
    local tenant_name=$1

    # Allocate subnet for tenant gateway
    init_subnet_tracker
    local gateway_subnet=$(allocate_subnet "gateway" "tenant-$tenant_name")

    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: tenant-gateway
  namespace: tenant-$tenant_name
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
spec:
  type: LoadBalancer
  selector:
    app: tenant-gateway
    tenant: $tenant_name
  ports:
    - port: 80
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenant-gateway
  namespace: tenant-$tenant_name
  annotations:
    cilium.io/ipv6-pool: "$gateway_subnet"
spec:
  replicas: 2
  selector:
    matchLabels:
      app: tenant-gateway
      tenant: $tenant_name
  template:
    metadata:
      labels:
        app: tenant-gateway
        tenant: $tenant_name
        role: gateway
      annotations:
        cilium.io/ipv6-pool: "$gateway_subnet"
    spec:
      runtimeClassName: gvisor
      containers:
      - name: gateway
        image: nginx:alpine  # Replace with your gateway
        ports:
        - containerPort: 8080
        env:
        - name: TENANT_NAME
          value: "$tenant_name"
        - name: GATEWAY_SUBNET
          value: "$gateway_subnet"
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 1000m
            memory: 512Mi
EOF

    echo -e "${GREEN}Tenant gateway created with subnet: $gateway_subnet${NC}"
}

# Show statistics
show_stats() {
    echo -e "${BLUE}=== Platform Statistics ===${NC}"

    # Get subnet allocation info
    if kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS &>/dev/null; then
        local next_id=$(kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS -o jsonpath='{.data.next_subnet_id}')
        local used=$((next_id - 2))
        local available=$((65534 - used))

        echo -e "${YELLOW}Subnet Allocation:${NC}"
        echo -e "  Used: $used / 65534"
        echo -e "  Available: $available"
        echo -e "  Utilization: $(( (used * 100) / 65534 ))%"
    fi

    echo -e "${YELLOW}Tenants:${NC}"
    local tenant_count=$(kubectl get namespaces -l tenant --no-headers 2>/dev/null | wc -l)
    echo -e "  Total: $tenant_count"

    echo -e "${YELLOW}Deployments by Tenant:${NC}"
    for ns in $(kubectl get namespaces -l tenant -o jsonpath='{.items[*].metadata.name}'); do
        local deploy_count=$(kubectl get deployments -n $ns --no-headers 2>/dev/null | wc -l)
        local tenant_name=${ns#tenant-}
        echo -e "  $tenant_name: $deploy_count deployments"
    done

    echo -e "${YELLOW}Total Deployments:${NC}"
    local total_deployments=$(kubectl get deployments -A -l has-subnet=true --no-headers 2>/dev/null | wc -l)
    echo -e "  $total_deployments"
}

# List all allocated subnets
list_subnets() {
    echo -e "${BLUE}=== Allocated Subnets ===${NC}"

    kubectl get configmap $SUBNET_TRACKER_CM -n $SUBNET_TRACKER_NS -o json | \
        jq -r '.data | to_entries[] | select(.key | startswith("deployment_")) | "\(.key | split("_")[1]): \(.value)"' | \
        column -t -s ':'
}

# Main script
case "$1" in
    init)
        init_subnet_tracker
        echo -e "${GREEN}Subnet tracker initialized${NC}"
        ;;

    create-tenant)
        if [ -z "$2" ]; then
            echo "Usage: $0 create-tenant <tenant-name>"
            exit 1
        fi
        create_tenant "$2"
        ;;

    create-deployment)
        if [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ]; then
            echo "Usage: $0 create-deployment <tenant-name> <deployment-name> <image> [replicas]"
            exit 1
        fi
        create_deployment "$2" "$3" "$4" "$5"
        ;;

    create-gateway)
        if [ -z "$2" ]; then
            echo "Usage: $0 create-gateway <mode> [tenant-name] [deployment-name]"
            echo "  Modes: central, per-tenant, per-deployment"
            exit 1
        fi
        create_gateway "$2" "$3" "$4"
        ;;

    stats)
        show_stats
        ;;

    list-subnets)
        list_subnets
        ;;

    *)
        echo "Enhanced Multi-tenant Kubernetes Platform Manager"
        echo ""
        echo "Usage: $0 {command} [arguments]"
        echo ""
        echo "Commands:"
        echo "  init                                    Initialize subnet tracker"
        echo "  create-tenant <name>                    Create tenant namespace"
        echo "  create-deployment <tenant> <name>       Create deployment with auto-allocated subnet"
        echo "                    <image> [replicas]"
        echo "  create-gateway <mode> [args...]         Create gateway (modes: central/per-tenant/per-deployment)"
        echo "  stats                                   Show platform statistics"
        echo "  list-subnets                           List all allocated subnets"
        echo ""
        echo "Gateway Modes:"
        echo "  central                                 One gateway for entire platform"
        echo "  per-tenant <tenant>                    One gateway per tenant"
        echo "  per-deployment <tenant> <deployment>   One gateway per deployment"
        echo ""
        echo "Examples:"
        echo "  $0 init"
        echo "  $0 create-tenant acme-corp"
        echo "  $0 create-deployment acme-corp web-app nginx:alpine 3"
        echo "  $0 create-gateway central"
        echo "  $0 create-gateway per-tenant acme-corp"
        echo "  $0 stats"
        exit 1
        ;;
esac