#!/bin/bash
set -e

BASE_URL="${1:-http://localhost:8080}"

echo "Testing MCP Private Registry at $BASE_URL"
echo "=========================================="

echo -e "\n1. Health check..."
curl -s "$BASE_URL/health" | jq .

echo -e "\n2. List servers (replace with real org/repo)..."
# curl -s "$BASE_URL/your-org/your-repo/main/v0.1/servers" | jq .

echo -e "\n3. Search servers..."
# curl -s "$BASE_URL/your-org/your-repo/main/v0.1/servers?search=test" | jq .

echo -e "\n4. Get latest version..."
# curl -s "$BASE_URL/your-org/your-repo/main/v0.1/servers/test%2Fserver/versions/latest" | jq .

echo -e "\n5. Test CORS..."
curl -s -X OPTIONS \
  -H "Origin: https://example.com" \
  -H "Access-Control-Request-Method: GET" \
  "$BASE_URL/health" -v 2>&1 | grep -i "access-control"

echo -e "\n6. Test error handling (invalid repo)..."
curl -s "$BASE_URL/invalid/repo/main/v0.1/servers" | jq .

echo -e "\nDone!"
