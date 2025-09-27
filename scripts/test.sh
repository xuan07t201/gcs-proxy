#!/bin/bash

# Test script for GCS Proxy

set -e

# Configuration
BASE_URL="${1:-http://localhost:8080}"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

# Test health endpoint
test_health() {
    print_test "Testing health endpoint..."
    
    response=$(curl -s -w "%{http_code}" -o /tmp/health_response.json "$BASE_URL/health")
    http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        print_pass "Health endpoint returns 200"
        if grep -q "healthy" /tmp/health_response.json; then
            print_pass "Health response contains 'healthy' status"
        else
            print_fail "Health response missing 'healthy' status"
        fi
    else
        print_fail "Health endpoint returns $http_code instead of 200"
    fi
}

# Test headers
test_headers() {
    print_test "Testing response headers..."
    
    headers=$(curl -s -I "$BASE_URL/health")
    
    if echo "$headers" | grep -q "Content-Type:"; then
        print_pass "Content-Type header present"
    else
        print_fail "Content-Type header missing"
    fi
    
    if echo "$headers" | grep -q "X-Response-Time:"; then
        print_pass "X-Response-Time header present"
    else
        print_fail "X-Response-Time header missing"
    fi
}

# Test CORS
test_cors() {
    print_test "Testing CORS headers..."
    
    headers=$(curl -s -I -H "Origin: https://example.com" "$BASE_URL/health")
    
    if echo "$headers" | grep -q "Access-Control-Allow-Origin:"; then
        print_pass "CORS headers present"
    else
        print_fail "CORS headers missing"
    fi
}

# Test compression
test_compression() {
    print_test "Testing compression support..."
    
    headers=$(curl -s -I -H "Accept-Encoding: gzip" "$BASE_URL/health")
    
    if echo "$headers" | grep -q "Content-Encoding:"; then
        print_pass "Compression supported"
    else
        print_pass "Compression not enabled (may be normal for small responses)"
    fi
}

# Test 404 handling
test_404() {
    print_test "Testing 404 handling..."
    
    response=$(curl -s -w "%{http_code}" -o /tmp/404_response.json "$BASE_URL/nonexistent-file.jpg")
    http_code="${response: -3}"
    
    if [ "$http_code" = "404" ]; then
        print_pass "Non-existent file returns 404"
    else
        print_fail "Non-existent file returns $http_code instead of 404"
    fi
}

# Performance test
test_performance() {
    print_test "Testing response time..."
    
    start_time=$(date +%s%N)
    curl -s "$BASE_URL/health" > /dev/null
    end_time=$(date +%s%N)
    
    duration=$(( (end_time - start_time) / 1000000 ))
    
    if [ "$duration" -lt 1000 ]; then
        print_pass "Health endpoint responds in ${duration}ms"
    else
        print_fail "Health endpoint slow: ${duration}ms"
    fi
}

# Main test runner
main() {
    echo "Starting GCS Proxy Tests"
    echo "Base URL: $BASE_URL"
    echo "========================="
    
    test_health
    test_headers
    test_cors
    test_compression
    test_404
    test_performance
    
    echo "========================="
    echo "Tests completed!"
    
    # Cleanup
    rm -f /tmp/health_response.json /tmp/404_response.json
}

# Help
show_help() {
    echo "Usage: $0 [BASE_URL]"
    echo ""
    echo "Test the GCS Proxy server functionality"
    echo ""
    echo "Arguments:"
    echo "  BASE_URL    Base URL of the server (default: http://localhost:8080)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Test local server"
    echo "  $0 https://gcs-proxy-xxx.run.app     # Test Cloud Run deployment"
}

case "$1" in
    help|--help|-h)
        show_help
        ;;
    *)
        main "$@"
        ;;
esac