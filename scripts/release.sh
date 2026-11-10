#!/bin/bash
# scripts/release.sh - Release candidate build and verification script

set -euo pipefail

# Configuration
VERSION="${VERSION:-$(git describe --tags --always --dirty)}"
BUILD_ID="${BUILD_ID:-$(date +%Y%m%d%H%M%S)}"
GIT_SHA="${GIT_SHA:-$(git rev-parse HEAD)}"
GO_VERSION="${GO_VERSION:-1.23}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Directories
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${REPO_ROOT}/build"
DIST_DIR="${BUILD_DIR}/dist"
ARTIFACTS_DIR="${BUILD_DIR}/artifacts"

cleanup() {
    log_info "Cleaning up..."
    # Any cleanup needed
}
trap cleanup EXIT

main() {
    log_info "Starting release build for version: ${VERSION}"
    log_info "Build ID: ${BUILD_ID}"
    log_info "Git SHA: ${GIT_SHA}"

    # Create build directories
    mkdir -p "${DIST_DIR}" "${ARTIFACTS_DIR}"

    # Step 1: Run all tests
    log_info "Step 1/8: Running full test suite..."
    run_tests

    # Step 2: Run security scans
    log_info "Step 2/8: Running security scans..."
    run_security_scans

    # Step 3: Run performance benchmarks
    log_info "Step 3/8: Running performance benchmarks..."
    run_performance_benchmarks

    # Step 4: Run memory budget validation
    log_info "Step 4/8: Running memory budget validation..."
    run_memory_validation

    # Step 5: Build Go binary
    log_info "Step 5/8: Building Go binary..."
    build_go_binary

    # Step 6: Build frontend assets
    log_info "Step 6/8: Building frontend assets..."
    build_frontend

    # Step 7: Create distribution packages
    log_info "Step 7/8: Creating distribution packages..."
    create_packages

    # Step 8: Generate SBOM and checksums
    log_info "Step 8/8: Generating SBOM and checksums..."
    generate_artifacts

    log_success "Release build completed successfully!"
    log_info "Artifacts available in: ${ARTIFACTS_DIR}"
    ls -la "${ARTIFACTS_DIR}"
}

run_tests() {
    cd "${REPO_ROOT}"

    # Unit tests with race detector
    go test -race -count=1 ./... 2>&1 | tee "${BUILD_DIR}/test-results.txt"

    # Check for test failures
    if grep -q "FAIL" "${BUILD_DIR}/test-results.txt"; then
        log_error "Tests failed!"
        exit 1
    fi

    log_success "All tests passed"
}

run_security_scans() {
    cd "${REPO_ROOT}"

    # govulncheck
    log_info "Running govulncheck..."
    if command -v govulncheck &> /dev/null; then
        govulncheck ./... 2>&1 | tee "${BUILD_DIR}/govulncheck.txt"
    else
        log_warn "govulncheck not installed, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./... 2>&1 | tee "${BUILD_DIR}/govulncheck.txt"
    fi

    # gosec
    log_info "Running gosec..."
    if command -v gosec &> /dev/null; then
        gosec ./... 2>&1 | tee "${BUILD_DIR}/gosec.txt"
    else
        log_warn "gosec not installed, installing..."
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec ./... 2>&1 | tee "${BUILD_DIR}/gosec.txt"
    fi

    # gitleaks
    log_info "Running gitleaks..."
    if command -v gitleaks &> /dev/null; then
        gitleaks detect --source . --verbose 2>&1 | tee "${BUILD_DIR}/gitleaks.txt"
    else
        log_warn "gitleaks not installed, installing..."
        go install github.com/zricethezav/gitleaks/v8@latest
        gitleaks detect --source . --verbose 2>&1 | tee "${BUILD_DIR}/gitleaks.txt"
    fi

    # Static analysis
    log_info "Running staticcheck..."
    if command -v staticcheck &> /dev/null; then
        staticcheck ./... 2>&1 | tee "${BUILD_DIR}/staticcheck.txt"
    else
        log_warn "staticcheck not installed, installing..."
        go install honnef.co/go/tools/cmd/staticcheck@latest
        staticcheck ./... 2>&1 | tee "${BUILD_DIR}/staticcheck.txt"
    fi

    log_success "Security scans completed"
}

run_performance_benchmarks() {
    cd "${REPO_ROOT}"

    # Build the application first
    go build -o "${BUILD_DIR}/crm" ./cmd/crm

    # Start the server in background
    "${BUILD_DIR}/crm" &
    SERVER_PID=$!
    sleep 5

    # Run load tests with hey
    if command -v hey &> /dev/null; then
        log_info "Running load tests..."
        hey -n 1000 -c 20 http://localhost:8080/health > "${BUILD_DIR}/bench_health.txt" 2>&1
        hey -n 1000 -c 20 http://localhost:8080/api/v1/companies > "${BUILD_DIR}/bench_companies.txt" 2>&1
        hey -n 1000 -c 20 http://localhost:8080/api/v1/contacts > "${BUILD_DIR}/bench_contacts.txt" 2>&1
        hey -n 500 -c 10 "http://localhost:8080/api/v1/search?q=test" > "${BUILD_DIR}/bench_search.txt" 2>&1
        hey -n 500 -c 10 http://localhost:8080/api/v1/dashboard/metrics > "${BUILD_DIR}/bench_dashboard.txt" 2>&1
    else
        log_warn "hey not installed, skipping load tests"
        go install github.com/rakyll/hey@latest
    fi

    # Stop server
    kill ${SERVER_PID} 2>/dev/null || true
    wait ${SERVER_PID} 2>/dev/null || true

    log_success "Performance benchmarks completed"
}

run_memory_validation() {
    cd "${REPO_ROOT}"

    # Run memory tests
    go test -v -run "TestMemory" ./internal/memtrack/... 2>&1 | tee "${BUILD_DIR}/memory-tests.txt"

    # Check memory budget
    go test -v -run "TestMemoryBudget" ./internal/memtrack/... 2>&1 | tee -a "${BUILD_DIR}/memory-tests.txt"

    log_success "Memory validation completed"
}

build_go_binary() {
    cd "${REPO_ROOT}"

    # Build flags for reproducible builds
    LDFLAGS="-X 'crm.local/internal/config.ServerVersion=${VERSION}' \
             -X 'crm.local/internal/config.ServerBuildID=${BUILD_ID}' \
             -X 'crm.local/internal/config.ServerGitSHA=${GIT_SHA}' \
             -X 'crm.local/internal/config.ServerCommit=${GIT_SHA}' \
             -X 'crm.local/internal/config.ServerEnv=production' \
             -s -w"

    # Build for multiple architectures
    for arch in amd64 arm64; do
        log_info "Building for linux/${arch}..."
        GOOS=linux GOARCH=${arch} CGO_ENABLED=1 \
            go build -ldflags="${LDFLAGS}" \
            -o "${DIST_DIR}/crm-linux-${arch}" ./cmd/crm

        # Verify binary
        file "${DIST_DIR}/crm-linux-${arch}"
    done

    # Also build for current platform
    log_info "Building for current platform..."
    go build -ldflags="${LDFLAGS}" -o "${DIST_DIR}/crm" ./cmd/crm

    log_success "Go binary built successfully"
}

build_frontend() {
    cd "${REPO_ROOT}/web"

    # Install dependencies
    log_info "Installing frontend dependencies..."
    npm ci 2>&1 | tee "${BUILD_DIR}/npm-install.txt"

    # Build production assets
    log_info "Building production frontend..."
    npm run build 2>&1 | tee "${BUILD_DIR}/npm-build.txt"

    # Copy to dist
    cp -r dist/* "${DIST_DIR}/public/"

    log_success "Frontend built successfully"
}

create_packages() {
    cd "${REPO_ROOT}"

    # Create release directory structure
    RELEASE_DIR="${DIST_DIR}/crm-${VERSION}"
    mkdir -p "${RELEASE_DIR}"

    # Copy binary
    cp "${DIST_DIR}/crm" "${RELEASE_DIR}/"

    # Copy frontend assets
    cp -r "${DIST_DIR}/public" "${RELEASE_DIR}/"

    # Copy config template
    if [ -f "config.example.yaml" ]; then
        cp config.example.yaml "${RELEASE_DIR}/"
    fi

    # Copy migrations
    cp -r db/migrations "${RELEASE_DIR}/"

    # Copy documentation
    cp -r *.md "${RELEASE_DIR}/" 2>/dev/null || true

    # Create tar.gz
    log_info "Creating tar.gz package..."
    tar -czf "${ARTIFACTS_DIR}/crm-${VERSION}.tar.gz" -C "${DIST_DIR}" "crm-${VERSION}"

    # Create zip
    log_info "Creating zip package..."
    (cd "${DIST_DIR}" && zip -r "${ARTIFACTS_DIR}/crm-${VERSION}.zip" "crm-${VERSION}")

    # Create Docker images
    log_info "Building Docker images..."
    build_docker_images

    log_success "Packages created successfully"
}

build_docker_images() {
    cd "${REPO_ROOT}"

    # Build multi-arch Docker image
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --tag "crm:${VERSION}" \
        --tag "crm:latest" \
        --load \
        . 2>&1 | tee "${BUILD_DIR}/docker-build.txt"

    # Save Docker images
    docker save "crm:${VERSION}" | gzip > "${ARTIFACTS_DIR}/crm-${VERSION}.docker.tar.gz"
    docker save "crm:latest" | gzip > "${ARTIFACTS_DIR}/crm-latest.docker.tar.gz"
}

generate_artifacts() {
    cd "${REPO_ROOT}"

    # Generate SBOM
    log_info "Generating SBOM..."
    if command -v syft &> /dev/null; then
        syft "crm:${VERSION}" -o spdx-json > "${ARTIFACTS_DIR}/sbom-${VERSION}.spdx.json"
        syft "crm:${VERSION}" -o cyclonedx-json > "${ARTIFACTS_DIR}/sbom-${VERSION}.cyclonedx.json"
    else
        log_warn "syft not installed, skipping SBOM generation"
    fi

    # Generate checksums
    log_info "Generating checksums..."
    (cd "${ARTIFACTS_DIR}" && sha256sum * > "checksums-${VERSION}.txt")

    # Sign artifacts (if cosign available)
    if command -v cosign &> /dev/null && [ -n "${COSIGN_PRIVATE_KEY:-}" ]; then
        log_info "Signing artifacts with cosign..."
        for artifact in "${ARTIFACTS_DIR}"/*; do
            if [[ -f "${artifact}" && ! "${artifact}" =~ \.(sig|pem|crt)$ ]]; then
                cosign sign-blob --yes "${artifact}" --output-signature "${artifact}.sig" --output-certificate "${artifact}.crt"
            fi
        done
    else
        log_warn "cosign not available or COSIGN_PRIVATE_KEY not set, skipping signing"
    fi

    # Create release manifest
    cat > "${ARTIFACTS_DIR}/release-manifest-${VERSION}.json" <<EOF
{
    "version": "${VERSION}",
    "build_id": "${BUILD_ID}",
    "git_sha": "${GIT_SHA}",
    "go_version": "$(go version | awk '{print $3}')",
    "build_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "artifacts": [
EOF

    for artifact in "${ARTIFACTS_DIR}"/*; do
        if [[ -f "${artifact}" ]]; then
            filename=$(basename "${artifact}")
            sha256=$(sha256sum "${artifact}" | awk '{print $1}')
            size=$(stat -c%s "${artifact}")
            cat >> "${ARTIFACTS_DIR}/release-manifest-${VERSION}.json" <<EOF
        {
            "name": "${filename}",
            "sha256": "${sha256}",
            "size_bytes": ${size}
        },
EOF
        fi
    done

    # Remove trailing comma and close JSON
    sed -i '$ s/,$//' "${ARTIFACTS_DIR}/release-manifest-${VERSION}.json"
    echo "    ]" >> "${ARTIFACTS_DIR}/release-manifest-${VERSION}.json"
    echo "}" >> "${ARTIFACTS_DIR}/release-manifest-${VERSION}.json"

    log_success "Artifacts generated successfully"
}

# Run main function
main "$@"