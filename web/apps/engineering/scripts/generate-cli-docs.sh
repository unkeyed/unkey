#!/bin/bash
set -e

# Build the binary first
echo "Building unkey binary..."
cd ../../../
make build-go
cd web/apps/engineering/

# Set docs directory relative to engineering/
DOCS_DIR="./content/docs/cli"

echo "Starting automated documentation generation..."
echo "DOCS_DIR: $DOCS_DIR"

# Get available commands dynamically
echo "Discovering available commands..."
AVAILABLE_COMMANDS=$(../../../unkey --help 2>/dev/null | awk '/COMMANDS:/{flag=1; next} /^$/{flag=0} flag && /^   [a-zA-Z]/ {gsub(/,.*/, "", $1); print $1}' | grep -v "help" | tr '\n' ' ')
echo "Found commands: $AVAILABLE_COMMANDS"

if [ -z "$AVAILABLE_COMMANDS" ]; then
  echo "ERROR: No commands found. Is the binary working?"
  ../../../unkey --help
  exit 1
fi

total=0
success=0
generated_commands=()

for cmd in $AVAILABLE_COMMANDS; do
  echo ""
  echo "=== Processing $cmd ==="
  mkdir -p "$DOCS_DIR/$cmd"
  total=$((total + 1))

  if ../../../unkey $cmd mdx >"$DOCS_DIR/$cmd/index.mdx" 2>/dev/null; then
    echo "✓ Generated $DOCS_DIR/$cmd/index.mdx"
    success=$((success + 1))
    generated_commands+=("$cmd")
  else
    echo "✗ Failed to generate $cmd"
    continue
  fi

  # Check for subcommands
  help_output=$(../../../unkey $cmd --help 2>/dev/null)
  if echo "$help_output" | grep -q "COMMANDS:"; then
    subcmds=$(echo "$help_output" | awk '/COMMANDS:/{flag=1; next} /^$/{flag=0} flag && /^   [a-zA-Z]/ {gsub(/,.*/, "", $1); print $1}' | grep -v "help")
    subcount=$(echo "$subcmds" | wc -w)
    echo "  Found $subcount subcommands: $subcmds"

    for subcmd in $subcmds; do
      if [ "$subcmd" != "" ]; then
        mkdir -p "$DOCS_DIR/$cmd/$subcmd"
        total=$((total + 1))
        if ../../../unkey $cmd $subcmd mdx >"$DOCS_DIR/$cmd/$subcmd/index.mdx" 2>/dev/null; then
          echo "  ✓ Generated $DOCS_DIR/$cmd/$subcmd/index.mdx"
          success=$((success + 1))
        else
          echo "  ✗ Failed to generate $cmd/$subcmd"
        fi
      fi
    done
  else
    echo "  No subcommands found"
  fi
done

# Update meta.json with generated commands
META_FILE="./content/docs/architecture/services/meta.json"
echo ""
echo "Updating $META_FILE..."

# Read existing meta.json or create minimal structure
if [ -f "$META_FILE" ]; then
  EXISTING_PAGES=$(jq -r '.pages // []' "$META_FILE")
  META_BASE=$(jq 'del(.pages)' "$META_FILE")
else
  EXISTING_PAGES='[]'
  META_BASE='{"title": "Services", "icon": "Pencil", "root": false}'
fi

# Convert generated commands to JSON array
GENERATED_JSON=$(printf '%s\n' "${generated_commands[@]}" | jq -R . | jq -s .)

# Merge existing pages with generated commands and remove duplicates
COMBINED_PAGES=$(echo "$EXISTING_PAGES $GENERATED_JSON" | jq -s 'add | unique')

# Create the complete meta.json
echo "$META_BASE" | jq --argjson pages "$COMBINED_PAGES" '. + {pages: $pages}' >"$META_FILE"

echo "✓ Updated $META_FILE with $(echo "$COMBINED_PAGES" | jq 'length') total pages"
echo ""
echo "Summary: $success/$total files generated successfully"

if [ $success -eq 0 ]; then
  echo "ERROR: No documentation files were generated"
  exit 1
fi
