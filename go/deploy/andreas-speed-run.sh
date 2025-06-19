#!/usr/bin/env bash

set -euo pipefail

# Do all the OS package stuff first then run this script.
make clean-all && sleep 1
make generate && sleep 2
make o11y-stop && make o11y
make -C spire install && sleep 1
make -C spire service-start-server && sleep 1
make -C spire bootstrap-agent && sleep 1
make -C spire register-agent && sleep 1
make -C spire register-services && sleep 1
sudo ./scripts/install-firecracker.sh && sleep 1
sudo ./scripts/setup-vm-assets.sh && sleep 1
# sudo ./scripts/register-vm-assets.sh && sleep 1
make -C assetmanagerd install && sleep 1
make -C builderd install && sleep 1
make -C builderd install && sleep 1
make -C billaged install && sleep 1
make -C metald install && sleep 1
sleep 1
./metald/contrib/example-client/metald-client --spiffe-socket=/var/lib/spire/agent/agent.sock --tls-mode spiffe --action create-and-boot && sleep 1
ping 10.100.0.2
