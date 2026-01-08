#!/bin/bash

# Script to run a series of uptime tests with different configurations
# Starts docker compose, runs tests, then stops docker compose

set -e  # Exit on any error

# Test configuration
TEST_REPETITIONS=4
TEST_DURATION="10m"
TEST_FILESIZE=10024
WAIT_BETWEEN_TESTS=300  # 5 minutes in seconds

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to start docker compose
start_docker_compose() {
    local backend_type=$1
    local gogc=$2
    local gomemlimit=$3
    
    print_status "Starting Docker Compose with BACKEND_TYPE=$backend_type, GOGC=$gogc, GOMEMLIMIT=$gomemlimit"
    
    # Set environment variables and start in detached mode
    BACKEND_TYPE="$backend_type" GOGC="$gogc" GOMEMLIMIT="$gomemlimit" docker-compose up -d
    
    # Wait for services to be ready
    print_status "Waiting for services to start..."
    #sleep 30
    
    # Check if server is responding
    local attempts=0
    local max_attempts=10
    while [ $attempts -lt $max_attempts ]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1 || curl -s http://localhost:8080/ > /dev/null 2>&1; then
            print_success "Server is ready"
            return 0
        fi
        attempts=$((attempts + 1))
        print_status "Waiting for server... (attempt $attempts/$max_attempts)"
        sleep 10
    done
    
    print_error "Server failed to start after $max_attempts attempts"
    return 1
}

# Function to stop docker compose
stop_docker_compose() {
    print_status "Stopping Docker Compose"
    docker-compose down
    print_success "Docker Compose stopped"
}

# Function to run uptime test
run_uptime_test() {
    local test_name=$1
    local backend_type=$2
    
    print_status "Running uptime test: $test_name"
    print_status "Command: go run ./cmd/uptime_test -repetitions=$TEST_REPETITIONS -duration=$TEST_DURATION -filesize=$TEST_FILESIZE"
    
    # Run the test and capture output
    if go run ./cmd/uptime_test -repetitions="$TEST_REPETITIONS" -duration="$TEST_DURATION" -filesize="$TEST_FILESIZE"; then
        print_success "Test '$test_name' completed successfully"
    else
        print_error "Test '$test_name' failed"
        return 1
    fi
}

# Main execution
main() {
    print_status "Waiting 5 minutes before starting tests..."
    sleep 300

    print_status "Starting sequential uptime tests"
    print_status "Configuration: $TEST_REPETITIONS repetitions, $TEST_DURATION duration, $TEST_FILESIZE filesize"
    
    # Test 0: Git backend with default memory settings
    print_status "=== TEST 0: Git Backend (Default Memory Settings) ==="
    start_docker_compose "git" "" ""
    run_uptime_test "Git Backend (Default Memory Settings)" "git"
    stop_docker_compose
    
    print_status "=== GAP BETWEEN TESTS ==="
    print_status "Waiting 5 minutes before next test run..."
    sleep 300
    
    # Test 1: Git backend with GOGC=50 and GOMEMLIMIT=200MiB
    print_status "=== TEST 1: Git Backend ==="
    start_docker_compose "git" "50" "200MiB"
    run_uptime_test "Git Backend (GOGC=50, GOMEMLIMIT=200MiB)" "git"
    stop_docker_compose
    
    print_status "=== GAP BETWEEN TESTS ==="
    print_status "Waiting 5 minutes before next test run..."
    sleep 300
    
    # Test 2: S3 backend with default memory settings
    print_status "=== TEST 2: S3 Backend (Default Memory Settings) ==="
    start_docker_compose "s3" "" ""
    run_uptime_test "S3 Backend (Default Memory Settings)" "s3"
    stop_docker_compose
    
    print_success "All tests completed successfully!"
}

# Handle script interruption
trap 'print_warning "Script interrupted. Cleaning up..."; stop_docker_compose; exit 1' INT TERM

# Run main function
main "$@"
