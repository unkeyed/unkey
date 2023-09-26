#!/bin/bash

# add this flag if running on prod
# --app unkey-api-production

fly machines list  --json | jq .[].id | sed 's/"//g' | while read machine; do
  fly machines update $machine --restart always --yes 
done
