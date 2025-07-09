#!/usr/bin/env bash
set -euo pipefail

# Version Hook Script
# Automatically bumps patch version and updates changelog for pillar services
# and their clients when their Go code changes

# Define pillar services
PILLAR_SERVICES=("assetmanagerd" "billaged" "builderd" "metald")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log function
log() {
    echo -e "${BLUE}[VERSION-HOOK]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[VERSION-HOOK WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[VERSION-HOOK ERROR]${NC} $1"
}

success() {
    echo -e "${GREEN}[VERSION-HOOK SUCCESS]${NC} $1"
}

# Function to get current version from Makefile
get_current_version() {
    local service=$1
    local type=${2:-"service"}  # "service" or "client"
    local makefile
    
    if [[ "$type" == "client" ]]; then
        makefile="${service}/client/Makefile"
    else
        makefile="${service}/Makefile"
    fi
    
    if [[ ! -f "$makefile" ]]; then
        error "Makefile not found: $makefile"
        return 1
    fi
    
    # Extract version using sed/grep
    grep -E '^VERSION \?= ' "$makefile" | sed -E 's/VERSION \?= ([0-9]+\.[0-9]+\.[0-9]+).*/\1/'
}

# Function to bump patch version
bump_patch_version() {
    local version=$1
    # Split version into major.minor.patch
    local major=$(echo "$version" | cut -d. -f1)
    local minor=$(echo "$version" | cut -d. -f2)
    local patch=$(echo "$version" | cut -d. -f3)
    
    # Increment patch
    patch=$((patch + 1))
    
    echo "${major}.${minor}.${patch}"
}

# Function to update version in Makefile only
update_version_in_makefile() {
    local service=$1
    local new_version=$2
    local type=${3:-"service"}  # "service" or "client"
    local makefile
    
    if [[ "$type" == "client" ]]; then
        makefile="${service}/client/Makefile"
    else
        makefile="${service}/Makefile"
    fi
    
    # Update Makefile version
    if [[ -f "$makefile" ]]; then
        sed -i.bak "s/^VERSION ?= [0-9][^[:space:]]*/VERSION ?= ${new_version}/" "$makefile"
        rm "${makefile}.bak" 2>/dev/null || true
        log "Updated ${service} ${type} Makefile version to ${new_version}"
    else
        warn "Makefile not found: $makefile"
    fi
}

# Function to update changelog
update_changelog() {
    local service=$1
    local new_version=$2
    local summary=$3
    local type=${4:-"service"}  # "service" or "client"
    local changelog_file
    
    if [[ "$type" == "client" ]]; then
        changelog_file="${service}/client/CHANGELOG.md"
    else
        changelog_file="${service}/CHANGELOG.md"
    fi
    
    # Get current date
    local date=$(date '+%Y-%m-%d')
    
    # Create changelog if it doesn't exist
    if [[ ! -f "$changelog_file" ]]; then
        log "Creating new changelog for ${service} ${type}"
        cat > "$changelog_file" << EOF
# Changelog

All notable changes to ${service} ${type} will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [${new_version}] - ${date}

### Changed
- ${summary}

EOF
    else
        # Insert new version at the top (after the header)
        local temp_file=$(mktemp)
        {
            # Copy header (everything before first version entry)
            awk '/^## \[/{exit} {print}' "$changelog_file"
            
            # Add new version entry
            echo "## [${new_version}] - ${date}"
            echo ""
            echo "### Changed"
            echo "- ${summary}"
            echo ""
            
            # Copy rest of the file (starting from first version entry)
            awk '/^## \[/{found=1} found{print}' "$changelog_file"
        } > "$temp_file"
        
        mv "$temp_file" "$changelog_file"
    fi
    
    success "Updated changelog for ${service} ${type} v${new_version}"
}

# Function to detect changes in a service (excluding client)
detect_service_changes() {
    local service=$1
    
    # Check if there are any *.go files that have been modified in the service
    # but exclude the client directory
    local go_files_changed=$(git diff --cached --name-only | grep -E "(^|/)${service}/.*\.go$" | grep -v "${service}/client/" || true)
    
    if [[ -n "$go_files_changed" ]]; then
        log "Detected Go code changes in ${service} service:"
        echo "$go_files_changed" | sed 's/^/  - /'
        return 0
    else
        return 1
    fi
}

# Function to detect changes in a client
detect_client_changes() {
    local service=$1
    
    # Check if there are any *.go files that have been modified in the client directory
    local go_files_changed=$(git diff --cached --name-only | grep -E "(^|/)${service}/client/.*\.go$" || true)
    
    if [[ -n "$go_files_changed" ]]; then
        log "Detected Go code changes in ${service} client:"
        echo "$go_files_changed" | sed 's/^/  - /'
        return 0
    else
        return 1
    fi
}

# Function to generate change summary
generate_change_summary() {
    local service=$1
    local type=${2:-"service"}  # "service" or "client"
    local pattern
    
    if [[ "$type" == "client" ]]; then
        pattern="(^|/)${service}/client/.*\.go$"
    else
        pattern="(^|/)${service}/.*\.go$"
        # Exclude client directory for service changes
        if [[ "$type" == "service" ]]; then
            pattern="${pattern}|grep -v ${service}/client/"
        fi
    fi
    
    # Get list of changed files
    local changed_files
    if [[ "$type" == "service" ]]; then
        changed_files=$(git diff --cached --name-only | grep -E "(^|/)${service}/.*\.go$" | grep -v "${service}/client/" | head -5)
    else
        changed_files=$(git diff --cached --name-only | grep -E "(^|/)${service}/client/.*\.go$" | head -5)
    fi
    
    local file_count=$(echo "$changed_files" | wc -l)
    
    if [[ -z "$changed_files" ]]; then
        echo "Update ${type} code"
    elif [[ $file_count -eq 1 ]]; then
        local filename=$(basename "$changed_files")
        echo "Update ${filename} in ${type}"
    elif [[ $file_count -le 3 ]]; then
        local filenames=$(echo "$changed_files" | xargs -I {} basename {} | tr '\n' ', ' | sed 's/, $//')
        echo "Update ${filenames} in ${type}"
    else
        echo "Update ${file_count} Go files in ${type}"
    fi
}

# Main function
main() {
    log "Starting version hook..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        error "Not in a git repository"
        exit 1
    fi
    
    # Check if we're in the correct directory
    if [[ ! -f "CLAUDE.md" ]]; then
        error "Not in the deploy directory (CLAUDE.md not found)"
        exit 1
    fi
    
    local changes_made=false
    
    # Process each pillar service
    for service in "${PILLAR_SERVICES[@]}"; do
        if [[ ! -d "$service" ]]; then
            warn "Service directory not found: $service"
            continue
        fi
        
        # Check for service changes (excluding client)
        if detect_service_changes "$service"; then
            # Get current version
            local current_version
            if ! current_version=$(get_current_version "$service" "service"); then
                error "Failed to get current version for $service service"
                continue
            fi
            
            # Bump patch version
            local new_version
            new_version=$(bump_patch_version "$current_version")
            
            # Generate change summary
            local summary
            summary=$(generate_change_summary "$service" "service")
            
            log "Processing ${service} service: ${current_version} -> ${new_version}"
            
            # Update version in Makefile
            update_version_in_makefile "$service" "$new_version" "service"
            
            # Update changelog
            update_changelog "$service" "$new_version" "$summary" "service"
            
            # Stage the changes
            git add "${service}/Makefile"
            git add "${service}/CHANGELOG.md"
            
            success "Processed ${service} service v${new_version}"
            changes_made=true
        fi
        
        # Check for client changes
        if [[ -d "${service}/client" ]] && detect_client_changes "$service"; then
            # Get current client version
            local current_client_version
            if ! current_client_version=$(get_current_version "$service" "client"); then
                error "Failed to get current version for $service client"
                continue
            fi
            
            # Bump patch version
            local new_client_version
            new_client_version=$(bump_patch_version "$current_client_version")
            
            # Generate change summary
            local client_summary
            client_summary=$(generate_change_summary "$service" "client")
            
            log "Processing ${service} client: ${current_client_version} -> ${new_client_version}"
            
            # Update version in client Makefile
            update_version_in_makefile "$service" "$new_client_version" "client"
            
            # Update client changelog
            update_changelog "$service" "$new_client_version" "$client_summary" "client"
            
            # Stage the client changes
            git add "${service}/client/Makefile"
            git add "${service}/client/CHANGELOG.md"
            
            success "Processed ${service} client v${new_client_version}"
            changes_made=true
        fi
    done
    
    if [[ "$changes_made" == true ]]; then
        log "Version updates complete. Changes have been staged."
        log "You can now commit with: git commit"
    else
        log "No pillar service changes detected. No version bumps needed."
    fi
}

# Check if script is being sourced or executed
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi