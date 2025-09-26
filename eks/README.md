# Multi-Tenant Kubernetes Platform on EKS

A production-ready multi-tenant Kubernetes platform with strong network isolation, IPv6 subnet allocation, and enhanced security using Cilium CNI and gVisor.

## Features

- **AWS EKS with Managed Node Groups**: Reliable, auto-scaling Kubernetes infrastructure
- **Dual-Stack Networking**: IPv4 for services, IPv6 for pod networking
- **Per-Deployment Subnet Isolation**: Each deployment gets its own `/96` IPv6 subnet (4.3 billion IPs)
- **Cilium CNI**: eBPF-based networking with WireGuard encryption
- **gVisor Runtime**: Sandboxed container execution for enhanced security
- **Subnet Recycling**: Automatic subnet reuse when deployments are deleted
- **Multi-Tenant Isolation**: Complete network isolation between tenants and deployments

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      EKS Cluster                         │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Tenant: acme-corp                    │   │
│  │                                                   │   │
│  │  ┌─────────────────────┐  ┌──────────────────┐   │   │
│  │  │  Deployment: web-app │  │ Deployment: api  │   │   │
│  │  │  Subnet: fd00:0001:: │  │ Subnet: fd00:0002│   │   │
│  │  │       /96            │  │      ::/96       │   │   │
│  │  │  ┌────┐ ┌────┐      │  │  ┌────┐         │   │   │
│  │  │  │Pod1│ │Pod2│      │  │  │Pod3│         │   │   │
│  │  │  └────┘ └────┘      │  │  └────┘         │   │   │
│  │  └─────────────────────┘  └──────────────────┘   │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Tenant: big-corp                     │   │
│  │                                                   │   │
│  │  ┌─────────────────────┐                         │   │
│  │  │  Deployment: db     │                         │   │
│  │  │  Subnet: fd00:0003::│                         │   │
│  │  │       /96           │                         │   │
│  │  └─────────────────────┘                         │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Capacity

- **Maximum Concurrent Deployments**: 281,474,976,710,656 (281 trillion) with `/96` subnets
- **IPs per Deployment**: 4,294,967,296 (4.3 billion)
- **Tenants**: Unlimited (tenants are logical namespaces)
- **Subnet Recycling**: Yes, deleted deployments release their subnets for reuse

## Prerequisites

- AWS CLI configured with appropriate credentials
- kubectl installed
- eksctl installed
- jq installed (for subnet management scripts)
- AWS account with permissions to create EKS clusters

## Installation

### 1. Clone and Setup

```bash
cd /Users/florianeikel/Developer/unkey/eks
chmod +x *.sh
```

### 2. Configure Environment

Edit `setup.sh` to set your configuration:

```bash
AWS_PROFILE=${AWS_PROFILE:-sandbox}
PRIMARY_REGION=${PRIMARY_REGION:-us-east-1}
PRIMARY_CLUSTER_NAME=${PRIMARY_CLUSTER_NAME:-my-cluster}
```

### 3. Create EKS Cluster

```bash
./setup.sh
```

This will:
- Create an EKS cluster with managed node groups
- Install Cilium CNI with dual-stack support
- Configure gVisor runtime for enhanced security
- Set up IPv6 subnet allocation with `/96` per deployment

## Usage

### Initialize Subnet Manager

```bash
./subnet-manager.sh init
```

### Create a Tenant

Tenants are logical namespaces with resource quotas and network isolation:

```bash
# Using tenant-manager.sh (for /64 subnets)
./tenant-manager.sh create-tenant acme-corp

# Or using subnet-manager.sh (for /96 subnets - recommended)
# Tenants are created automatically when you create deployments
```

### Deploy Applications

Each deployment automatically gets its own IPv6 subnet:

```bash
# Create deployment with automatic subnet allocation
./subnet-manager.sh create <tenant> <deployment-name> <image> [replicas]

# Examples
./subnet-manager.sh create acme-corp web-frontend nginx:alpine 3
./subnet-manager.sh create acme-corp api-backend node:18-alpine 2
./subnet-manager.sh create big-corp database postgres:14 1
```

### Manage Deployments

```bash
# View subnet allocations
./subnet-manager.sh stats
./subnet-manager.sh list

# Delete deployment (subnet is recycled)
./subnet-manager.sh delete acme-corp web-frontend

# The subnet is now available for the next deployment
```

### Deploy Gateways

Three gateway patterns are supported:

```bash
# Option 1: Central gateway for all tenants
./tenant-manager.sh create-gateway central

# Option 2: Per-tenant gateway
./tenant-manager.sh create-gateway per-tenant acme-corp

# Option 3: Per-deployment gateway
./tenant-manager.sh create-gateway per-deployment acme-corp web-frontend
```

## Testing

### 1. Verify Cluster Components

```bash
# Check nodes
kubectl get nodes

# Verify Cilium is running
kubectl -n kube-system get pods -l k8s-app=cilium
kubectl exec -n kube-system -t ds/cilium -- cilium status

# Check gVisor
kubectl get runtimeclass gvisor
kubectl get ds -n kube-system gvisor-install
```

### 2. Test Deployments

```bash
# Create test deployments
./subnet-manager.sh init
./subnet-manager.sh create test-tenant app1 nginx:alpine 2
./subnet-manager.sh create test-tenant app2 nginx:alpine 2

# Check pods have IPv6 addresses
kubectl get pods -n tenant-test-tenant -o wide

# View allocated subnets
./subnet-manager.sh stats
```

### 3. Verify Network Isolation

```bash
# Get pod IPs
kubectl get pods -n tenant-test-tenant -o custom-columns=NAME:.metadata.name,IPv4:.status.podIP,IPv6:.status.podIPs[1].ip

# Test: Pods in SAME deployment can communicate
kubectl exec -n tenant-test-tenant deploy/app1 -- sh -c "
  apk add curl
  curl -v http://[<ANOTHER_POD_IN_APP1_IPV6>]:80  # Should work
"

# Test: Pods in DIFFERENT deployments are isolated
kubectl exec -n tenant-test-tenant deploy/app1 -- sh -c "
  timeout 5 curl -v http://[<POD_IN_APP2_IPV6>]:80 || echo 'Blocked as expected'
"
```

### 4. Test Subnet Recycling

```bash
# Delete a deployment
./subnet-manager.sh delete test-tenant app1

# Check subnet is recycled
./subnet-manager.sh stats  # Shows recycled subnets

# Create new deployment (reuses recycled subnet)
./subnet-manager.sh create test-tenant app3 nginx:alpine 1

# Verify subnet was reused
./subnet-manager.sh list
```

### 5. Debug Networking

```bash
# Deploy debug pod with network tools
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: netshoot
  namespace: tenant-test-tenant
spec:
  runtimeClassName: gvisor
  containers:
  - name: netshoot
    image: nicolaka/netshoot
    command: ["sleep", "3600"]
EOF

# Use debug pod
kubectl exec -it -n tenant-test-tenant netshoot -- /bin/bash

# Inside pod:
ip addr show         # View IPv6 addresses
ip -6 route         # View IPv6 routes
nslookup kubernetes.default
ping6 -c 2 <pod-ipv6>  # Test IPv6 connectivity
```

### 6. Monitor with Hubble

```bash
# View network flows
kubectl exec -n kube-system -t ds/cilium -- hubble observe --namespace tenant-test-tenant

# Check Hubble UI
kubectl -n kube-system get svc hubble-ui
kubectl port-forward -n kube-system svc/hubble-ui 12000:80
# Open http://localhost:12000
```

## Quick Test Script

Save as `test-all.sh`:

```bash
#!/bin/bash
set -e

echo "=== Cluster Status ==="
kubectl get nodes

echo -e "\n=== Cilium Status ==="
kubectl -n kube-system get pods -l k8s-app=cilium

echo -e "\n=== gVisor Status ==="
kubectl get runtimeclass gvisor

echo -e "\n=== Creating Test Deployments ==="
./subnet-manager.sh init
./subnet-manager.sh create test-tenant frontend nginx:alpine 2
./subnet-manager.sh create test-tenant backend nginx:alpine 2
./subnet-manager.sh create test-tenant-2 database nginx:alpine 1

echo -e "\n=== Subnet Allocation ==="
./subnet-manager.sh stats

echo -e "\n=== Pod Information ==="
kubectl get pods -A -o wide | grep tenant

echo -e "\n=== Network Policies ==="
kubectl get ciliumnetworkpolicies -A

echo -e "\n=== Testing Subnet Recycling ==="
./subnet-manager.sh delete test-tenant frontend
./subnet-manager.sh stats
./subnet-manager.sh create test-tenant new-app nginx:alpine 1
./subnet-manager.sh list

echo -e "\n=== All tests completed successfully! ==="
```

## Network Policy Examples

### Allow Gateway Access

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-gateway
  namespace: tenant-acme-corp
spec:
  endpointSelector:
    matchLabels:
      deployment: web-frontend
  ingress:
    - fromEndpoints:
        - matchLabels:
            role: gateway
```

### Cross-Deployment Communication

```yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-api-to-db
  namespace: tenant-acme-corp
spec:
  endpointSelector:
    matchLabels:
      deployment: database
  ingress:
    - fromEndpoints:
        - matchLabels:
            deployment: api-backend
```

## Troubleshooting

### Cilium Issues

```bash
# Check Cilium status
kubectl exec -n kube-system -t ds/cilium -- cilium status

# View Cilium logs
kubectl logs -n kube-system -l k8s-app=cilium --tail=100

# Restart Cilium if needed
kubectl rollout restart -n kube-system ds/cilium
```

### Subnet Allocation Issues

```bash
# Check ConfigMap
kubectl get configmap -n kube-system ipv6-subnet-allocator -o yaml

# Reset subnet allocation (WARNING: affects all deployments)
kubectl delete configmap -n kube-system ipv6-subnet-allocator
./subnet-manager.sh init
```

### Pod Connectivity Issues

```bash
# Check network policies
kubectl describe ciliumnetworkpolicy -n <namespace> <policy-name>

# Verify pod has IPv6
kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.status.podIPs}'

# Test DNS
kubectl exec -n <namespace> <pod> -- nslookup kubernetes.default
```

## Security Considerations

1. **gVisor Runtime**: All deployments use gVisor by default for sandboxed execution
2. **Network Encryption**: Cilium uses WireGuard for pod-to-pod encryption
3. **Subnet Isolation**: Each deployment is isolated in its own /96 subnet
4. **Network Policies**: Strict ingress/egress rules per deployment
5. **Resource Quotas**: Per-tenant resource limits prevent noisy neighbors

## Scripts Overview

- **setup.sh**: Creates EKS cluster with Cilium and gVisor
- **subnet-manager.sh**: Manages /96 IPv6 subnet allocation with recycling
- **tenant-manager.sh**: Legacy script for /64 subnet allocation
- **tenant-manager-v2.sh**: Enhanced tenant management with flexible allocation

## Clean Up

```bash
# Delete test resources
kubectl delete namespace tenant-test-tenant
kubectl delete namespace tenant-test-tenant-2

# Delete entire cluster (WARNING: destroys everything)
eksctl delete cluster --name $PRIMARY_CLUSTER_NAME --region $PRIMARY_REGION
```

## Additional Resources

- [Cilium Documentation](https://docs.cilium.io/)
- [gVisor Documentation](https://gvisor.dev/)
- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [IPv6 Addressing](https://www.rfc-editor.org/rfc/rfc4291.html)

## License

MIT

## Support

For issues or questions, please open an issue in the repository.