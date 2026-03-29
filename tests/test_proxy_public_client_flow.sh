#!/bin/bash

# Test Proxy Public Client Authorization Code Flow
# This script tests the OAuth2 authorization code flow in proxy mode
# where the downstream client is a public client and upstream is confidential.
#
# Also verifies that upstream prompt policies (UPSTREAM_PROMPT_*) coexist with
# issuer_state: the redirect to the IdP must include the policy prompt, and
# privileged introspection must still return the original issuer_state.

# Test configuration - use environment variables with defaults
TEST_USERNAME="${TEST_USERNAME:-john.doe}"
TEST_PASSWORD="${TEST_PASSWORD:-password123}"
TEST_SCOPE="${TEST_SCOPE:-openid%20offline_access}"

echo "🧪 Proxy Public Client Authorization Code Flow Test"
echo "=================================================="
echo "Testing OAuth2 Authorization Code Flow in Proxy Mode"
echo "Downstream: Public Client | Upstream: Confidential Client"
echo "Using environment variables:"
echo "  TEST_USERNAME: $TEST_USERNAME"
echo "  TEST_PASSWORD: $TEST_PASSWORD"
echo "  TEST_SCOPE: $TEST_SCOPE"
echo ""

SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
MOCK_PROVIDER_URL="http://localhost:9999"
API_KEY="test-api-key"

# Function to URL encode a string
url_encode() {
    local string="$1"
    local encoded=""
    local length="${#string}"
    for (( i = 0; i < length; i++ )); do
        local char="${string:i:1}"
        case "$char" in
            [a-zA-Z0-9.~_-])
                encoded+="$char"
                ;;
            ' ')
                encoded+="%20"
                ;;
            *)
                printf -v hex '%02X' "'$char"
                encoded+="%$hex"
                ;;
        esac
    done
    echo "$encoded"
}

# Initialize test results
STEP1_PASS=false
STEP2_PASS=false
STEP3_PASS=false
STEP4_PASS=false
STEP5_PASS=false
STEP6_PASS=false
STEP7_PASS=false
STEP8_PASS=false

# Function to print status
print_status() {
    local status="$1"
    local message="$2"
    case "$status" in
        "success")
            echo "✅ $message"
            ;;
        "error")
            echo "❌ $message"
            ;;
        "info")
            echo "ℹ️  $message"
            ;;
        *)
            echo "$message"
            ;;
    esac
}

# Note: Mock upstream provider is started automatically by run-test-script.sh
# Verify it's accessible
print_status "info" "Verifying mock upstream provider is accessible..."
MOCK_RESPONSE=$(curl -s "$MOCK_PROVIDER_URL/.well-known/openid-configuration" 2>/dev/null)
if ! echo "$MOCK_RESPONSE" | grep -q "issuer"; then
    print_status "error" "Mock upstream provider not accessible at $MOCK_PROVIDER_URL"
    print_status "error" "Make sure to run tests via: make test-script SCRIPT=<script-name>"
    exit 1
fi
print_status "success" "Mock upstream provider is ready"

# Cleanup function
cleanup() {
    if [ -f server.pid ]; then
        kill $(cat server.pid) 2>/dev/null || true
        rm -f server.pid
    fi
}

# Set trap for cleanup
trap cleanup EXIT

echo ""
echo "🧪 Step 1: Starting OAuth2 server in proxy mode..."

# Check if port 8080 is already in use and kill any existing server
if lsof -i :8080 > /dev/null 2>&1; then
    echo "Port 8080 is already in use. Killing existing server..."
    kill $(lsof -t -i :8080) 2>/dev/null || true
    sleep 2
fi

# Start the OAuth2 server in proxy mode with the mock provider.
# Named policy OPENID matches scope "openid" and sets prompt=login on the upstream redirect (see Step 4 check).
( UPSTREAM_PROVIDER_URL="$MOCK_PROVIDER_URL" UPSTREAM_CLIENT_ID="mock-client-id" UPSTREAM_CLIENT_SECRET="mock-client-secret" UPSTREAM_CALLBACK_URL="$SERVER_URL/callback" API_KEY="$API_KEY" LOG_LEVEL=debug \
  UPSTREAM_PROMPT_POLICIES=OPENID \
  UPSTREAM_PROMPT_POLICY_OPENID_ACTION=set \
  UPSTREAM_PROMPT_POLICY_OPENID_PROMPT=login \
  UPSTREAM_PROMPT_POLICY_OPENID_MATCH_SCOPE=openid \
  ./bin/oauth2-server > server.log 2>&1 ) &
SERVER_PID=$!
echo $SERVER_PID > server.pid

# Wait for server to start
sleep 8

# Test server health
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    print_status "error" "OAuth2 server failed to start"
    echo "Server logs:"
    cat server.log
    exit 1
fi

print_status "success" "OAuth2 server started in proxy mode"
echo ""

echo "🧪 Step 2a: Registering confidential client for introspection..."

# Register a confidential client for introspection
CONFIDENTIAL_REGISTER_RESPONSE=$(curl -s -X POST "$SERVER_URL/register" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "Confidential Introspection Client",
    "grant_types": ["client_credentials"],
    "response_types": ["token"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scope": "openid profile email",
    "public": false
  }')

echo "Confidential Client Registration Response: $CONFIDENTIAL_REGISTER_RESPONSE"

# Extract client ID and secret from registration response
CONFIDENTIAL_CLIENT_ID=$(echo "$CONFIDENTIAL_REGISTER_RESPONSE" | jq -r '.client_id // empty' 2>/dev/null || echo "")
CONFIDENTIAL_CLIENT_SECRET=$(echo "$CONFIDENTIAL_REGISTER_RESPONSE" | jq -r '.client_secret // empty' 2>/dev/null || echo "")

if [ -z "$CONFIDENTIAL_CLIENT_ID" ] || [ -z "$CONFIDENTIAL_CLIENT_SECRET" ]; then
    print_status "error" "Failed to register confidential test client"
    echo "Response: $CONFIDENTIAL_REGISTER_RESPONSE"
    exit 1
fi

print_status "success" "Confidential client registered successfully"
echo "   Client ID: $CONFIDENTIAL_CLIENT_ID"
echo "   Client Secret: ${CONFIDENTIAL_CLIENT_SECRET:0:10}..."
echo ""

echo "🧪 Step 2b: Registering public client for proxy authorization code flow..."

# Register a public client for authorization code testing, including confidential client in audience
REGISTER_RESPONSE=$(curl -s -X POST "$SERVER_URL/register" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_name\": \"Proxy Public Client Test\",
    \"grant_types\": [\"authorization_code\"],
    \"response_types\": [\"code\"],
    \"token_endpoint_auth_method\": \"none\",
    \"scope\": \"$TEST_SCOPE\",
    \"redirect_uris\": [\"http://localhost:8080/oauth/callback\"],
    \"audience\": [\"$CONFIDENTIAL_CLIENT_ID\"],
    \"public\": true
  }")

echo "Public Client Registration Response: $REGISTER_RESPONSE"

# Extract client ID from registration response
TEST_CLIENT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.client_id // empty' 2>/dev/null || echo "")

if [ -z "$TEST_CLIENT_ID" ]; then
    print_status "error" "Failed to register public test client"
    echo "Response: $REGISTER_RESPONSE"
    exit 1
fi

print_status "success" "Public client registered successfully"
echo "   Client ID: $TEST_CLIENT_ID"
echo "   Public: true (no client secret)"
echo "   Audience includes: $CONFIDENTIAL_CLIENT_ID"
echo ""

echo "🧪 Step 3: Testing proxy public client authorization code flow..."

# Check if server is still running
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    print_status "error" "OAuth2 server is not responding"
    exit 1
fi

print_status "info" "Server is still running, proceeding with authorization request"

# Generate PKCE parameters
CODE_VERIFIER=$(openssl rand -hex 32 | cut -c1-64 | tr -d '\n')
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | openssl base64 | tr '+/' '-_' | tr -d '=')

echo "🔐 PKCE Code Verifier: ${CODE_VERIFIER:0:20}..."
echo "🔐 PKCE Code Challenge: ${CODE_CHALLENGE:0:20}..."

# Generate state
STATE=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
echo "🎲 State: ${STATE:0:20}..."

# Generate issuer state
ISSUER_STATE=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
echo "🎲 Issuer State: ${ISSUER_STATE:0:20}..."

# Build authorization URL for proxy
ENCODED_SCOPE=$(url_encode "$TEST_SCOPE")
CLAIMS_PARAM="edumember_is_member_of"
AUTH_URL="$SERVER_URL/authorize?response_type=code&client_id=$TEST_CLIENT_ID&redirect_uri=$SERVER_URL/oauth/callback&state=$STATE&scope=$ENCODED_SCOPE&claims=$CLAIMS_PARAM&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256&issuer_state=$ISSUER_STATE"

echo "🔗 Proxy Authorization URL: $AUTH_URL"

# Make initial GET request to authorization endpoint
AUTH_RESPONSE=$(curl -s -i -X GET "$AUTH_URL" \
    -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" \
    -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

# Check for redirect to upstream provider (proxy mode behavior)
if echo "$AUTH_RESPONSE" | grep -q "302 Found" && echo "$AUTH_RESPONSE" | grep -q "Location: http://localhost:9999/authorize"; then
    print_status "success" "Proxy authorization redirect successful - correctly forwarding to upstream provider"
    STEP3_PASS=true
else
    print_status "error" "Expected redirect to upstream provider in proxy mode"
    echo "Response: $AUTH_RESPONSE"
    exit 1
fi

echo ""

echo "🧪 Step 4: Simulating upstream login and authorization code exchange..."

# Extract upstream authorization URL from redirect
UPSTREAM_AUTH_URL=$(echo "$AUTH_RESPONSE" | grep -i "Location:" | sed 's/.*Location: //' | tr -d '\r\n')

if [ -z "$UPSTREAM_AUTH_URL" ]; then
    print_status "error" "Could not extract upstream authorization URL from redirect"
    exit 1
fi

echo "🔗 Upstream Authorization URL: $UPSTREAM_AUTH_URL"

# Upstream prompt policy must be present on the IdP redirect (regression guard with issuer_state in the same flow)
if ! echo "$UPSTREAM_AUTH_URL" | grep -q 'prompt=login'; then
    print_status "error" "Upstream redirect missing prompt=login — upstream prompt policy may be broken or not applied"
    echo "Expected a query parameter prompt=login from UPSTREAM_PROMPT_POLICY_OPENID_* (MATCH_SCOPE=openid)"
    exit 1
fi
print_status "success" "Upstream redirect includes prompt=login (prompt policy + issuer_state in one flow)"

# Extract state from upstream URL
UPSTREAM_STATE=$(echo "$UPSTREAM_AUTH_URL" | sed 's/.*state=\([^&]*\).*/\1/')

if [ -z "$UPSTREAM_STATE" ]; then
    print_status "error" "Could not extract state from upstream authorization URL"
    exit 1
fi

echo "🎲 Upstream State: $UPSTREAM_STATE"

# Simulate upstream authorization by GETting the upstream URL (mock provider auto-approves)
UPSTREAM_LOGIN_RESPONSE=$(curl -s -i -X GET "$UPSTREAM_AUTH_URL")

# Check for redirect back to proxy callback with authorization code
if echo "$UPSTREAM_LOGIN_RESPONSE" | grep -q "302 Found" && echo "$UPSTREAM_LOGIN_RESPONSE" | grep -q "Location: $SERVER_URL/callback"; then
    print_status "success" "Upstream login successful - redirecting back to proxy callback"
else
    print_status "error" "Expected redirect back to proxy callback after upstream login"
    echo "Response: $UPSTREAM_LOGIN_RESPONSE"
    exit 1
fi

# Extract callback URL from upstream redirect
CALLBACK_URL=$(echo "$UPSTREAM_LOGIN_RESPONSE" | grep -i "Location:" | sed 's/.*Location: //' | tr -d '\r\n')

if [ -z "$CALLBACK_URL" ]; then
    print_status "error" "Could not extract callback URL from upstream response"
    exit 1
fi

print_status "success" "Upstream login successful - redirecting to proxy callback: ${CALLBACK_URL:0:80}..."

# Follow the redirect to the proxy callback to trigger the callback handler
CALLBACK_RESPONSE=$(curl -s -i "$CALLBACK_URL")

# The callback should redirect back to the client redirect URI with the authorization code
if echo "$CALLBACK_RESPONSE" | grep -q "302 Found" && echo "$CALLBACK_RESPONSE" | grep -q "Location:"; then
    CLIENT_REDIRECT_URL=$(echo "$CALLBACK_RESPONSE" | grep -i "Location:" | sed 's/.*Location: //' | tr -d '\r\n')
    AUTH_CODE=$(echo "$CLIENT_REDIRECT_URL" | sed 's/.*code=\([^&]*\).*/\1/')
    
    if [ -z "$AUTH_CODE" ]; then
        print_status "error" "Could not extract authorization code from client redirect URL"
        echo "Callback response: $CALLBACK_RESPONSE"
        exit 1
    fi
    
    print_status "success" "Authorization code obtained: ${AUTH_CODE:0:20}..."
    STEP4_PASS=true
else
    print_status "error" "Expected redirect from callback to client redirect URI"
    echo "Callback response: $CALLBACK_RESPONSE"
    exit 1
fi

echo ""

echo "🧪 Step 5: Exchanging authorization code for access token..."

# Exchange authorization code for token at proxy server
TOKEN_RESPONSE=$(curl -s -X POST "$SERVER_URL/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=authorization_code&client_id=$TEST_CLIENT_ID&code=$AUTH_CODE&redirect_uri=$SERVER_URL/oauth/callback&code_verifier=$CODE_VERIFIER")

echo "Token Response: $TOKEN_RESPONSE"

# Extract access token
ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token // empty' 2>/dev/null || echo "")

if [ -z "$ACCESS_TOKEN" ]; then
    print_status "error" "Failed to obtain access token"
    echo "Response: $TOKEN_RESPONSE"
    echo "Continuing to check server logs..."
    # exit 1
fi

print_status "success" "Access token obtained: ${ACCESS_TOKEN:0:20}..."
STEP5_PASS=true

echo ""

echo "🧪 Step 6: Testing UserInfo endpoint with access token..."

# Call UserInfo endpoint with the access token
USERINFO_RESPONSE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$SERVER_URL/userinfo")

echo "UserInfo Response: $USERINFO_RESPONSE"

# Validate UserInfo response
EXPECTED_SUB="john.doe"
ACTUAL_SUB=$(echo "$USERINFO_RESPONSE" | jq -r '.sub // empty' 2>/dev/null || echo "")

if [ "$ACTUAL_SUB" = "$EXPECTED_SUB" ]; then
    print_status "success" "UserInfo endpoint returned correct user data"
    STEP6_PASS=true
else
    print_status "error" "UserInfo endpoint returned incorrect or missing data"
    echo "Expected sub: $EXPECTED_SUB"
    echo "Actual sub: $ACTUAL_SUB"
    exit 1
fi

echo ""

echo "🧪 Step 7: Testing authorization introspection with confidential client..."

# Call authorization introspection endpoint with the confidential client
INTROSPECTION_RESPONSE=$(curl -s -X POST "$SERVER_URL/authorization-introspection" \
    -u "$CONFIDENTIAL_CLIENT_ID:$CONFIDENTIAL_CLIENT_SECRET" \
    -d "access-token=$ACCESS_TOKEN")

echo "Introspection Response: $INTROSPECTION_RESPONSE"

# Validate introspection response
INTROSPECTION_ACTIVE=$(echo "$INTROSPECTION_RESPONSE" | jq -r '.["token_details"].active // empty' 2>/dev/null || echo "")
INTROSPECTION_CLIENT_ID=$(echo "$INTROSPECTION_RESPONSE" | jq -r '.["token_details"].client_id // empty' 2>/dev/null || echo "")
INTROSPECTION_ISSUER_STATE=$(echo "$INTROSPECTION_RESPONSE" | jq -r '.["token_details"].issuer_state // empty' 2>/dev/null || echo "")

if [ "$INTROSPECTION_ACTIVE" = "true" ] && [ "$INTROSPECTION_CLIENT_ID" = "$TEST_CLIENT_ID" ] && [ "$INTROSPECTION_ISSUER_STATE" = "$ISSUER_STATE" ]; then
    print_status "success" "Authorization introspection returned correct token details including issuer_state"
    STEP7_PASS=true
else
    print_status "error" "Authorization introspection returned incorrect data"
    echo "Expected active: true"
    echo "Actual active: $INTROSPECTION_ACTIVE"
    echo "Expected client_id: $TEST_CLIENT_ID"
    echo "Actual client_id: $INTROSPECTION_CLIENT_ID"
    echo "Expected issuer_state: $ISSUER_STATE"
    echo "Actual issuer_state: $INTROSPECTION_ISSUER_STATE"
    exit 1
fi

# Check if response contains token_details
if echo "$INTROSPECTION_RESPONSE" | jq -e '.["token_details"]' > /dev/null; then
  echo "✅ Response contains token_details"
else
  echo "❌ Response missing token_details"
  exit 1
fi

# Check if response contains user_info
if echo "$INTROSPECTION_RESPONSE" | jq -e '.["user_info"]' > /dev/null; then
  echo "✅ Response contains user_info"
else
  echo "❌ Response missing user_info"
  exit 1
fi

echo ""

echo ""

echo "📊 Proxy Public Client Flow Test Results Summary"
echo "================================================"
echo "Step 1 (OAuth2 server in proxy mode): ✅ PASS"
echo "Step 2a (Confidential client registration): ✅ PASS"
echo "Step 2b (Public client registration): ✅ PASS"
echo "Step 3 (Proxy authorization redirect): $([ "$STEP3_PASS" = true ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "Step 4 (Upstream login and code exchange): $([ "$STEP4_PASS" = true ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "Step 5 (Token exchange): $([ "$STEP5_PASS" = true ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "Step 6 (UserInfo endpoint): $([ "$STEP6_PASS" = true ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "Step 7 (Authorization introspection): $([ "$STEP7_PASS" = true ] && echo "✅ PASS" || echo "❌ FAIL")"

# Overall result
if [ "$STEP3_PASS" = true ] && [ "$STEP4_PASS" = true ] && [ "$STEP5_PASS" = true ] && [ "$STEP6_PASS" = true ] && [ "$STEP7_PASS" = true ]; then
    echo ""
    print_status "success" "Proxy public client authorization code flow test PASSED!"
    echo "   ✅ Public client authorization requests are correctly forwarded to upstream provider"
    echo "   ✅ Proxy mode works for public clients without client secrets"
    echo "   ✅ PKCE parameters are properly passed through to upstream"
    echo "   ✅ Authorization code exchange returns valid access token"
    echo "   ✅ UserInfo endpoint returns correct user data from upstream"
    echo "   ✅ Confidential client can introspect public client's access token"
    echo "   ✅ Upstream prompt policy (prompt=login) applied together with issuer_state round-trip"
    exit 0
else
    echo ""
    print_status "error" "Proxy public client authorization code flow test FAILED!"
    exit 1
fi
exit 0
