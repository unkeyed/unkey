#!/bin/bash
set -e

# Build packages for billaged and metald
# Supports both RPM and DEB package generation

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
BUILD_RPM=false
BUILD_DEB=false
BUILD_ALL=false
CLEAN=false
VERSION="0.1.0"

usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Build RPM and/or Debian packages for billaged and metald"
    echo ""
    echo "Options:"
    echo "  -r, --rpm          Build RPM packages"
    echo "  -d, --deb          Build Debian packages"
    echo "  -a, --all          Build both RPM and Debian packages"
    echo "  -c, --clean        Clean build artifacts before building"
    echo "  -v, --version VER  Set package version (default: $VERSION)"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --rpm                    # Build only RPM packages"
    echo "  $0 --deb                    # Build only Debian packages"
    echo "  $0 --all                    # Build both RPM and DEB packages"
    echo "  $0 --all --clean            # Clean and build all packages"
    echo "  $0 --rpm --version 1.0.0    # Build RPM with specific version"
}

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

check_dependencies() {
    local missing_deps=()
    
    if [[ "$BUILD_RPM" == "true" || "$BUILD_ALL" == "true" ]]; then
        if ! command -v rpmbuild &> /dev/null; then
            missing_deps+=("rpm-build")
        fi
        if ! command -v rpmdev-setuptree &> /dev/null; then
            missing_deps+=("rpmdevtools")
        fi
    fi
    
    if [[ "$BUILD_DEB" == "true" || "$BUILD_ALL" == "true" ]]; then
        if ! command -v debuild &> /dev/null; then
            missing_deps+=("devscripts")
        fi
        if ! command -v dh &> /dev/null; then
            missing_deps+=("debhelper")
        fi
    fi
    
    if [[ ${#missing_deps[@]} -ne 0 ]]; then
        error "Missing dependencies: ${missing_deps[*]}"
    fi
}

clean_build_artifacts() {
    log "Cleaning build artifacts..."
    
    # Clean Go build artifacts
    cd "$PROJECT_ROOT/billaged" && make clean
    cd "$PROJECT_ROOT/metald" && make clean
    
    # Clean packaging artifacts
    rm -rf "$PROJECT_ROOT"/build-packages
    rm -rf "$PROJECT_ROOT"/billaged/build
    rm -rf "$PROJECT_ROOT"/metald/build
    
    log "Build artifacts cleaned"
}

prepare_build() {
    log "Preparing build environment..."
    
    # Create build directory
    mkdir -p "$PROJECT_ROOT/build-packages"
    
    # Setup RPM build tree if needed
    if [[ "$BUILD_RPM" == "true" || "$BUILD_ALL" == "true" ]]; then
        rpmdev-setuptree
    fi
}

create_source_tarball() {
    local service="$1"
    local version="$2"
    
    log "Creating source tarball for $service..."
    
    cd "$PROJECT_ROOT"
    
    # Create temporary directory for source
    local temp_dir="/tmp/${service}-${version}"
    rm -rf "$temp_dir"
    mkdir -p "$temp_dir"
    
    # Special handling for metald - create a workspace with both services
    if [[ "$service" == "metald" ]]; then
        log "Creating workspace with billaged dependency for metald..."
        
        # Create the directory structure that matches the replace directive
        mkdir -p "$temp_dir/metald"
        mkdir -p "$temp_dir/billaged"
        
        # Copy metald source
        rsync -av --exclude='.git' \
                  --exclude='build/' \
                  --exclude='*.rpm' \
                  --exclude='*.deb' \
                  --exclude='debian/tmp' \
                  --exclude='debian/.debhelper' \
                  --exclude='debian/files' \
                  --exclude='debian/*debhelper*' \
                  "metald/" "$temp_dir/metald/"
        
        # Copy billaged source (dependency)
        rsync -av --exclude='.git' \
                  --exclude='build/' \
                  --exclude='*.rpm' \
                  --exclude='*.deb' \
                  "billaged/" "$temp_dir/billaged/"
        
        # Move metald contents to root and adjust paths
        cp -r "$temp_dir/metald/"* "$temp_dir/"
        rm -rf "$temp_dir/metald"
    else
        # Standard copy for billaged
        rsync -av --exclude='.git' \
                  --exclude='build/' \
                  --exclude='*.rpm' \
                  --exclude='*.deb' \
                  --exclude='debian/tmp' \
                  --exclude='debian/.debhelper' \
                  --exclude='debian/files' \
                  --exclude='debian/*debhelper*' \
                  "$service/" "$temp_dir/"
    fi
    
    # Create tarball
    cd "/tmp"
    tar -czf "${HOME}/rpmbuild/SOURCES/${service}-${version}.tar.gz" "${service}-${version}"
    
    # Cleanup
    rm -rf "$temp_dir"
    
    log "Source tarball created: ${HOME}/rpmbuild/SOURCES/${service}-${version}.tar.gz"
}

build_rpm() {
    local service="$1"
    local version="$2"
    
    log "Building RPM package for $service..."
    
    # Create source tarball
    create_source_tarball "$service" "$version"
    
    # Build RPM
    cd "$PROJECT_ROOT/$service"
    rpmbuild -ba "${service}.spec" \
        --define "_version $version" \
        --define "_release 1%{?dist}"
    
    # Copy RPM to build directory
    mkdir -p "$PROJECT_ROOT/build-packages/rpm"
    cp "${HOME}/rpmbuild/RPMS/x86_64/${service}-${version}-"*.rpm "$PROJECT_ROOT/build-packages/rpm/"
    cp "${HOME}/rpmbuild/SRPMS/${service}-${version}-"*.src.rpm "$PROJECT_ROOT/build-packages/rpm/"
    
    log "RPM package built: $PROJECT_ROOT/build-packages/rpm/${service}-${version}-*.rpm"
}

build_deb() {
    local service="$1"
    local version="$2"
    
    log "Building Debian package for $service..."
    
    cd "$PROJECT_ROOT/$service"
    
    # Update changelog with current version
    if [[ -f debian/changelog ]]; then
        # Create backup
        cp debian/changelog debian/changelog.bak
        
        # Update version in changelog
        sed -i "1s/([^)]*)/($version-1)/" debian/changelog
    fi
    
    # Build package
    debuild -us -uc -b
    
    # Restore original changelog
    if [[ -f debian/changelog.bak ]]; then
        mv debian/changelog.bak debian/changelog
    fi
    
    # Copy DEB to build directory
    mkdir -p "$PROJECT_ROOT/build-packages/deb"
    cp "../${service}_${version}-"*.deb "$PROJECT_ROOT/build-packages/deb/" 2>/dev/null || \
    cp "../${service}_${version}"*.deb "$PROJECT_ROOT/build-packages/deb/" 2>/dev/null || \
    warn "Could not find built .deb file for $service"
    
    log "Debian package built: $PROJECT_ROOT/build-packages/deb/${service}_${version}*.deb"
}

build_service_packages() {
    local service="$1"
    
    log "Building packages for $service..."
    
    # Ensure service directory exists
    if [[ ! -d "$PROJECT_ROOT/$service" ]]; then
        error "Service directory not found: $PROJECT_ROOT/$service"
    fi
    
    # Special handling for metald - enable replace directive temporarily
    if [[ "$service" == "metald" ]]; then
        cd "$PROJECT_ROOT/$service"
        log "Enabling billaged replace directive for metald build..."
        sed -i 's|//replace github.com/unkeyed/unkey/go/deploy/billaged|replace github.com/unkeyed/unkey/go/deploy/billaged|g' go.mod
        
        # Build Go binary
        make build
        
        # Restore commented replace directive
        sed -i 's|replace github.com/unkeyed/unkey/go/deploy/billaged|//replace github.com/unkeyed/unkey/go/deploy/billaged|g' go.mod
    else
        # Build Go binary first
        cd "$PROJECT_ROOT/$service"
        make build
    fi
    
    # Build RPM if requested
    if [[ "$BUILD_RPM" == "true" || "$BUILD_ALL" == "true" ]]; then
        build_rpm "$service" "$VERSION"
    fi
    
    # Build DEB if requested
    if [[ "$BUILD_DEB" == "true" || "$BUILD_ALL" == "true" ]]; then
        build_deb "$service" "$VERSION"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--rpm)
            BUILD_RPM=true
            shift
            ;;
        -d|--deb)
            BUILD_DEB=true
            shift
            ;;
        -a|--all)
            BUILD_ALL=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Validate arguments
if [[ "$BUILD_RPM" == "false" && "$BUILD_DEB" == "false" && "$BUILD_ALL" == "false" ]]; then
    error "Must specify at least one of: --rpm, --deb, or --all"
fi

# Main execution
log "Starting package build process..."
log "Version: $VERSION"
log "Build RPM: $([ "$BUILD_RPM" == "true" ] || [ "$BUILD_ALL" == "true" ] && echo "yes" || echo "no")"
log "Build DEB: $([ "$BUILD_DEB" == "true" ] || [ "$BUILD_ALL" == "true" ] && echo "yes" || echo "no")"

# Check dependencies
check_dependencies

# Clean if requested
if [[ "$CLEAN" == "true" ]]; then
    clean_build_artifacts
fi

# Prepare build environment
prepare_build

# Build packages for both services
build_service_packages "billaged"
build_service_packages "metald"

log "Package build complete!"
log "Packages are available in: $PROJECT_ROOT/build-packages/"

if [[ "$BUILD_RPM" == "true" || "$BUILD_ALL" == "true" ]]; then
    log "RPM packages:"
    ls -la "$PROJECT_ROOT/build-packages/rpm/" 2>/dev/null || warn "No RPM packages found"
fi

if [[ "$BUILD_DEB" == "true" || "$BUILD_ALL" == "true" ]]; then
    log "Debian packages:"
    ls -la "$PROJECT_ROOT/build-packages/deb/" 2>/dev/null || warn "No Debian packages found"
fi