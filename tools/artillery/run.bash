#!/bin/bash

# Define the regions
regions=(
  "ams"
  "iad"
  "atl"
  "bog"
  "bos"
  "otp"
  "ord"
  "dfw"
  "den"
  "eze"
  "fra"
  "gdl"
  "hkg"
  "jnb"
  "lhr"
  "lax"
  "mad"
  "mia"
  "yul"
  "bom"
  "cdg"
  "phx"
  "qro"
  "gig"
  "sjc"
  "scl"
  "gru"
  "sea"
  "ewr"
  "sin"
  "arn"
  "syd"
  "nrt"
  "yyz"
  "waw"
)

# Check for required environment variables
if [ -z "$ARTILLERY_CLOUD_API_KEY" ] || [ -z "$FLY_API_KEY" ]; then
  echo "Missing environment variables ARTILLERY_CLOUD_API_KEY or FLY_API_KEY"
  exit 1
fi

FLY_APP_NAME="artillery"
ARTILLERY_YAML_FILE=$1

# Function to run a machine in a specific region
runMachine() {
  local image=$1
  local region=$2
  response=$(curl -s -X POST "https://api.machines.dev/v1/apps/${FLY_APP_NAME}/machines" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${FLY_API_KEY}" \
    -d '{
      "region": "'"${region}"'",
      "config": {
        "init": {
          "exec": [
            "artillery",
            "run",
            "'"${ARTILLERY_YAML_FILE}"'",
            "--record",
            "--tags=platform:fly,region:'"${region}"'"
          ]
        },
        "image": "'"${image}"'",
        "restart": {
          "policy": "no"
        },
        "env": {
          "ARTILLERY_CLOUD_API_KEY": "'"${ARTILLERY_CLOUD_API_KEY}"'",
          "AGENT_AUTH_TOKEN": "'"${AGENT_AUTH_TOKEN}"'"
        },
        "auto_destroy": true,
        "size": "performance-1x"
      }
    }')
  echo "Machine $(echo $response | jq -r '.id') started in ${region}"
}

# Main function
main() {
  echo "Building..."

  # Deploy the app and get the image
  stdout=$(fly deploy --build-only --push --access-token=${FLY_API_KEY} 2>&1)
  echo "$stdout"

  image=$(echo "$stdout" | awk '/image: / {print $2}')
  echo "Image: $image"
  
  if [ -z "$image" ]; then
    echo "Error: Image not detected"
    exit 1
  fi

  # Run machines in all regions
  for region in "${regions[@]}"; do
    runMachine "$image" "$region"
  done

  echo "Deployment complete"
}

main
