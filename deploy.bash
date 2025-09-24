#!/bin/bash



for i in {1..100}; do
  go run . deploy \
    --context=./demo_api \
    --workspace-id="ws_4pqNeih1b62QPv6g" \
    --project-id="proj_4pqPKyhJ3b9fCnuH" \
    --control-plane-url="http://localhost:7091" \
    --env=production &
    sleep 1
done

wait
