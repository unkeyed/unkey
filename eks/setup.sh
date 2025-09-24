#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration - modify these as needed
AWS_PROFILE=${AWS_PROFILE:-sandbox}
PRIMARY_REGION=${PRIMARY_REGION:-us-east-1}
PRIMARY_CLUSTER_NAME=${PRIMARY_CLUSTER_NAME:-flo-testing}
REPO_URL="https://github.com/unkeyed/infra.git"
REPO_BRANCH="main"

echo -e "${BLUE}EKS Cluster Setup${NC}"
echo -e "${YELLOW}Configuration:${NC}"
echo -e "   AWS Profile: $AWS_PROFILE"
echo -e "   Primary Region: $PRIMARY_REGION"
echo -e "   Primary Cluster: $PRIMARY_CLUSTER_NAME"
echo -e "   Repository: $REPO_URL"
echo -e "   Branch: $REPO_BRANCH"
echo ""

# Function to wait for user confirmation
confirm() {
    read -p "Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Aborted by user${NC}"
        exit 1
    fi
}

# Function to wait for deployment to be ready
wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}

    echo -e "${YELLOW}Waiting for $deployment in $namespace to be ready...${NC}"
    if kubectl wait --for=condition=available --timeout=${timeout}s deployment/$deployment -n $namespace; then
        echo -e "${GREEN}$deployment is ready${NC}"
    else
        echo -e "${RED}$deployment failed to become ready within ${timeout}s${NC}"
        exit 1
    fi
}

echo -e "${BLUE}Step 1: Verify Prerequisites${NC}"

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}kubectl is not installed${NC}"
    exit 1
fi

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo -e "${RED}AWS CLI is not installed${NC}"
    exit 1
fi

# Check if eksctl is installed
if ! command -v eksctl &> /dev/null; then
    echo -e "${RED}eksctl is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}All prerequisites are installed${NC}
${YELLOW}Note: Helm will be installed automatically if not present${NC}"

# Check AWS credentials
echo -e "${BLUE}Checking AWS credentials...${NC}"
if ! aws sts get-caller-identity --profile $AWS_PROFILE &> /dev/null; then
    echo -e "${RED}AWS credentials not configured for profile: $AWS_PROFILE${NC}"
    exit 1
fi

echo -e "${GREEN}AWS credentials are valid${NC}"

# Check if primary cluster exists
echo -e "${BLUE}Step 2: Check Primary Cluster${NC}"
if aws eks describe-cluster --name $PRIMARY_CLUSTER_NAME --region $PRIMARY_REGION --profile $AWS_PROFILE &> /dev/null; then
    echo -e "${GREEN}Primary cluster $PRIMARY_CLUSTER_NAME exists${NC}"

    # Update kubeconfig
    echo -e "${YELLOW}Updating kubeconfig...${NC}"
    aws eks update-kubeconfig --region $PRIMARY_REGION --name $PRIMARY_CLUSTER_NAME --profile $AWS_PROFILE

    # Switch to primary cluster context
    kubectl config use-context arn:aws:eks:$PRIMARY_REGION:$(aws sts get-caller-identity --profile $AWS_PROFILE --query Account --output text):cluster/$PRIMARY_CLUSTER_NAME

    echo -e "${YELLOW}Primary cluster exists. Skip creation? (y/N): ${NC}"
    read -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Skipping primary cluster creation${NC}"
    else
        echo -e "${RED}Cannot recreate existing cluster. Delete it first or choose a different name.${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}Primary cluster $PRIMARY_CLUSTER_NAME does not exist${NC}"
    echo -e "${BLUE}Creating primary cluster...${NC}"

    # Create cluster config for primary region
    cat > /tmp/cluster-config.yaml << EOF
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
    name: $PRIMARY_CLUSTER_NAME
    region: $PRIMARY_REGION
    version: "1.33"

# IRSA Service Accounts for External Secrets
iam:
    withOIDC: true

# VPC configuration with dual-stack support
vpc:
    nat:
        gateway: HighlyAvailable
    clusterEndpoints:
        publicAccess: true
        privateAccess: true

# Kubernetes networking configuration for dual-stack
kubernetesNetworkConfig:
    ipFamily: IPv4
    serviceIPv4CIDR: 10.100.0.0/16

# AWS managed node groups
managedNodeGroups:
    - name: primary-nodes
      instanceType: m5.large
      minSize: 2
      maxSize: 10
      desiredCapacity: 3
      volumeSize: 50
      volumeType: gp3
      amiFamily: AmazonLinux2
      # Enable IPv6 on nodes
      privateNetworking: true
      iam:
          attachPolicyARNs:
              - arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy
              - arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy
              - arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly
      # Enable container runtime security for gVisor
      securityGroups:
          attachIDs: []
      ssh:
          allow: false
      # Taint nodes so that application pods are not scheduled until Cilium is deployed
      taints:
          - key: "node.cilium.io/agent-not-ready"
            value: "true"
            effect: "NoExecute"
      tags:
          Environment: production
          ManagedBy: eksctl

# Add-ons configuration (we'll install Cilium manually to replace AWS VPC CNI)
addons:
    - name: kube-proxy
      version: latest
    - name: coredns
      version: latest
EOF

    eksctl create cluster -f /tmp/cluster-config.yaml --profile=$AWS_PROFILE
    rm /tmp/cluster-config.yaml

    echo -e "${GREEN}Primary cluster created successfully${NC}"
fi

# Install Cilium CNI (always run this, whether cluster was created or already existed)
echo -e "${BLUE}Step 3: Installing Cilium CNI${NC}"

# Check if Cilium is already installed
if kubectl get daemonset cilium -n kube-system &>/dev/null; then
    echo -e "${YELLOW}Cilium is already installed. Skipping installation.${NC}"
    echo -e "${YELLOW}To reinstall Cilium, run: helm uninstall cilium -n kube-system${NC}"
else
    # Check if Helm is installed
    if ! command -v helm &> /dev/null; then
        echo -e "${RED}Helm is not installed. Installing Helm...${NC}"
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    fi

    # Add Cilium Helm repository
    helm repo add cilium https://helm.cilium.io/
    helm repo update

    # Patch AWS VPC CNI to prevent conflicts with Cilium (from EKS docs)
    echo -e "${YELLOW}Patching AWS VPC CNI to prevent conflicts...${NC}"
    kubectl -n kube-system patch daemonset aws-node --type='strategic' \
        -p='{"spec":{"template":{"spec":{"nodeSelector":{"io.cilium/aws-node-enabled":"true"}}}}}' || true

    # Get correct API server endpoint
    API_SERVER_URL=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
    API_SERVER_HOST=$(echo $API_SERVER_URL | sed 's|https://||' | cut -d: -f1)

    # Check if port is explicitly specified, default to 443 if not
    if echo $API_SERVER_URL | grep -q ':[0-9]*$'; then
        API_SERVER_PORT=$(echo $API_SERVER_URL | sed 's|https://||' | cut -d: -f2)
    else
        API_SERVER_PORT=443
    fi

    echo -e "${YELLOW}Using API server: $API_SERVER_HOST:$API_SERVER_PORT${NC}"

    # Install Cilium with EKS-compatible configuration and IPv6 support
    helm install cilium cilium/cilium --version 1.16.4 \
        --namespace kube-system \
        --set cluster.name=$PRIMARY_CLUSTER_NAME \
        --set cluster.id=1 \
        --set k8sServiceHost=$API_SERVER_HOST \
        --set k8sServicePort=$API_SERVER_PORT \
        --set ipam.mode=cluster-pool \
        --set ipam.operator.clusterPoolIPv4PodCIDRList="10.0.0.0/8" \
        --set ipam.operator.clusterPoolIPv6PodCIDRList="fd00::/48" \
        --set ipam.operator.clusterPoolIPv4MaskSize=24 \
        --set ipam.operator.clusterPoolIPv6MaskSize=64 \
        --set enableIPv4Masquerade=true \
        --set enableIPv6Masquerade=true \
        --set ipv6.enabled=true \
        --set hubble.relay.enabled=true \
        --set hubble.ui.enabled=true

    # Wait for Cilium to be ready
    echo -e "${YELLOW}Waiting for Cilium to be ready...${NC}"
    kubectl rollout status -n kube-system daemonset/cilium --timeout=300s

    # Restart CoreDNS to use Cilium networking
    kubectl rollout restart -n kube-system deployment/coredns

    echo -e "${GREEN}Cilium installed successfully${NC}"
fi

# Install gVisor (always run this, whether cluster was created or already existed)
echo -e "${BLUE}Step 4: Installing gVisor${NC}"

# Check if gVisor is already installed
if kubectl get runtimeclass gvisor &>/dev/null; then
    echo -e "${YELLOW}gVisor is already installed. Skipping installation.${NC}"
    echo -e "${YELLOW}To reinstall gVisor, delete the runtime class and daemonset first.${NC}"
else

    # Install gVisor using official method from https://gvisor.dev/docs/user_guide/containerd/quick_start/
    echo -e "${YELLOW}Installing gVisor using official method...${NC}"

    # Apply the validated gVisor installation YAML
    kubectl apply -f gvisor-install.yaml

    # Wait for gVisor installation to complete (with shorter timeout)
    echo -e "${YELLOW}Waiting for gVisor installation...${NC}"
    if kubectl rollout status daemonset/install-gvisor -n kube-system --timeout=120s; then
        echo -e "${GREEN}gVisor installation completed${NC}"
    else
        echo -e "${YELLOW}gVisor installation taking longer than expected, continuing...${NC}"
        echo -e "${YELLOW}You can check status later with: kubectl get pods -n kube-system -l app=install-gvisor${NC}"
    fi

    echo -e "${GREEN}gVisor installed successfully${NC}"

    # Test gVisor installation
    echo -e "${BLUE}Testing gVisor installation...${NC}"
    cat > /tmp/gvisor-test.yaml << 'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: gvisor-test
spec:
  runtimeClassName: gvisor
  containers:
  - name: test
    image: nginx:alpine
    ports:
    - containerPort: 80
  restartPolicy: Never
EOF

    kubectl apply -f /tmp/gvisor-test.yaml

    # Wait for test pod and then clean up
    echo -e "${YELLOW}Testing gVisor runtime...${NC}"
    if kubectl wait --for=condition=Ready pod/gvisor-test --timeout=60s; then
        echo -e "${GREEN}gVisor test successful${NC}"
        kubectl delete -f /tmp/gvisor-test.yaml
    else
        echo -e "${YELLOW}gVisor test pod did not become ready, but installation may still be successful${NC}"
        kubectl delete -f /tmp/gvisor-test.yaml || true
    fi

    rm /tmp/gvisor-test.yaml
fi

echo -e "${GREEN}=== EKS Cluster Setup Complete! ===${NC}"
echo -e "${BLUE}Next steps:${NC}"
echo -e "  1. Initialize subnet manager: ${YELLOW}./subnet-manager.sh init${NC}"
echo -e "  2. Create a test deployment: ${YELLOW}./subnet-manager.sh create test-tenant web-app nginx:alpine${NC}"
echo -e "  3. Check cluster status: ${YELLOW}kubectl get pods -A${NC}"
echo -e "  4. Monitor Cilium: ${YELLOW}kubectl exec -n kube-system -t ds/cilium -- cilium status${NC}"


# serviceAccounts:
    #     - metadata:
    #           name: gw
    #           namespace: gw
    #           labels:
    #               app.kubernetes.io/name: gw
    #       roleName: eks-secrets-manager-gw-$PRIMARY_REGION
    #       roleOnly: false
    #       attachPolicyARNs:
    #           - arn:aws:iam::aws:policy/SecretsManagerReadWrite
    #       tags:
    #           Service: gw
    #           Environment: production
    #           Region: $PRIMARY_REGION
    #     - metadata:
    #           name: ctrl
    #           namespace: ctrl
    #           labels:
    #               app.kubernetes.io/name: ctrl
    #       roleName: eks-secrets-manager-ctrl-$PRIMARY_REGION
    #       roleOnly: false
    #       attachPolicyARNs:
    #           - arn:aws:iam::aws:policy/SecretsManagerReadWrite
    #       tags:
    #           Service: ctrl
    #           Environment: production
    #           Region: $PRIMARY_REGION
