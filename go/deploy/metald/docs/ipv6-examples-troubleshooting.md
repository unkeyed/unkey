# IPv6 Examples and Security Troubleshooting

## Overview

This document provides production-ready examples, security-focused use cases, and comprehensive troubleshooting guidance for IPv6 networking in metald. It covers secure configuration patterns, attack scenario mitigation, and solutions to security and operational issues encountered in production environments.

## Secure Configuration Examples

### Example 1: High-Security Web Server VM

**Scenario**: Deploy a production web server VM with comprehensive IPv6 security controls.

**Configuration**:
```json
{
  "config": {
    "cpu": {"vcpu_count": 2},
    "memory": {"size_bytes": 536870912},
    "boot": {
      "kernel_path": "/opt/vm-assets/vmlinux",
      "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off ipv6.disable_ipv6=0"
    },
    "storage": [{
      "path": "/opt/vm-assets/hardened-web-server-rootfs.ext4",
      "readonly": false
    }],
    "network": [{
      "id": "eth0",
      "interface_type": "virtio-net",
      "bridge_name": "br-dmz",
      "vlan_id": "100",
      "ipv6": {
        "address": "2600:BEEF:1000:web::10/64",
        "gateway": "2600:BEEF:1000:web::1",
        "dns_servers": [
          "2001:4860:4860::8888",
          "2001:4860:4860::8844"
        ],
        "enable_privacy_extensions": true,
        "enable_stable_privacy": true,
        "temp_valid_lifetime": 3600,
        "security": {
          "enable_ra_guard": true,
          "allowed_ra_sources": ["2600:BEEF:1000:web::1"],
          "enable_source_guard": true,
          "strict_source_validation": true,
          "block_hop_by_hop": true,
          "block_routing_header": true,
          "block_fragment_header": true,
          "block_destination_options": true,
          "nd_rate_limit": 100,
          "icmpv6_rate_limit": 100,
          "ra_rate_limit": 10,
          "enable_mld_snooping": true,
          "multicast_rate_limit": 50
        }
      },
      "security_policy": {
        "ingress_rules": [
          "allow icmpv6 type destination-unreachable,packet-too-big,time-exceeded from any to any",
          "allow tcp from any to any port 80,443",
          "allow tcp from 2600:BEEF:1000:mgmt::/64 to any port 22",
          "deny all from any to any"
        ],
        "egress_rules": [
          "allow tcp from any to any port 53,80,443",
          "allow icmpv6 type destination-unreachable,packet-too-big,time-exceeded from any to any",
          "allow icmpv6 type neighbor-solicitation,neighbor-advertisement from any to any",
          "deny all from any to any"
        ],
        "enable_deep_packet_inspection": true,
        "log_security_events": true,
        "bandwidth_limit_mbps": 1000,
        "packet_rate_limit": 10000,
        "tenant_isolation_level": "strict",
        "allowed_destinations": [
          "2600:BEEF:1000:web::/64",
          "2001:4860:4860::/48"
        ]
      }
    }],
    "tenant_id": "secure-web-customer",
    "security_profile": {
      "profile_name": "high",
      "enable_network_microsegmentation": true,
      "require_encrypted_traffic": false,
      "security_groups": ["dmz-web-servers", "monitored"]
    }
  }
}
```

**Secure Deployment**:
```bash
# Create VM with security validation
VM_ID=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d @secure-web-server-config.json | jq -r '.vmId')

# Verify security controls applied
curl -s http://localhost:8080/vmprovisioner.v1.VmService/GetVm \
  -d "{\"vm_id\":\"$VM_ID\"}" | jq '.network[0].ipv6.security'

# Test security controls (should be blocked)
ping6 -c 1 2600:BEEF:1000:web::10 -I fe80::bad:addr 2>/dev/null && echo "SECURITY FAILURE" || echo "Security working"

# Boot with security monitoring
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/BootVm \
  -H "Content-Type: application/json" \
  -d "{\"vm_id\":\"$VM_ID\"}"

# Monitor security events
curl -s http://localhost:8080/debug/security/events | jq '.[] | select(.vm_id == "'$VM_ID'")'
```

### Example 2: Multi-Tier Application with Security Zones

**Scenario**: Deploy a 3-tier application with strict network segmentation and security controls.

**Web Tier (DMZ)**:
```json
{
  "config": {
    "cpu": {"vcpu_count": 2},
    "memory": {"size_bytes": 1073741824},
    "boot": {
      "kernel_path": "/opt/vm-assets/vmlinux",
      "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
    },
    "storage": [{"path": "/opt/vm-assets/nginx-hardened-rootfs.ext4"}],
    "network": [{
      "id": "dmz",
      "interface_type": "virtio-net",
      "vlan_id": "100",
      "bridge_name": "br-dmz",
      "ipv6": {
        "address": "2600:BEEF:1000:dmz::100/64",
        "gateway": "2600:BEEF:1000:dmz::1",
        "security": {
          "enable_ra_guard": true,
          "allowed_ra_sources": ["2600:BEEF:1000:dmz::1"],
          "enable_source_guard": true,
          "strict_source_validation": true,
          "block_hop_by_hop": true,
          "block_routing_header": true,
          "block_fragment_header": true,
          "nd_rate_limit": 100,
          "icmpv6_rate_limit": 100
        }
      },
      "security_policy": {
        "ingress_rules": [
          "allow tcp from any to any port 80,443",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "egress_rules": [
          "allow tcp from any to 2600:BEEF:1001:app::/64 port 8080",
          "allow tcp from any to 2001:4860:4860::/48 port 53",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "tenant_isolation_level": "strict",
        "log_security_events": true
      }
    }],
    "tenant_id": "app-customer-web",
    "security_profile": {
      "profile_name": "high",
      "enable_network_microsegmentation": true,
      "security_groups": ["dmz", "web-tier"]
    }
  }
}
```

**Application Tier (Internal)**:
```json
{
  "config": {
    "cpu": {"vcpu_count": 4},
    "memory": {"size_bytes": 2147483648},
    "boot": {
      "kernel_path": "/opt/vm-assets/vmlinux",
      "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
    },
    "storage": [{"path": "/opt/vm-assets/app-server-rootfs.ext4"}],
    "network": [{
      "id": "app",
      "interface_type": "virtio-net",
      "vlan_id": "200",
      "bridge_name": "br-app",
      "ipv6": {
        "address": "2600:BEEF:1001:app::100/64",
        "gateway": "2600:BEEF:1001:app::1",
        "enable_privacy_extensions": true,
        "security": {
          "enable_ra_guard": true,
          "allowed_ra_sources": ["2600:BEEF:1001:app::1"],
          "enable_source_guard": true,
          "strict_source_validation": true,
          "block_hop_by_hop": true,
          "block_routing_header": true,
          "block_fragment_header": true,
          "nd_rate_limit": 200,
          "icmpv6_rate_limit": 200
        }
      },
      "security_policy": {
        "ingress_rules": [
          "allow tcp from 2600:BEEF:1000:dmz::/64 to any port 8080",
          "allow tcp from 2600:BEEF:1002:mgmt::/64 to any port 22",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "egress_rules": [
          "allow tcp from any to 2600:BEEF:1002:db::/64 port 5432",
          "allow tcp from any to 2001:4860:4860::/48 port 53",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "tenant_isolation_level": "strict",
        "enable_deep_packet_inspection": true,
        "log_security_events": true
      }
    }],
    "tenant_id": "app-customer-app",
    "security_profile": {
      "profile_name": "high",
      "enable_network_microsegmentation": true,
      "security_groups": ["internal", "app-tier"]
    }
  }
}
```

**Database Tier (Secure)**:
```json
{
  "config": {
    "cpu": {"vcpu_count": 8},
    "memory": {"size_bytes": 8589934592},
    "boot": {
      "kernel_path": "/opt/vm-assets/vmlinux",
      "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
    },
    "storage": [{"path": "/opt/vm-assets/postgres-hardened-rootfs.ext4"}],
    "network": [{
      "id": "db",
      "interface_type": "virtio-net",
      "vlan_id": "300",
      "bridge_name": "br-db",
      "ipv6": {
        "address": "2600:BEEF:1002:db::100/64",
        "gateway": "2600:BEEF:1002:db::1",
        "enable_stable_privacy": true,
        "security": {
          "enable_ra_guard": true,
          "allowed_ra_sources": ["2600:BEEF:1002:db::1"],
          "enable_source_guard": true,
          "strict_source_validation": true,
          "block_hop_by_hop": true,
          "block_routing_header": true,
          "block_fragment_header": true,
          "block_destination_options": true,
          "nd_rate_limit": 50,
          "icmpv6_rate_limit": 50,
          "multicast_rate_limit": 10
        }
      },
      "security_policy": {
        "ingress_rules": [
          "allow tcp from 2600:BEEF:1001:app::/64 to any port 5432",
          "allow tcp from 2600:BEEF:1002:mgmt::/64 to any port 22",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "egress_rules": [
          "allow tcp from any to 2001:4860:4860::/48 port 53",
          "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
          "deny all from any to any"
        ],
        "tenant_isolation_level": "strict",
        "enable_deep_packet_inspection": true,
        "log_security_events": true,
        "bandwidth_limit_mbps": 10000
      }
    }],
    "tenant_id": "app-customer-db",
    "security_profile": {
      "profile_name": "high",
      "enable_network_microsegmentation": true,
      "require_encrypted_traffic": true,
      "security_groups": ["secure", "db-tier", "encrypted"]
    }
  }
}
```

### Example 3: Enterprise Multi-Tenant Deployment

**Scenario**: Large enterprise deployment with strict tenant isolation and compliance requirements.

**Tenant A (Financial Services)**:
```json
{
  "config": {
    "cpu": {"vcpu_count": 16},
    "memory": {"size_bytes": 17179869184},
    "boot": {
      "kernel_path": "/opt/vm-assets/vmlinux",
      "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
    },
    "storage": [{"path": "/opt/vm-assets/financial-app-rootfs.ext4"}],
    "network": [
      {
        "id": "primary",
        "interface_type": "virtio-net",
        "vlan_id": "1000",
        "bridge_name": "br-financial",
        "ipv6": {
          "address": "2600:BEEF:2000:fin::100/64",
          "gateway": "2600:BEEF:2000:fin::1",
          "enable_privacy_extensions": true,
          "enable_stable_privacy": true,
          "temp_valid_lifetime": 1800,
          "security": {
            "enable_ra_guard": true,
            "allowed_ra_sources": ["2600:BEEF:2000:fin::1"],
            "enable_source_guard": true,
            "strict_source_validation": true,
            "block_hop_by_hop": true,
            "block_routing_header": true,
            "block_fragment_header": true,
            "block_destination_options": true,
            "nd_rate_limit": 50,
            "icmpv6_rate_limit": 50,
            "ra_rate_limit": 5,
            "enable_mld_snooping": true,
            "multicast_rate_limit": 10
          }
        },
        "security_policy": {
          "ingress_rules": [
            "allow tcp from 2600:BEEF:2000:fin::/64 to any port 443,8443",
            "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
            "deny all from any to any"
          ],
          "egress_rules": [
            "allow tcp from any to 2600:BEEF:2000:fin::/64 port 443,8443,5432",
            "allow tcp from any to 2001:4860:4860::/48 port 853",
            "allow icmpv6 type destination-unreachable,packet-too-big from any to any",
            "deny all from any to any"
          ],
          "enable_deep_packet_inspection": true,
          "log_security_events": true,
          "bandwidth_limit_mbps": 10000,
          "packet_rate_limit": 100000,
          "tenant_isolation_level": "strict",
          "allowed_destinations": [
            "2600:BEEF:2000:fin::/64",
            "2001:4860:4860::/48"
          ]
        }
      },
      {
        "id": "backup",
        "interface_type": "virtio-net",
        "vlan_id": "1001",
        "bridge_name": "br-financial-backup",
        "ipv6": {
          "address": "2600:BEEF:2001:backup::100/64",
          "gateway": "2600:BEEF:2001:backup::1",
          "security": {
            "enable_ra_guard": true,
            "allowed_ra_sources": ["2600:BEEF:2001:backup::1"],
            "enable_source_guard": true,
            "strict_source_validation": true,
            "block_hop_by_hop": true,
            "block_routing_header": true,
            "block_fragment_header": true,
            "nd_rate_limit": 25,
            "icmpv6_rate_limit": 25
          }
        },
        "security_policy": {
          "ingress_rules": [
            "allow tcp from 2600:BEEF:2000:fin::/64 to any port 22",
            "deny all from any to any"
          ],
          "egress_rules": [
            "allow tcp from any to 2600:BEEF:3000:backup::/64 port 443",
            "deny all from any to any"
          ],
          "tenant_isolation_level": "strict",
          "log_security_events": true
        }
      }
    ],
    "tenant_id": "financial-services-corp",
    "security_profile": {
      "profile_name": "high",
      "enable_network_microsegmentation": true,
      "require_encrypted_traffic": true,
      "security_groups": ["financial", "pci-compliant", "regulated"]
    },
    "compliance": {
      "frameworks": ["PCI-DSS", "SOC2", "ISO27001"],
      "data_residency": "US",
      "encryption_required": true,
      "audit_logging": true
    }
  }
}
```

## Security Attack Scenarios and Mitigation

### Scenario 1: IPv6 Neighbor Discovery DoS Attack

**Attack Description**: Attacker floods the network with Neighbor Discovery packets to exhaust the neighbor cache.

**Detection**:
```bash
# Monitor ND cache exhaustion
watch -n 1 'ip -6 neigh show | wc -l'

# Check for ND flood in logs
sudo journalctl -k | grep -i "neighbor.*table.*full"

# Monitor ND rate limiting
curl -s http://localhost:9464/metrics | grep metald_ipv6_nd_rate_limit_exceeded_total
```

**Mitigation Applied**:
```json
{
  "ipv6": {
    "security": {
      "nd_rate_limit": 100,
      "enable_source_guard": true,
      "strict_source_validation": true
    }
  }
}
```

**Validation**:
```bash
# Test ND rate limiting (should be blocked after limit)
for i in {1..200}; do
  ping6 -c 1 2600:BEEF:1000::$(printf "%x" $i) &
done
wait

# Check rate limit counters
curl -s http://localhost:9464/metrics | grep nd_rate_limit
```

### Scenario 2: Router Advertisement Spoofing Attack

**Attack Description**: Rogue device sends malicious Router Advertisements to redirect traffic or cause DoS.

**Detection**:
```bash
# Monitor for unauthorized RAs
sudo tcpdump -i br-prod -v 'icmp6[0] == 134' | while read line; do
  echo "$(date): $line" | grep -v "2600:BEEF:1000::1" && echo "ROGUE RA DETECTED"
done

# Check RA Guard blocks
curl -s http://localhost:9464/metrics | grep metald_ipv6_ra_guard_blocks_total
```

**Mitigation Applied**:
```json
{
  "ipv6": {
    "security": {
      "enable_ra_guard": true,
      "allowed_ra_sources": ["2600:BEEF:1000::1"],
      "ra_rate_limit": 10
    }
  }
}
```

**Validation**:
```bash
# Test RA Guard (should be blocked)
sudo ip netns add rogue-ra
sudo ip netns exec rogue-ra ip link set lo up
sudo ip netns exec rogue-ra ip -6 addr add fe80::bad:actor/64 dev lo

# Try to send rogue RA (should fail)
sudo ip netns exec rogue-ra radvd -C <(cat << 'EOF'
interface lo {
    AdvSendAdvert on;
    prefix fe80::/64 { AdvOnLink on; };
};
EOF
) -d 1 2>&1 | grep -i "blocked\|denied" && echo "RA Guard working"
```

### Scenario 3: IPv6 Extension Header Attack

**Attack Description**: Attacker uses malicious extension headers to bypass firewalls or cause processing delays.

**Detection**:
```bash
# Monitor extension header usage
sudo tcpdump -i br-prod -v 'ip6[6] == 0 or ip6[6] == 43 or ip6[6] == 44'

# Check extension header blocks
curl -s http://localhost:9464/metrics | grep metald_ipv6_extension_header_blocks_total
```

**Mitigation Applied**:
```json
{
  "ipv6": {
    "security": {
      "block_hop_by_hop": true,
      "block_routing_header": true,
      "block_fragment_header": true,
      "block_destination_options": true
    }
  }
}
```

**Validation**:
```bash
# Test extension header blocking
python3 << 'EOF'
import socket
import struct

# Test hop-by-hop header (should be blocked)
try:
    sock = socket.socket(socket.AF_INET6, socket.SOCK_RAW, socket.IPPROTO_HOPOPTS)
    sock.sendto(b'\x00' * 64, ('2600:beef:1000::100', 0))
    print("SECURITY FAILURE: Extension header not blocked")
except Exception as e:
    print("GOOD: Extension header blocked:", str(e))
finally:
    sock.close()
EOF
```

### Scenario 4: IPv6 Source Address Spoofing

**Attack Description**: Attacker spoofs source addresses to bypass access controls or attribution.

**Detection**:
```bash
# Monitor for spoofed sources
sudo tcpdump -i br-prod -v 'not src net 2600:beef:1000::/44'

# Check source guard violations
curl -s http://localhost:9464/metrics | grep metald_ipv6_source_guard_violations_total
```

**Mitigation Applied**:
```json
{
  "ipv6": {
    "security": {
      "enable_source_guard": true,
      "strict_source_validation": true
    }
  }
}
```

**Validation**:
```bash
# Test source validation (should be blocked)
sudo ip netns add spoof-test
sudo ip netns exec spoof-test ip -6 addr add fe80::dead:beef/64 dev lo
sudo ip netns exec spoof-test ip link set lo up

# Try to send with spoofed source (should fail)
sudo ip netns exec spoof-test ping6 -c 1 2600:beef:1000::1 2>&1 | grep -i "network unreachable\|permission denied" && echo "Source guard working"
```

## Production Troubleshooting Scenarios

### Issue 1: VM Cannot Obtain IPv6 Address

**Symptoms**:
```bash
# VM shows no IPv6 address
ip -6 addr show eth0
# Only shows link-local fe80:: address

# SLAAC not working
systemctl status systemd-networkd
```

**Diagnosis Steps**:
```bash
# 1. Check RA reception
sudo tcpdump -i eth0 -v 'icmp6[0] == 134'

# 2. Verify RA Guard configuration
curl -s http://localhost:8080/debug/network/vm/$VM_ID | jq '.ipv6.security.allowed_ra_sources'

# 3. Check bridge RA transmission
sudo radvdump -i br-prod

# 4. Verify VM is in correct VLAN
bridge vlan show dev tap-$VM_ID

# 5. Check security policy blocking
curl -s http://localhost:8080/debug/security/events | jq '.[] | select(.vm_id == "'$VM_ID'" and .event_type == "ra_guard_block")'
```

**Resolution**:
```bash
# Fix RA Guard configuration if needed
curl -X PUT http://localhost:8080/vmprovisioner.v1.VmService/UpdateVmNetwork \
  -d '{
    "vm_id": "'$VM_ID'",
    "interface_id": "eth0", 
    "ipv6": {
      "security": {
        "enable_ra_guard": true,
        "allowed_ra_sources": ["2600:BEEF:1000::1"]
      }
    }
  }'

# Restart RA daemon if needed
sudo systemctl restart radvd

# Trigger RA manually
sudo kill -USR1 $(pgrep radvd)
```

### Issue 2: IPv6 Connectivity Between VMs Blocked

**Symptoms**:
```bash
# Ping fails between VMs in same subnet
ping6 2600:BEEF:1000::101
# connect: Network is unreachable

# TCP connections fail
telnet 2600:beef:1000::101 80
# Connection timed out
```

**Diagnosis Steps**:
```bash
# 1. Check firewall rules
sudo nft list table ip6 metald_security

# 2. Verify VLAN configuration
bridge vlan show | grep $VM_ID

# 3. Check security policy
curl -s http://localhost:8080/debug/vm/$VM_ID/security-policy | jq '.ingress_rules'

# 4. Monitor blocked packets
sudo journalctl -k | grep "metald-ipv6-deny"

# 5. Check bridge forwarding
cat /sys/class/net/br-prod/bridge/multicast_snooping
```

**Resolution**:
```bash
# Update security policy to allow communication
curl -X PUT http://localhost:8080/vmprovisioner.v1.VmService/UpdateSecurityPolicy \
  -d '{
    "vm_id": "'$VM_ID'",
    "security_policy": {
      "ingress_rules": [
        "allow tcp from 2600:BEEF:1000::/64 to any port 80,443",
        "allow icmpv6 from any to any",
        "deny all from any to any"
      ]
    }
  }'

# Restart network stack if needed
sudo systemctl restart systemd-networkd
```

### Issue 3: IPv6 Pool Exhaustion

**Symptoms**:
```bash
# VM creation fails
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm -d @config.json
# {"code": "resource_exhausted", "message": "no available IPv6 addresses in pool 'default'"}

# Pool utilization high
curl -s http://localhost:9464/metrics | grep metald_ipv6_pool_utilization
# metald_ipv6_pool_utilization{pool_id="default"} 0.95
```

**Diagnosis Steps**:
```bash
# 1. Check pool utilization
curl -s http://localhost:8080/debug/ipv6/pools | jq '.[] | {id: .id, utilization: .utilization, available: .available_addresses}'

# 2. Find unused allocations
curl -s http://localhost:8080/debug/ipv6/allocations | jq '.[] | select(.vm_id == null)'

# 3. Check for leaked allocations
for vm in $(curl -s http://localhost:8080/vmprovisioner.v1.VmService/ListVms | jq -r '.vms[].vmId'); do
  curl -s http://localhost:8080/vmprovisioner.v1.VmService/GetVm -d "{\"vm_id\":\"$vm\"}" | jq '.state' | grep -q "TERMINATED" && echo "Leaked allocation: $vm"
done

# 4. Monitor allocation rate
curl -s http://localhost:9464/metrics | grep metald_ipv6_allocations_per_hour
```

**Resolution**:
```bash
# Clean up leaked allocations
curl -X POST http://localhost:8080/debug/ipv6/cleanup-allocations

# Expand pool if needed (requires RIR allocation)
curl -X PUT http://localhost:8080/debug/ipv6/pools/default \
  -d '{
    "prefix": "2600:BEEF:1001::/48",
    "description": "Expanded pool for growth"
  }'

# Add new pool for specific tenant
curl -X POST http://localhost:8080/debug/ipv6/pools \
  -d '{
    "id": "enterprise-pool",
    "prefix": "2600:BEEF:2000::/48",
    "description": "Dedicated enterprise customer pool",
    "rate_limit": 1000
  }'
```

### Issue 4: Security Event Investigation

**Symptoms**:
```bash
# High security event rate
curl -s http://localhost:9464/metrics | grep metald_network_security_events_total
# metald_network_security_events_total{severity="high"} 156

# SIEM alerts on IPv6 violations
tail -f /var/log/metald/security.log | grep "IPv6_VIOLATION"
```

**Investigation Steps**:
```bash
# 1. Analyze security events by type
curl -s http://localhost:8080/debug/security/events | jq 'group_by(.event_type) | map({event_type: .[0].event_type, count: length}) | sort_by(.count) | reverse'

# 2. Identify top violating VMs
curl -s http://localhost:8080/debug/security/events | jq 'group_by(.vm_id) | map({vm_id: .[0].vm_id, violations: length}) | sort_by(.violations) | reverse | .[0:10]'

# 3. Correlate with tenant activity
curl -s http://localhost:8080/debug/security/events | jq 'group_by(.tenant_id) | map({tenant_id: .[0].tenant_id, violations: length}) | sort_by(.violations) | reverse'

# 4. Check for attack patterns
curl -s http://localhost:8080/debug/security/events | jq '.[] | select(.event_type == "extension_header_block" and .timestamp > "'$(date -d '1 hour ago' -Is)'")' | wc -l
```

**Response Actions**:
```bash
# Isolate compromised VM
VM_ID="vm-suspicious-123"
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/IsolateVm \
  -d "{\"vm_id\":\"$VM_ID\", \"reason\":\"Security violation investigation\"}"

# Enhance monitoring for tenant
TENANT_ID="suspicious-tenant"
curl -X POST http://localhost:8080/debug/security/enhance-monitoring \
  -d "{\"tenant_id\":\"$TENANT_ID\", \"duration\":\"24h\", \"log_level\":\"DEBUG\"}"

# Block source if external attack
ATTACKER_IP="2001:db8:bad::actor"
sudo nft add element ip6 metald_security blocked_sources { $ATTACKER_IP }

# Generate security report
curl -s http://localhost:8080/debug/security/report \
  -d "{\"start_time\":\"$(date -d '24 hours ago' -Is)\", \"end_time\":\"$(date -Is)\"}" > security-report-$(date +%Y%m%d).json
```

## Performance Optimization Troubleshooting

### Issue 5: IPv6 Performance Degradation

**Symptoms**:
```bash
# High latency
ping6 -c 10 2600:BEEF:1000::100 | grep "time="
# time=150ms (normally <1ms)

# Low throughput
iperf3 -6 -c 2600:beef:1000::100
# 100 Mbits/sec (normally 10 Gbits/sec)
```

**Diagnosis**:
```bash
# 1. Check security processing overhead
curl -s http://localhost:9464/metrics | grep metald_ipv6_security_processing_time

# 2. Monitor neighbor cache efficiency
ip -6 neigh show | grep -c STALE

# 3. Check MTU discovery issues
ping6 -M do -s 1450 2600:beef:1000::100

# 4. Analyze bridge performance
ethtool -S br-prod | grep -E "(rx_|tx_).*drop"

# 5. Check CPU utilization in security processing
top -p $(pgrep metald)
```

**Optimization**:
```bash
# Tune neighbor cache parameters
sudo sysctl -w net.ipv6.neigh.default.gc_thresh3=8192
sudo sysctl -w net.ipv6.neigh.default.base_reachable_time=30

# Optimize bridge parameters
echo 1 > /sys/class/net/br-prod/bridge/multicast_fast_leave
echo 100 > /sys/class/net/br-prod/bridge/hash_max

# Reduce security processing if safe
curl -X PUT http://localhost:8080/debug/security/optimize \
  -d '{
    "batch_processing": true,
    "hardware_acceleration": true,
    "cache_policies": true
  }'

# Enable jumbo frames if supported
ip link set br-prod mtu 9000
```

This comprehensive troubleshooting guide provides the necessary tools and procedures to maintain a secure, high-performance IPv6 network infrastructure in production environments.