# IPv6 Deployment Guide - Production Security Hardened

## Overview

This guide provides comprehensive, security-focused deployment instructions for IPv6 networking capabilities in metald. It covers prerequisites, hardened configuration, security validation, and migration strategies for production environments supporting global VM deployments.

## Security Requirements

### Production Security Baseline
- **Provider Independent IPv6 Block**: Must have allocated /44 or /48 from RIR
- **IPv6 Security Controls**: RA Guard, source validation, extension header filtering
- **Network Segmentation**: VLAN isolation mandatory for multi-tenant
- **Monitoring & Logging**: Security event logging and SIEM integration
- **Compliance**: SOC 2, PCI DSS, and GDPR privacy controls

## Prerequisites

### System Requirements

**Operating System**:
- Ubuntu 22.04+ or RHEL 9+ (with hardened IPv6 kernel support)
- Linux kernel 6.1+ (for IPv6 security features and performance)
- systemd-networkd with IPv6 security extensions

**Network Infrastructure**:
- **Provider Independent IPv6 allocation** (from ARIN/RIPE/APNIC)
- **BGP multi-homing** with 2+ upstream providers
- **IPv6-capable firewalls** with extension header filtering
- **RPKI validation** capability
- **DDoS protection** with IPv6 support

**Security Dependencies**:
```bash
# Ubuntu/Debian - Security-hardened packages
sudo apt update
sudo apt install -y \
    bridge-utils \
    iproute2 \
    iptables-persistent \
    ipset \
    conntrack \
    nftables \
    radvd \
    ndppd \
    ipv6toolkit \
    fail2ban

# RHEL/CentOS - Security packages  
sudo dnf install -y \
    bridge-utils \
    iproute \
    iptables-services \
    ipset \
    conntrack-tools \
    nftables \
    radvd \
    ipv6toolkit \
    fail2ban

# Install IPv6 security tools
sudo apt install -y sipcalc ndisc6 alive6
```

### Host Network Hardening

**Enable IPv6 with security controls**:
```bash
# IPv6 forwarding with security
sudo tee -a /etc/sysctl.conf << 'EOF'
# IPv6 Security Hardening
net.ipv6.conf.all.forwarding=1
net.ipv6.conf.default.forwarding=1

# Disable IPv6 privacy extensions on bridges (use stable addresses)
net.ipv6.conf.all.use_tempaddr=0
net.ipv6.conf.default.use_tempaddr=0

# Enable IPv6 source validation
net.ipv6.conf.all.accept_ra=0
net.ipv6.conf.default.accept_ra=0

# Prevent IPv6 redirect attacks  
net.ipv6.conf.all.accept_redirects=0
net.ipv6.conf.default.accept_redirects=0

# Neighbor Discovery security
net.ipv6.neigh.default.gc_thresh1=1024
net.ipv6.neigh.default.gc_thresh2=2048  
net.ipv6.neigh.default.gc_thresh3=4096
net.ipv6.neigh.default.gc_interval=30

# ICMPv6 rate limiting  
net.ipv6.icmp.ratelimit=100

# Disable source routing
net.ipv6.conf.all.accept_source_route=0
net.ipv6.conf.default.accept_source_route=0
EOF

sudo sysctl -p
```

**Verify security configuration**:
```bash
# Test IPv6 connectivity with security validation
ping6 -c 3 2001:4860:4860::8888

# Verify IPv6 addresses and routing
ip -6 addr show
ip -6 route show

# Check IPv6 security parameters
sysctl net.ipv6.conf.all.accept_ra
sysctl net.ipv6.conf.all.accept_redirects
sysctl net.ipv6.neigh.default.gc_thresh3
```

## Configuration Setup

### 1. Production IPv6 Configuration

Create `/etc/metald/config.yaml` with production-ready IPv6 settings:

```yaml
# metald IPv6 Production Configuration
ipv6:
  enabled: true
  
  # Production IPv6 pool (replace with your ARIN allocation)
  default_pool:
    id: "production"
    prefix: "2600:BEEF:1000::/44"  # Your allocated prefix
    description: "Production IPv6 pool from ARIN allocation" 
    auto_assign: true
    reserved: 
      - "2600:BEEF:1000::1"       # Primary bridge gateway
      - "2600:BEEF:1000::2"       # Secondary gateway (HA)
      - "2600:BEEF:1000::10"      # DNS resolver
      - "2600:BEEF:1000::11"      # Secondary DNS
      - "2600:BEEF:1000::100"     # Management interface
  
  # Security-hardened bridge configuration
  bridge:
    name: "br-prod"
    ipv6_address: "2600:BEEF:1000::1/64"
    mtu: 1500
    enable_stp: true             # Enable STP for redundancy
    forward_delay: 10            # Fast convergence
    enable_ra_guard: true        # MANDATORY: Enable RA Guard
    enable_mld_snooping: true    # Enable MLD snooping
  
  # Secure DNS configuration  
  dns:
    servers:
      - "2001:4860:4860::8888"   # Google IPv6 DNS primary
      - "2001:4860:4860::8844"   # Google IPv6 DNS secondary  
      - "2606:4700:4700::1111"   # Cloudflare DNS primary
      - "2606:4700:4700::1001"   # Cloudflare DNS secondary
    search_domain: "vm.prod.company.com"
    enable_dnssec: true          # Enable DNSSEC validation
    dns_over_tls: true           # Use DNS-over-TLS
  
  # VLAN security (MANDATORY for production)
  enable_vlans: true
  default_vlan: 100             # No untagged traffic
  management_vlan: 1            # Reserved for management
  
  # Security policies
  security:
    enable_ra_guard: true        # RA Guard globally enabled
    enable_source_guard: true    # Source validation globally
    enable_extension_header_filtering: true
    default_security_profile: "high"
    
    # Rate limiting (global defaults)
    global_nd_rate_limit: 1000   # Neighbor Discovery rate limit
    global_icmpv6_rate_limit: 1000
    global_multicast_rate_limit: 500
    
    # Firewall integration
    enable_firewall_integration: true
    firewall_backend: "nftables" # Use nftables for IPv6
    
    # Logging and monitoring
    enable_security_logging: true
    log_level: "INFO"
    syslog_facility: "local0"

# Production metald settings  
backend: "firecracker"
port: 8080
max_vms: 50000
metrics_collection_interval: "100ms"

# Billing configuration
billing:
  enabled: true
  billaged_endpoint: "https://billing.company.com"
  collection_interval: "100ms"

# Observability
otel:
  enabled: true
  sampling_rate: 0.1           # 10% sampling for production
  endpoint: "https://tracing.company.com"
  
# Security and compliance
security:
  enable_process_isolation: true
  jailer_enabled: true         # Enable Firecracker jailer
  seccomp_enabled: true
```

### 2. Multi-Tenant Enterprise Configuration

For large-scale multi-tenant deployments:

```yaml
ipv6:
  enabled: true
  
  # Multiple pools for tenant isolation
  pools:
    - id: "tier1-customers"
      prefix: "2600:BEEF:1000::/48"
      description: "Tier 1 enterprise customers"
      auto_assign: true
      reserved: ["2600:BEEF:1000::1"]
      security_profile: "high"
      rate_limit: 1000           # Allocations per hour
    
    - id: "tier2-customers"
      prefix: "2600:BEEF:1001::/48" 
      description: "Tier 2 business customers"
      auto_assign: true
      reserved: ["2600:BEEF:1001::1"]
      security_profile: "medium"
      rate_limit: 500
    
    - id: "shared-services"
      prefix: "2600:BEEF:1010::/48"
      description: "Shared infrastructure services"
      auto_assign: false         # Manual assignment only
      security_profile: "high"
  
  # Tenant-specific bridges with security
  bridges:
    - name: "br-tier1"
      ipv6_address: "2600:BEEF:1000::1/64"
      mtu: 1500
      enable_ra_guard: true
      allowed_ra_sources: ["2600:BEEF:1000::1"]
      enable_mld_snooping: true
    
    - name: "br-tier2"
      ipv6_address: "2600:BEEF:1001::1/64"  
      mtu: 1500
      enable_ra_guard: true
      allowed_ra_sources: ["2600:BEEF:1001::1"]
      enable_mld_snooping: true
    
    - name: "br-shared"
      ipv6_address: "2600:BEEF:1010::1/64"
      mtu: 9000                  # Jumbo frames for shared services
      enable_ra_guard: true
      allowed_ra_sources: ["2600:BEEF:1010::1"]
  
  # Advanced security policies
  security_policies:
    - id: "strict-isolation"
      description: "Complete tenant isolation with micro-segmentation"
      rules:
        - "DENY ipv6 between vlans except same-tenant"
        - "ALLOW ipv6 to shared-services vlan 999"
        - "DENY ipv6 extension-headers hop-by-hop,routing,fragment"
        - "RATE-LIMIT icmpv6 100pps per-vm"
        - "RATE-LIMIT nd 50pps per-vm"
    
    - id: "moderate-isolation"  
      description: "Controlled inter-tenant communication"
      rules:
        - "ALLOW ipv6 between vlans with explicit policy"
        - "DENY ipv6 extension-headers hop-by-hop,routing"
        - "RATE-LIMIT icmpv6 200pps per-vm"
        
# High-availability configuration
ha:
  enabled: true
  primary_node: "node1.company.com"
  secondary_nodes: ["node2.company.com", "node3.company.com"]
  heartbeat_interval: "1s"
  failover_timeout: "30s"
```

## Deployment Steps

### Step 1: Infrastructure Security Setup

**1.1 Configure production IPv6 firewall**:
```bash
# Create comprehensive nftables IPv6 rules
sudo tee /etc/nftables/ipv6-security.nft << 'EOF'
#!/usr/sbin/nft -f

# IPv6 Security Rules for metald Production
table ip6 metald_security {
    # Rate limiting for ICMPv6
    set icmpv6_ratelimit {
        type ipv6_addr
        size 65536
        flags dynamic,timeout
        timeout 1m
    }
    
    # Allowed RA sources per bridge
    set allowed_ra_sources {
        type ipv6_addr
        elements = {
            2600:beef:1000::1,
            2600:beef:1001::1, 
            2600:beef:1010::1
        }
    }
    
    # VM address ranges for source validation
    set vm_address_ranges {
        type ipv6_addr
        flags interval
        elements = {
            2600:beef:1000:100::/56,
            2600:beef:1001:100::/56
        }
    }
    
    chain input {
        type filter hook input priority 0; policy drop;
        
        # Allow loopback
        iif lo accept
        
        # Allow established connections
        ct state established,related accept
        
        # Rate limit ICMPv6
        icmpv6 type { destination-unreachable, packet-too-big, time-exceeded } \
            limit rate 100/second accept
            
        # Allow essential ICMPv6 with rate limiting
        icmpv6 type { neighbor-solicitation, neighbor-advertisement } \
            limit rate 1000/second accept
            
        # RA Guard - only from allowed sources
        icmpv6 type router-advertisement ip6 saddr @allowed_ra_sources accept
        icmpv6 type router-advertisement drop
        
        # Management access
        tcp dport 22 ip6 saddr 2600:beef:1000:mgmt::/64 accept
        tcp dport 8080 ip6 saddr 2600:beef:1000:mgmt::/64 accept
    }
    
    chain forward {
        type filter hook forward priority 0; policy drop;
        
        # Allow established connections
        ct state established,related accept
        
        # Source validation for VMs
        ip6 saddr @vm_address_ranges accept
        
        # Block dangerous extension headers
        exthdr hop-by-hop exists drop
        exthdr routing exists drop  
        exthdr fragment exists drop
        
        # Allow inter-VM communication with VLAN isolation
        iif br-tier1 oif br-tier1 accept
        iif br-tier2 oif br-tier2 accept
        
        # Allow access to shared services
        oif br-shared accept
        
        # Rate limit forwarded ICMPv6
        icmpv6 limit rate 1000/second accept
        
        # Default deny with logging
        log prefix "metald-ipv6-deny: " drop
    }
}
EOF

# Apply firewall rules
sudo nft -f /etc/nftables/ipv6-security.nft

# Make persistent
sudo systemctl enable nftables
```

**1.2 Configure bridge security**:
```bash
# Create secure bridge with RA Guard
sudo ip link add name br-prod type bridge
sudo ip link set br-prod up

# Configure bridge security features
echo 1 > /sys/class/net/br-prod/bridge/multicast_snooping
echo 1 > /sys/class/net/br-prod/bridge/multicast_querier
echo 30 > /sys/class/net/br-prod/bridge/ageing_time

# Add IPv6 address with security
sudo ip -6 addr add 2600:BEEF:1000::1/64 dev br-prod

# Enable IPv6 forwarding for bridge
sudo sysctl -w net.ipv6.conf.br-prod.forwarding=1
sudo sysctl -w net.ipv6.conf.br-prod.accept_ra=0
```

**1.3 Configure Router Advertisement Guard**:
```bash
# Install and configure radvd for controlled RA
sudo tee /etc/radvd.conf << 'EOF'
interface br-prod {
    AdvSendAdvert on;
    AdvManagedFlag off;
    AdvOtherConfigFlag off;
    MinRtrAdvInterval 30;
    MaxRtrAdvInterval 60;
    
    # Security: restrict RA to specific prefix
    prefix 2600:BEEF:1000::/64 {
        AdvOnLink on;
        AdvAutonomous on;
        AdvRouterAddr on;
        AdvValidLifetime 3600;
        AdvPreferredLifetime 1800;
    };
    
    # Security: include DNS servers
    RDNSS 2001:4860:4860::8888 2001:4860:4860::8844 {
        AdvRDNSSLifetime 60;
    };
};
EOF

sudo systemctl enable radvd
sudo systemctl start radvd
```

### Step 2: metald Security Deployment

**2.1 Build metald with security features**:
```bash
cd metald

# Build with security flags
CGO_ENABLED=1 go build \
    -ldflags="-s -w -extldflags=-static" \
    -tags="netgo,osusergo,static_build" \
    -o build/metald ./cmd/api
```

**2.2 Deploy with security hardening**:
```bash
# Create secure directories
sudo mkdir -p /etc/metald/{config,certs}
sudo mkdir -p /var/lib/metald/{wal,pools}  
sudo mkdir -p /var/log/metald
sudo mkdir -p /opt/metald/security

# Set strict permissions
sudo chmod 750 /etc/metald
sudo chmod 700 /var/lib/metald
sudo chmod 755 /var/log/metald

# Create metald user with minimal privileges
sudo useradd -r -s /bin/false -d /var/lib/metald metald
sudo chown -R metald:metald /var/lib/metald
sudo chown -R metald:metald /var/log/metald

# Copy configuration  
sudo cp config.yaml /etc/metald/
sudo chown root:metald /etc/metald/config.yaml
sudo chmod 640 /etc/metald/config.yaml
```

**2.3 Create hardened systemd service**:
```bash
sudo tee /etc/systemd/system/metald.service > /dev/null << 'EOF'
[Unit]
Description=metald VM management service with IPv6 security
After=network.target nftables.service radvd.service
Requires=network.target
Wants=nftables.service radvd.service

[Service]
Type=simple
User=metald
Group=metald
ExecStart=/usr/local/bin/metald
WorkingDirectory=/var/lib/metald
Environment=UNKEY_METALD_CONFIG=/etc/metald/config.yaml

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
RemoveIPC=yes
PrivateTmp=yes
PrivateDevices=yes
RestrictNamespaces=yes
ProtectHostname=yes
ProtectClock=yes
ProtectKernelLogs=yes
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6 AF_NETLINK
SystemCallFilter=@system-service @network-io
SystemCallErrorNumber=EPERM

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096
MemoryHigh=4G
MemoryMax=8G

# Restart policy
Restart=always
RestartSec=5
StartLimitInterval=0

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=metald

[Install]
WantedBy=multi-user.target
EOF

# Enable and start with security
sudo systemctl daemon-reload
sudo systemctl enable metald
sudo systemctl start metald

# Verify security status
sudo systemctl status metald
sudo journalctl -u metald -f | grep -i "ipv6\|security"
```

### Step 3: Security Validation Testing

**3.1 Test IPv6 security controls**:
```bash
# Test RA Guard functionality
echo "Testing RA Guard protection..."

# This should be blocked (rogue RA)
sudo ip netns add test-ra
sudo ip netns exec test-ra radvd -C /tmp/rogue-ra.conf -d 1 2>&1 | grep -i "blocked\|denied" || echo "RA Guard test needed"

# Verify extension header filtering
echo "Testing extension header filtering..."
python3 << 'EOF'
import socket
import struct

# Test that dangerous extension headers are blocked
sock = socket.socket(socket.AF_INET6, socket.SOCK_RAW, socket.IPPROTO_HOPOPTS)
# This should fail or be blocked by firewall
try:
    sock.sendto(b'\x00' * 64, ('2600:beef:1000::100', 0))
    print("WARNING: Hop-by-hop header not blocked!")
except Exception as e:
    print("GOOD: Extension header blocked:", str(e))
sock.close()
EOF

# Test source address validation
echo "Testing source address validation..."
sudo ip netns add test-spoof
sudo ip netns exec test-spoof ip -6 addr add fe80::dead:beef/64 dev lo
# Attempt to send from invalid source should be blocked
```

**3.2 Create production test VM with security**:
```bash
# Test VM creation with full security
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{"path": "/opt/vm-assets/rootfs.ext4"}],
      "network": [{
        "id": "eth0",
        "interface_type": "virtio-net",
        "bridge_name": "br-prod", 
        "vlan_id": "100",
        "ipv6": {
          "address": "2600:BEEF:1000:100::10/64",
          "gateway": "2600:BEEF:1000:100::1",
          "enable_privacy_extensions": true,
          "security": {
            "enable_ra_guard": true,
            "allowed_ra_sources": ["2600:BEEF:1000:100::1"],
            "enable_source_guard": true,
            "strict_source_validation": true,
            "block_hop_by_hop": true,
            "block_routing_header": true,
            "block_fragment_header": true,
            "nd_rate_limit": 100,
            "icmpv6_rate_limit": 100,
            "multicast_rate_limit": 50
          }
        },
        "security_policy": {
          "ingress_rules": [
            "allow icmpv6 from any to any",
            "allow tcp from any to any port 80,443", 
            "deny all from any to any"
          ],
          "egress_rules": [
            "allow tcp from any to any port 53,80,443",
            "allow icmpv6 from any to any",
            "deny all from any to any"
          ],
          "tenant_isolation_level": "strict",
          "log_security_events": true
        }
      }],
      "tenant_id": "test-customer",
      "security_profile": {
        "profile_name": "high",
        "enable_network_microsegmentation": true
      }
    }
  }'
```

**3.3 Validate security monitoring**:
```bash
# Check security event logging
sudo journalctl -u metald | grep -i "security\|violation\|blocked"

# Monitor IPv6 traffic
sudo tcpdump -i br-prod -v ip6 | head -20

# Check firewall logs
sudo journalctl -k | grep "metald-ipv6"

# Verify SIEM integration
curl -X GET http://localhost:8080/debug/security/events | jq .
```

## Migration Strategies

### Strategy 1: Security-First Gradual Migration

**Phase 1: Security Infrastructure** (Week 1)
- Deploy IPv6 security controls (RA Guard, source validation)
- Configure hardened firewalls and monitoring  
- Validate security baseline with penetration testing

**Phase 2: Controlled Rollout** (Week 2-3)
- Enable IPv6 for new VMs only with full security
- Monitor security events and performance impact
- Tune security policies based on operational data

**Phase 3: Production Migration** (Week 4-6)  
- Migrate existing VMs during maintenance windows
- Implement zero-downtime migration procedures
- Complete security compliance validation

### Strategy 2: Blue-Green with Security Validation

**Preparation**:
```bash
# Blue environment (current IPv4)
./metald-current --port 8080 --config current-config.yaml

# Green environment (IPv6 security hardened)
./metald-ipv6 --port 8081 --config security-hardened-config.yaml

# Security testing environment
./metald-test --port 8082 --config security-test-config.yaml
```

**Security-Validated Cutover**:
1. Deploy green environment with full security controls
2. Run comprehensive security penetration testing
3. Validate compliance with security frameworks
4. Cutover with real-time security monitoring
5. Immediate rollback capability on security violations

## Monitoring and Security Operations

### Security Metrics Dashboard

**Critical IPv6 Security Metrics**:
```bash
# Monitor via metrics endpoint
curl http://localhost:9464/metrics | grep -E "(ipv6|security)" | sort

# Key metrics to track:
# - metald_ipv6_ra_guard_blocks_total
# - metald_ipv6_source_guard_violations_total  
# - metald_ipv6_extension_header_blocks_total
# - metald_ipv6_rate_limit_exceeded_total
# - metald_network_security_events_total
```

**Security Event Response Procedures**:
```bash
# Automated security response script
sudo tee /opt/metald/security/incident-response.sh << 'EOF'
#!/bin/bash
# metald IPv6 Security Incident Response

SEVERITY=$1
EVENT_TYPE=$2
VM_ID=$3
TENANT_ID=$4

case $SEVERITY in
    "critical")
        # Immediate isolation
        echo "CRITICAL: Isolating VM $VM_ID"
        curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/IsolateVm \
             -d "{\"vm_id\":\"$VM_ID\"}"
        ;;
    "high")
        # Enhanced monitoring
        echo "HIGH: Enabling enhanced monitoring for tenant $TENANT_ID"
        # Implement tenant-specific monitoring
        ;;
    "medium")
        # Log and alert
        echo "MEDIUM: Logging security event $EVENT_TYPE"
        logger -t metald-security "EVENT: $EVENT_TYPE VM: $VM_ID TENANT: $TENANT_ID"
        ;;
esac
EOF

chmod +x /opt/metald/security/incident-response.sh
```

This production-ready IPv6 deployment guide provides comprehensive security controls necessary for operating a global VM hosting platform with enterprise-grade security and compliance requirements.