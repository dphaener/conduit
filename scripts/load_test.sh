#!/bin/bash
#
# Load testing script for Conduit web framework
# Uses various load testing tools to measure performance
#
# Requirements:
#   - vegeta (https://github.com/tsenart/vegeta)
#   - ab (Apache Bench - usually pre-installed)
#   - wrk (https://github.com/wg/wrk) - optional
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
RATE="${RATE:-10000}"
WORKERS="${WORKERS:-100}"

# Check for required dependencies
check_dependencies() {
    local missing_deps=()
    local optional_deps=()

    # Required dependencies
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi

    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi

    if ! command -v bc &> /dev/null; then
        missing_deps+=("bc")
    fi

    # Optional but recommended dependencies
    if ! command -v vegeta &> /dev/null; then
        optional_deps+=("vegeta")
    fi

    if ! command -v ab &> /dev/null; then
        optional_deps+=("ab (Apache Bench)")
    fi

    # Report missing required dependencies
    if [ ${#missing_deps[@]} -ne 0 ]; then
        echo -e "${RED}Error: Missing required dependencies:${NC}"
        for dep in "${missing_deps[@]}"; do
            echo -e "  - $dep"
        done
        echo ""
        echo "Please install the missing dependencies:"
        echo "  macOS:   brew install ${missing_deps[*]}"
        echo "  Ubuntu:  sudo apt-get install ${missing_deps[*]}"
        echo "  RHEL:    sudo yum install ${missing_deps[*]}"
        exit 1
    fi

    # Report missing optional dependencies
    if [ ${#optional_deps[@]} -ne 0 ]; then
        echo -e "${YELLOW}Warning: Missing optional dependencies:${NC}"
        for dep in "${optional_deps[@]}"; do
            echo -e "  - $dep"
        done
        echo ""
        echo "Install optional tools for more comprehensive testing:"
        echo "  vegeta:  go install github.com/tsenart/vegeta@latest"
        echo "  ab:      (usually pre-installed or: brew install httpd / apt-get install apache2-utils)"
        echo "  wrk:     brew install wrk / apt-get install wrk"
        echo ""
    fi
}

echo -e "${GREEN}=== Conduit Load Testing ===${NC}"
echo ""

# Check dependencies first
check_dependencies

echo "Server: $SERVER_URL"
echo "Duration: $DURATION"
echo "Target Rate: $RATE req/s"
echo ""

# Check if server is running
echo -e "${YELLOW}Checking if server is running...${NC}"
if ! curl -s -f "$SERVER_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}Error: Server is not running at $SERVER_URL${NC}"
    echo "Please start the server first"
    exit 1
fi
echo -e "${GREEN}Server is running${NC}"
echo ""

# Function to run vegeta load test
run_vegeta() {
    if ! command -v vegeta &> /dev/null; then
        echo -e "${YELLOW}Vegeta not installed, skipping...${NC}"
        return
    fi

    echo -e "${GREEN}=== Running Vegeta Load Test ===${NC}"
    echo "GET $SERVER_URL/api/posts" | \
        vegeta attack -duration=$DURATION -rate=$RATE -workers=$WORKERS | \
        vegeta report -type=text

    echo ""
    echo -e "${GREEN}Generating Vegeta plot...${NC}"
    echo "GET $SERVER_URL/api/posts" | \
        vegeta attack -duration=$DURATION -rate=$RATE -workers=$WORKERS | \
        vegeta plot > /tmp/vegeta_plot.html
    echo -e "Plot saved to: /tmp/vegeta_plot.html"
    echo ""
}

# Function to run Apache Bench test
run_ab() {
    if ! command -v ab &> /dev/null; then
        echo -e "${YELLOW}Apache Bench not installed, skipping...${NC}"
        return
    fi

    echo -e "${GREEN}=== Running Apache Bench Test ===${NC}"
    ab -n 100000 -c 100 -k "$SERVER_URL/"
    echo ""
}

# Function to run wrk test
run_wrk() {
    if ! command -v wrk &> /dev/null; then
        echo -e "${YELLOW}wrk not installed, skipping...${NC}"
        return
    fi

    echo -e "${GREEN}=== Running wrk Load Test ===${NC}"
    wrk -t12 -c400 -d30s "$SERVER_URL/"
    echo ""
}

# Function to test different endpoints
test_endpoints() {
    echo -e "${GREEN}=== Testing Individual Endpoints ===${NC}"

    # Test homepage
    echo -e "${YELLOW}Testing GET /${NC}"
    echo "GET $SERVER_URL/" | vegeta attack -duration=10s -rate=5000 | vegeta report -type=text
    echo ""

    # Test API endpoint
    echo -e "${YELLOW}Testing GET /api/posts${NC}"
    echo "GET $SERVER_URL/api/posts" | vegeta attack -duration=10s -rate=5000 | vegeta report -type=text
    echo ""

    # Test POST endpoint
    echo -e "${YELLOW}Testing POST /api/posts${NC}"
    echo "POST $SERVER_URL/api/posts" | \
        vegeta attack -duration=10s -rate=1000 -body=/tmp/post_body.json | \
        vegeta report -type=text
    echo ""
}

# Function to create test data
create_test_data() {
    echo '{"title":"Test Post","content":"This is a test post content"}' > /tmp/post_body.json
}

# Function to run concurrent user simulation
run_concurrent_test() {
    echo -e "${GREEN}=== Running Concurrent User Test ===${NC}"
    echo "Simulating 1000 concurrent users..."

    for i in {1..1000}; do
        curl -s "$SERVER_URL/" > /dev/null &
    done
    wait

    echo -e "${GREEN}Completed${NC}"
    echo ""
}

# Function to test keep-alive connections
test_keepalive() {
    echo -e "${GREEN}=== Testing Keep-Alive Connections ===${NC}"

    # Test with keep-alive
    echo -e "${YELLOW}With Keep-Alive:${NC}"
    ab -n 10000 -c 100 -k "$SERVER_URL/" | grep "Requests per second"

    echo ""
    echo -e "${YELLOW}Without Keep-Alive:${NC}"
    ab -n 10000 -c 100 "$SERVER_URL/" | grep "Requests per second"

    echo ""
}

# Function to profile during load test
profile_during_load() {
    echo -e "${GREEN}=== Profiling During Load Test ===${NC}"

    # Start load test in background
    echo "GET $SERVER_URL/" | vegeta attack -duration=60s -rate=5000 > /tmp/results.bin &
    VEGETA_PID=$!

    sleep 5  # Let load build up

    # Capture CPU profile
    echo -e "${YELLOW}Capturing CPU profile...${NC}"
    curl -s "$SERVER_URL/debug/pprof/profile?seconds=30" > /tmp/cpu.prof

    # Wait for load test to finish
    wait $VEGETA_PID

    # Generate report
    vegeta report /tmp/results.bin

    echo -e "${GREEN}CPU profile saved to: /tmp/cpu.prof${NC}"
    echo "Analyze with: go tool pprof /tmp/cpu.prof"
    echo ""
}

# Function to check performance targets
check_targets() {
    echo -e "${GREEN}=== Checking Performance Targets ===${NC}"

    # Run quick test
    RESULTS=$(echo "GET $SERVER_URL/" | vegeta attack -duration=30s -rate=30000 | vegeta report -type=json)

    # Extract metrics
    P50=$(echo $RESULTS | jq -r '.latencies.p50 / 1000000')  # Convert to ms
    P95=$(echo $RESULTS | jq -r '.latencies.p95 / 1000000')
    P99=$(echo $RESULTS | jq -r '.latencies.p99 / 1000000')
    SUCCESS_RATE=$(echo $RESULTS | jq -r '.success_ratio * 100')

    echo "Latency P50: ${P50}ms (target: <5ms)"
    echo "Latency P95: ${P95}ms (target: <10ms)"
    echo "Latency P99: ${P99}ms (target: <50ms)"
    echo "Success Rate: ${SUCCESS_RATE}% (target: >99%)"
    echo ""

    # Check if targets are met
    if (( $(echo "$P95 < 10" | bc -l) )); then
        echo -e "${GREEN}✓ P95 latency target met${NC}"
    else
        echo -e "${RED}✗ P95 latency target not met${NC}"
    fi
}

# Main execution
main() {
    create_test_data

    # Run tests based on arguments
    case "${1:-all}" in
        vegeta)
            run_vegeta
            ;;
        ab)
            run_ab
            ;;
        wrk)
            run_wrk
            ;;
        endpoints)
            test_endpoints
            ;;
        concurrent)
            run_concurrent_test
            ;;
        keepalive)
            test_keepalive
            ;;
        profile)
            profile_during_load
            ;;
        targets)
            check_targets
            ;;
        all)
            run_vegeta
            run_ab
            test_keepalive
            check_targets
            ;;
        *)
            echo "Usage: $0 {vegeta|ab|wrk|endpoints|concurrent|keepalive|profile|targets|all}"
            echo ""
            echo "Options:"
            echo "  vegeta      - Run Vegeta load test"
            echo "  ab          - Run Apache Bench test"
            echo "  wrk         - Run wrk load test"
            echo "  endpoints   - Test individual endpoints"
            echo "  concurrent  - Simulate concurrent users"
            echo "  keepalive   - Test keep-alive performance"
            echo "  profile     - Profile during load test"
            echo "  targets     - Check if performance targets are met"
            echo "  all         - Run all tests (default)"
            echo ""
            echo "Environment variables:"
            echo "  SERVER_URL  - Server URL (default: http://localhost:8080)"
            echo "  DURATION    - Test duration (default: 30s)"
            echo "  RATE        - Request rate (default: 10000)"
            echo "  WORKERS     - Number of workers (default: 100)"
            exit 1
            ;;
    esac
}

main "$@"
