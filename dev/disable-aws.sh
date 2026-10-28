#!/usr/bin/env bash

for aws_variable in "${!AWS_@}"; do
  unset "$aws_variable"
done
unset aws_variable

export AWS_PROFILE=unkey-local-disabled
export AWS_CONFIG_FILE=/dev/null
export AWS_SHARED_CREDENTIALS_FILE=/dev/null
export AWS_EC2_METADATA_DISABLED=true

# Source this file for Bash; execute it to return the same settings to Tilt.
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  jq -n 'env | with_entries(select(.key | startswith("AWS_")))'
fi
