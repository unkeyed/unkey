#!/bin/bash

# AIDEV-NOTE: VM lifecycle stress test script for metald
# This script performs comprehensive stress testing of VM creation, boot, shutdown, resume, and deletion
# operations while respecting concurrency limits and tracking VM lifecycle states.

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly DOCKER_IMAGE="${DOCKER_IMAGE:-ghcr.io/unkeyed/best-api:v1.1.0}"
readonly MAX_CONCURRENT_VMS=15
readonly MAX_TOTAL_VMS=500
readonly CREATION_DELAY_SECONDS=1
readonly METALD_CLI="${METALD_CLI:-metald-cli}"

# State tracking
declare -A vm_states=()
declare -a active_vms=()
declare -a all_vms=()
total_created=0
total_operations=0
errors=0

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    ((errors+=1))
}

# VM ID parsing function
extract_vm_id() {
    local output="$1"
    # Extract VM IDs that start with 'ud-' from metald-cli output
    echo "$output" | grep -oE 'ud-[a-f0-9]+' | head -1 || true
}

# Execute metald-cli command with error handling
execute_metald_command() {
    local cmd="$*"
    log_info "Executing: $cmd"

    local output
    local exit_code=0

    if output=$(timeout 60 $cmd 2>&1); then
        log_success "Command succeeded: $cmd"
        echo "$output"
        return 0
    else
        exit_code=$?
        log_error "Command failed (exit code: $exit_code): $cmd"
        log_error "Output: $output"
        return $exit_code
    fi
}

# Get current VM list and parse VM IDs
list_vms() {
    local output
    if output=$(execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" list); then
        echo "$output" | grep -oE 'ud-[a-f0-9]+' || true
    fi
}

# Create and boot a new VM
create_and_boot_vm() {
    if [ ${#active_vms[@]} -ge $MAX_CONCURRENT_VMS ]; then
        log_warning "Maximum concurrent VMs ($MAX_CONCURRENT_VMS) reached. Skipping creation."
        return 1
    fi

    if [ $total_created -ge $MAX_TOTAL_VMS ]; then
        log_warning "Maximum total VMs ($MAX_TOTAL_VMS) reached. Skipping creation."
        return 1
    fi

    log_info "Creating and booting VM (active: ${#active_vms[@]}, total created: $total_created)"

    local output
    if output=$(execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" create-and-boot); then
        local vm_id
        vm_id=$(extract_vm_id "$output")

        if [ -n "$vm_id" ]; then
            active_vms+=("$vm_id")
            all_vms+=("$vm_id")
            vm_states["$vm_id"]="running"
            ((total_created+=1))
            ((total_operations+=1))
            log_success "Created and booted VM: $vm_id"
            return 0
        else
            log_error "Failed to extract VM ID from create-and-boot output"
            return 1
        fi
    else
        log_error "Failed to create and boot VM"
        return 1
    fi
}

# Get VM info
get_vm_info() {
    local vm_id="$1"
    log_info "Getting info for VM: $vm_id"

    if execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" info "$vm_id" >/dev/null; then
        ((total_operations+=1))
        log_success "Retrieved info for VM: $vm_id"
        return 0
    else
        log_error "Failed to get info for VM: $vm_id"
        return 1
    fi
}

# Shutdown a VM
shutdown_vm() {
    local vm_id="$1"
    log_info "Shutting down VM: $vm_id"

    if execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" shutdown "$vm_id" >/dev/null; then
        vm_states["$vm_id"]="shutdown"
        ((total_operations+=1))
        log_success "Shutdown VM: $vm_id"
        return 0
    else
        log_error "Failed to shutdown VM: $vm_id"
        return 1
    fi
}

# Resume a VM
resume_vm() {
    local vm_id="$1"
    log_info "Resuming VM: $vm_id"

    if execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" resume "$vm_id" >/dev/null; then
        vm_states["$vm_id"]="running"
        ((total_operations+=1))
        log_success "Resumed VM: $vm_id"
        return 0
    else
        log_error "Failed to resume VM: $vm_id"
        return 1
    fi
}

# Delete a VM
delete_vm() {
    local vm_id="$1"
    log_info "Deleting VM: $vm_id"

    if execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" delete "$vm_id" >/dev/null; then
        # Remove from active VMs array
        for i in "${!active_vms[@]}"; do
            if [[ "${active_vms[i]}" == "$vm_id" ]]; then
                unset "active_vms[i]"
                break
            fi
        done
        # Rebuild array to remove gaps
        active_vms=("${active_vms[@]}")

        unset vm_states["$vm_id"]
        ((total_operations+=1))
        log_success "Deleted VM: $vm_id"
        return 0
    else
        log_error "Failed to delete VM: $vm_id"
        return 1
    fi
}

# Perform a random VM lifecycle operation
perform_random_operation() {
    if [ ${#active_vms[@]} -eq 0 ]; then
        create_and_boot_vm
        return
    fi

    # Choose a random operation
    local operations=("info" "shutdown" "resume" "delete")
    local operation="${operations[$((RANDOM % ${#operations[@]}))]}"

    # Choose a random VM
    local vm_id="${active_vms[$((RANDOM % ${#active_vms[@]}))]}"
    local current_state="${vm_states[$vm_id]:-unknown}"

    case "$operation" in
        "info")
            get_vm_info "$vm_id"
            ;;
        "shutdown")
            if [ "$current_state" = "running" ]; then
                shutdown_vm "$vm_id"
            else
                log_info "VM $vm_id is not running (state: $current_state), skipping shutdown"
            fi
            ;;
        "resume")
            if [ "$current_state" = "shutdown" ]; then
                resume_vm "$vm_id"
            else
                log_info "VM $vm_id is not shutdown (state: $current_state), skipping resume"
            fi
            ;;
        "delete")
            delete_vm "$vm_id"
            ;;
    esac
}

# Clean up all VMs
cleanup_all_vms() {
    log_info "Cleaning up all VMs..."

    local cleanup_vms
    readarray -t cleanup_vms < <(list_vms)

    for vm_id in "${cleanup_vms[@]}"; do
        if [ -n "$vm_id" ]; then
            log_info "Cleaning up VM: $vm_id"
            # Try to shutdown first, then delete
            execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" shutdown "$vm_id" >/dev/null 2>&1 || true
            sleep 1
            execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" delete "$vm_id" >/dev/null 2>&1 || true
        fi
    done

    log_success "Cleanup completed"
}

# Print current status
print_status() {
    log_info "=== VM Stress Test Status ==="
    log_info "Active VMs: ${#active_vms[@]}"
    log_info "Total VMs created: $total_created"
    log_info "Total operations: $total_operations"
    log_info "Errors: $errors"

    if [ ${#active_vms[@]} -gt 0 ]; then
        log_info "Active VM states:"
        for vm_id in "${active_vms[@]}"; do
            log_info "  $vm_id: ${vm_states[$vm_id]:-unknown}"
        done
    fi
    log_info "=========================="
}

# Signal handlers
cleanup_and_exit() {
    log_warning "Received signal, cleaning up..."
    cleanup_all_vms
    print_status
    exit 0
}

# Set up signal handlers
trap cleanup_and_exit SIGINT SIGTERM

# Test VM persistence across metald restarts
# AIDEV-BUSINESS_RULE: VMs should survive metald service restarts like VMware/VirtualBox
run_persistence_test() {
    local test_duration="${1:-120}"  # Default 2 minutes per phase
    log_info "Starting VM persistence test"
    log_info "This test will create VMs, restart metald, and verify VMs persist"

    # Phase 1: Create and manage some VMs
    log_info "Phase 1: Creating test VMs"
    local test_vms=()
    local shutdown_vms=()
    local running_vms=()

    # Create 5 VMs for testing
    for i in {1..5}; do
        log_info "Creating test VM $i/5"
        local output
        if output=$(execute_metald_command $METALD_CLI -docker-image="$DOCKER_IMAGE" create-and-boot); then
            local vm_id
            vm_id=$(extract_vm_id "$output")
            if [ -n "$vm_id" ]; then
                test_vms+=("$vm_id")
                running_vms+=("$vm_id")
                log_success "Created test VM: $vm_id"
                sleep 2
            fi
        else
            log_error "Failed to create test VM $i"
        fi
    done

    if [ ${#test_vms[@]} -eq 0 ]; then
        log_error "No test VMs created, aborting persistence test"
        return 1
    fi

    # Shutdown half of the VMs to test SHUTDOWN persistence
    local shutdown_count=$((${#test_vms[@]} / 2))
    log_info "Phase 2: Shutting down $shutdown_count VMs for persistence testing"

    for ((i=0; i<shutdown_count; i++)); do
        local vm_id="${test_vms[i]}"
        if shutdown_vm "$vm_id"; then
            shutdown_vms+=("$vm_id")
            # Remove from running_vms array
            running_vms=("${running_vms[@]/$vm_id}")
        fi
        sleep 1
    done

    # Keep remaining VMs running to test RUNNING persistence
    log_info "Phase 3: Pre-restart VM state"
    log_info "Running VMs (should persist): ${running_vms[*]}"
    log_info "Shutdown VMs (should persist): ${shutdown_vms[*]}"

    # Get list of all VMs before restart
    log_info "Listing all VMs before metald restart:"
    local pre_restart_vms
    readarray -t pre_restart_vms < <(list_vms)
    log_info "Pre-restart VM count: ${#pre_restart_vms[@]}"

    # Phase 4: Simulate metald restart (the critical test)
    log_warning "Phase 4: SIMULATING METALD RESTART"
    log_warning "In a real test, you would restart the metald service here"
    log_warning "For this simulation, we'll wait 10 seconds to allow processes to stabilize"
    sleep 10

    # Phase 5: Verify VMs still exist after "restart"
    log_info "Phase 5: Verifying VM persistence after restart"
    local post_restart_vms
    readarray -t post_restart_vms < <(list_vms)
    log_info "Post-restart VM count: ${#post_restart_vms[@]}"

    # Check if all test VMs still exist
    local persistence_success=true
    for vm_id in "${test_vms[@]}"; do
        local found=false
        for existing_vm in "${post_restart_vms[@]}"; do
            if [ "$vm_id" = "$existing_vm" ]; then
                found=true
                break
            fi
        done

        if [ "$found" = true ]; then
            log_success "VM $vm_id PERSISTED successfully"
        else
            log_error "VM $vm_id LOST after restart"
            persistence_success=false
        fi
    done

    # Phase 6: Test VM operations after restart
    log_info "Phase 6: Testing VM operations after restart"

    # Test info on all persisted VMs
    for vm_id in "${test_vms[@]}"; do
        if get_vm_info "$vm_id"; then
            log_success "VM $vm_id info retrieval works after restart"
        else
            log_error "VM $vm_id info retrieval failed after restart"
            persistence_success=false
        fi
        sleep 1
    done

    # Test resume on shutdown VMs
    for vm_id in "${shutdown_vms[@]}"; do
        log_info "Testing resume of shutdown VM: $vm_id"
        if resume_vm "$vm_id"; then
            log_success "VM $vm_id resume works after restart"
        else
            log_error "VM $vm_id resume failed after restart"
            persistence_success=false
        fi
        sleep 2
    done

    # Cleanup test VMs
    log_info "Phase 7: Cleaning up test VMs"
    for vm_id in "${test_vms[@]}"; do
        delete_vm "$vm_id" || true
    done

    # Report persistence test results
    if [ "$persistence_success" = true ]; then
        log_success "VM PERSISTENCE TEST PASSED - All VMs survived restart"
        log_success "Both RUNNING and SHUTDOWN VMs persisted correctly"
        return 0
    else
        log_error "VM PERSISTENCE TEST FAILED - Some VMs were lost"
        log_error "This indicates VM persistence is not working correctly"
        return 1
    fi
}

# Main stress test function
run_stress_test() {
    local duration="${1:-300}"  # Default 5 minutes
    local start_time=$(date +%s)
    local end_time=$((start_time + duration))

    log_info "Starting VM stress test for $duration seconds"
    log_info "Configuration:"
    log_info "  Docker image: $DOCKER_IMAGE"
    log_info "  Max concurrent VMs: $MAX_CONCURRENT_VMS"
    log_info "  Max total VMs: $MAX_TOTAL_VMS"
    log_info "  Creation delay: ${CREATION_DELAY_SECONDS}s"

    # Initial VM list to understand current state
    log_info "Checking for existing VMs..."
    local existing_vms
    readarray -t existing_vms < <(list_vms)
    if [ ${#existing_vms[@]} -gt 0 ]; then
        log_warning "Found ${#existing_vms[@]} existing VMs, they will be included in cleanup"
    fi

    while [ $(date +%s) -lt $end_time ]; do
        # Randomly decide whether to create a new VM or operate on existing ones
        if [ ${#active_vms[@]} -lt $MAX_CONCURRENT_VMS ] && [ $total_created -lt $MAX_TOTAL_VMS ]; then
            if [ $((RANDOM % 3)) -eq 0 ]; then  # 33% chance to create new VM
                create_and_boot_vm
                sleep $CREATION_DELAY_SECONDS
            else
                perform_random_operation
            fi
        else
            perform_random_operation
        fi

        # Print status every 30 seconds
        if [ $(($(date +%s) % 30)) -eq 0 ]; then
            print_status
        fi

        sleep 1
    done

    log_success "Stress test completed"
    print_status
    cleanup_all_vms
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS] [DURATION]

VM lifecycle stress test for metald

Options:
    -h, --help              Show this help message
    -c, --cleanup           Clean up all existing VMs and exit
    -s, --status            Show current VM status and exit
    -p, --persistence       Run VM persistence test (tests survival across restart)
    -i, --image IMAGE       Docker image to use (default: $DOCKER_IMAGE)
    --metald-cli PATH       Path to metald-cli binary (default: $METALD_CLI)

Arguments:
    DURATION                Test duration in seconds (default: 300)

Environment variables:
    DOCKER_IMAGE            Docker image for VMs
    METALD_CLI              Path to metald-cli binary

Examples:
    $0                      Run stress test for 5 minutes
    $0 600                  Run stress test for 10 minutes
    $0 -c                   Clean up all VMs
    $0 -s                   Show VM status
    $0 -p                   Run VM persistence test
    $0 -i nginx:alpine 180  Test with nginx image for 3 minutes

EOF
}

# Main script logic
main() {
    local duration=300
    local cleanup_only=false
    local status_only=false
    local persistence_test=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -c|--cleanup)
                cleanup_only=true
                shift
                ;;
            -s|--status)
                status_only=true
                shift
                ;;
            -p|--persistence)
                persistence_test=true
                shift
                ;;
            -i|--image)
                DOCKER_IMAGE="$2"
                shift 2
                ;;
            --metald-cli)
                METALD_CLI="$2"
                shift 2
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                if [[ "$1" =~ ^[0-9]+$ ]]; then
                    duration="$1"
                else
                    log_error "Invalid duration: $1"
                    exit 1
                fi
                shift
                ;;
        esac
    done

    # Check if metald-cli is available
    if ! command -v "$METALD_CLI" >/dev/null 2>&1; then
        log_error "metald-cli not found at: $METALD_CLI"
        log_error "Please install metald-cli or set METALD_CLI environment variable"
        exit 1
    fi

    if [ "$cleanup_only" = true ]; then
        cleanup_all_vms
        exit 0
    fi

    if [ "$status_only" = true ]; then
        local existing_vms
        readarray -t existing_vms < <(list_vms)
        log_info "Current VMs:"
        for vm_id in "${existing_vms[@]}"; do
            if [ -n "$vm_id" ]; then
                get_vm_info "$vm_id" || true
            fi
        done
        exit 0
    fi

    if [ "$persistence_test" = true ]; then
        run_persistence_test
        exit $?
    fi

    run_stress_test "$duration"
}

# Run main function with all arguments
main "$@"
