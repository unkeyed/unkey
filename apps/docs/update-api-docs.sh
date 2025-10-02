#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Running @mintlify/scraping..."
RAW_OUTPUT=$(npx @mintlify/scraping openapi-file ../../go/apps/api/openapi/openapi-generated.yaml -o ./api-reference/v2 2>&1)

# Extract JSON from output (everything from first [ to last ])
GENERATED_JSON=$(echo "$RAW_OUTPUT" | sed -n '/^\[/,/^\]/p')

# Validate JSON
if ! echo "$GENERATED_JSON" | jq empty 2>/dev/null; then
  echo "Error: Could not extract valid JSON from scraping output"
  echo "Raw output was:"
  echo "$RAW_OUTPUT"
  exit 1
fi

echo "Updating docs.json..."
jq --argjson pages "$GENERATED_JSON" '
  .navigation.dropdowns |= map(
    if .dropdown == "api.unkey.com/v2" then
      .groups |= map(
        if .group == "Endpoints" and .hidden != true then
          .pages = $pages
        else
          .
        end
      )
    else
      .
    end
  )
' docs.json > docs.json.tmp && mv docs.json.tmp docs.json

echo "Successfully updated docs.json"
