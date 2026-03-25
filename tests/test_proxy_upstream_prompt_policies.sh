#!/bin/bash

# Test: Upstream prompt policies (proxy mode)
# Validates that the server applies prompt policies on the redirect to the upstream provider.
#
# We test:
# - Policy match by scope => prompt=login
# - Policy match by authorization_details => prompt=login and authorization_details forwarded
# - No match => prompt is not set

set -e

SERVER_URL="${SERVER_URL:-http://localhost:8080}"
MOCK_PROVIDER_URL="${MOCK_PROVIDER_URL:-http://localhost:9999}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"

echo "🧪 Proxy Mode Upstream Prompt Policies Test"
echo "=========================================="
echo "Server: $SERVER_URL"
echo "Mock Provider: $MOCK_PROVIDER_URL"
echo ""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
  local status=$1
  local message=$2
  if [ "$status" = "success" ]; then
    echo -e "${GREEN}✅ $message${NC}"
  elif [ "$status" = "error" ]; then
    echo -e "${RED}❌ $message${NC}"
  elif [ "$status" = "info" ]; then
    echo -e "${YELLOW}ℹ️  $message${NC}"
  else
    echo "$message"
  fi
}

extract_location() {
  # Reads raw curl -i response from stdin and prints the first Location header value.
  grep -i "^location:" | head -1 | sed 's/^[Ll]ocation:[[:space:]]*//' | tr -d '\r'
}

get_query_param() {
  local url="$1"
  local key="$2"
  python3 - "$url" "$key" <<'PY'
import sys
from urllib.parse import urlparse, parse_qs

u = sys.argv[1]
k = sys.argv[2]
qs = parse_qs(urlparse(u).query, keep_blank_values=True)
vals = qs.get(k, [])
print(vals[0] if vals else "")
PY
}

has_query_param() {
  local url="$1"
  local key="$2"
  python3 - "$url" "$key" <<'PY'
import sys
from urllib.parse import urlparse, parse_qs

u = sys.argv[1]
k = sys.argv[2]
qs = parse_qs(urlparse(u).query, keep_blank_values=True)
print("yes" if k in qs else "no")
PY
}

cleanup() {
  if [ -f server.pid ]; then
    kill "$(cat server.pid)" 2>/dev/null || true
    rm -f server.pid
  fi
}
trap cleanup EXIT

print_status "info" "Verifying mock upstream provider is accessible..."
MOCK_RESPONSE=$(curl -s "$MOCK_PROVIDER_URL/.well-known/openid-configuration" 2>/dev/null || true)
if ! echo "$MOCK_RESPONSE" | grep -q "issuer"; then
  print_status "error" "Mock upstream provider not accessible at $MOCK_PROVIDER_URL"
  print_status "error" "Run via: make test-script SCRIPT=test_proxy_upstream_prompt_policies.sh"
  exit 1
fi
print_status "success" "Mock upstream provider is ready"

# Start OAuth2 server in proxy mode if not running
if ! curl -s "$SERVER_URL/health" > /dev/null; then
  print_status "info" "Starting OAuth2 server in proxy mode..."

  UPSTREAM_PROVIDER_URL="$MOCK_PROVIDER_URL" \
  UPSTREAM_CLIENT_ID="mock-client-id" \
  UPSTREAM_CLIENT_SECRET="mock-client-secret" \
  UPSTREAM_CALLBACK_URL="$SERVER_URL/callback" \
  API_KEY="$API_KEY" \
  LOG_LEVEL=debug \
  UPSTREAM_PROMPT_POLICIES="EDUID_SCOPE,EDUID_AUTHZ_DETAILS" \
  UPSTREAM_PROMPT_POLICY_EDUID_SCOPE_ACTION="set" \
  UPSTREAM_PROMPT_POLICY_EDUID_SCOPE_PROMPT="login" \
  UPSTREAM_PROMPT_POLICY_EDUID_SCOPE_MATCH_SCOPE="eduID" \
  UPSTREAM_PROMPT_POLICY_EDUID_AUTHZ_DETAILS_ACTION="set" \
  UPSTREAM_PROMPT_POLICY_EDUID_AUTHZ_DETAILS_PROMPT="login" \
  UPSTREAM_PROMPT_POLICY_EDUID_AUTHZ_DETAILS_MATCH_AUTHZ_DETAILS_TYPE="openid_credential" \
  UPSTREAM_PROMPT_POLICY_EDUID_AUTHZ_DETAILS_MATCH_CREDENTIAL_CONFIGURATION_ID="eduID" \
  ./bin/oauth2-server > server.log 2>&1 &

  SERVER_PID=$!
  echo $SERVER_PID > server.pid

  for i in {1..12}; do
    if curl -s "$SERVER_URL/health" > /dev/null; then
      break
    fi
    sleep 1
  done

  if ! curl -s "$SERVER_URL/health" > /dev/null; then
    print_status "error" "Server failed to start"
    echo "Server logs:"
    cat server.log
    exit 1
  fi
fi

print_status "success" "OAuth2 server is ready"

print_status "info" "Registering test client (allows eduID scope)..."
CLIENT_RESPONSE=$(curl -s -X POST "$SERVER_URL/register" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"client_name\": \"Prompt Policy Test Client\",
    \"grant_types\": [\"authorization_code\"],
    \"response_types\": [\"code\"],
    \"redirect_uris\": [\"${SERVER_URL}/callback\"],
    \"scope\": \"openid eduID\"
  }")

CLIENT_ID=$(echo "$CLIENT_RESPONSE" | grep -o '"client_id":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
if [ -z "$CLIENT_ID" ]; then
  print_status "error" "Failed to register test client"
  echo "Response: $CLIENT_RESPONSE"
  exit 1
fi
print_status "success" "Client registered: $CLIENT_ID"

STATE="state-$(date +%s)"

print_status "info" "Case 1: scope contains eduID => expect prompt=login"
RESP1=$(curl -s -i -G "$SERVER_URL/authorize" \
  --data-urlencode "response_type=code" \
  --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "redirect_uri=${SERVER_URL}/callback" \
  --data-urlencode "scope=openid eduID" \
  --data-urlencode "state=$STATE")

LOC1=$(echo "$RESP1" | extract_location)
if [ -z "$LOC1" ]; then
  print_status "error" "No Location header on authorize response"
  echo "$RESP1" | head -40
  exit 1
fi
P1=$(get_query_param "$LOC1" "prompt")
if [ "$P1" != "login" ]; then
  print_status "error" "Expected prompt=login but got prompt='$P1'"
  echo "Redirect: $LOC1"
  exit 1
fi
print_status "success" "prompt=login applied (scope match)"

print_status "info" "Case 2: authorization_details match => expect prompt=login and authorization_details forwarded"
AUTHZ_DETAILS='[{"type":"openid_credential","credential_configuration_id":"eduID"}]'
RESP2=$(curl -s -i -G "$SERVER_URL/authorize" \
  --data-urlencode "response_type=code" \
  --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "redirect_uri=${SERVER_URL}/callback" \
  --data-urlencode "scope=openid" \
  --data-urlencode "authorization_details=$AUTHZ_DETAILS" \
  --data-urlencode "state=$STATE")

LOC2=$(echo "$RESP2" | extract_location)
if [ -z "$LOC2" ]; then
  print_status "error" "No Location header on authorize response"
  echo "$RESP2" | head -40
  exit 1
fi
P2=$(get_query_param "$LOC2" "prompt")
if [ "$P2" != "login" ]; then
  print_status "error" "Expected prompt=login but got prompt='$P2'"
  echo "Redirect: $LOC2"
  exit 1
fi
AD2=$(has_query_param "$LOC2" "authorization_details")
if [ "$AD2" != "yes" ]; then
  print_status "error" "Expected authorization_details to be forwarded to upstream"
  echo "Redirect: $LOC2"
  exit 1
fi
print_status "success" "prompt=login applied and authorization_details forwarded"

print_status "info" "Case 3: no match => expect no prompt parameter"
RESP3=$(curl -s -i -G "$SERVER_URL/authorize" \
  --data-urlencode "response_type=code" \
  --data-urlencode "client_id=$CLIENT_ID" \
  --data-urlencode "redirect_uri=${SERVER_URL}/callback" \
  --data-urlencode "scope=openid" \
  --data-urlencode "state=$STATE")

LOC3=$(echo "$RESP3" | extract_location)
if [ -z "$LOC3" ]; then
  print_status "error" "No Location header on authorize response"
  echo "$RESP3" | head -40
  exit 1
fi
P3_PRESENT=$(has_query_param "$LOC3" "prompt")
if [ "$P3_PRESENT" = "yes" ]; then
  P3=$(get_query_param "$LOC3" "prompt")
  print_status "error" "Expected no prompt parameter, but got prompt='$P3'"
  echo "Redirect: $LOC3"
  exit 1
fi
print_status "success" "No prompt set when no policy matches"

echo ""
print_status "success" "All upstream prompt policy checks passed"

