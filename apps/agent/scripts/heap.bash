#!/bin/bash

set -e


# The following environment variables are required:
# PPROF_USERNAME
# PPROF_PASSWORD
# MACHINE_ID

# Usage
# PPROF_USERNAME=xxx PPROF_PASSWORD=xxx MACHINE_ID=xxx bash ./scripts/heap.bash 

url="https://api.unkey.cloud"
seconds=60
now=$(date +"%Y-%m-%d_%H-%M-%S")
filename="heap-$now.out"


echo "Checking machine status"
curl -s -o /dev/null -w "%{http_code}" $url/v1/liveness -H "Fly-Force-Instance-Id: $MACHINE_ID"

echo ""
echo ""

echo "Fetching heap profile from $url, this takes $seconds seconds..."
curl -u $PPROF_USERNAME:$PPROF_PASSWORD \
  $url/debug/pprof/heap?seconds=$seconds \
  -H "Fly-Force-Instance-Id: $MACHINE_ID" \
   > $filename
go tool pprof -http=:9000 $filename