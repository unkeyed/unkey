#!/bin/bash
# Generate separate systemd-networkd configuration files for 8-bridge setup
# Each bridge needs its own .netdev and .network file

set -e

BRIDGE_COUNT=8
BASE_DIR="$(dirname "$0")/multi-bridge-8"

echo "Generating 8-bridge configuration files in $BASE_DIR"
mkdir -p "$BASE_DIR"

# Generate individual .netdev files for each bridge
for i in $(seq 0 $((BRIDGE_COUNT-1))); do
    cat > "$BASE_DIR/10-br-tenant-$i.netdev" << EOF
[NetDev]
Name=br-tenant-$i
Kind=bridge
Description=Metald VM Bridge $i

[Bridge]
DefaultPVID=none
VLANFiltering=true
STP=false
ForwardDelaySec=0
HelloTimeSec=2
MaxAgeSec=6
EOF
done

# Generate individual .network files for each bridge
for i in $(seq 0 $((BRIDGE_COUNT-1))); do
    cat > "$BASE_DIR/10-br-tenant-$i.network" << EOF
[Match]
Name=br-tenant-$i

[Network]
Description=Metald VM Bridge $i
Address=172.16.$i.1/24
IPv4Forwarding=yes
IPMasquerade=ipv4
ConfigureWithoutCarrier=yes
DNS=8.8.8.8
DNS=8.8.4.4
EOF
done

echo "Generated $BRIDGE_COUNT bridge configuration files:"
ls -la "$BASE_DIR"
