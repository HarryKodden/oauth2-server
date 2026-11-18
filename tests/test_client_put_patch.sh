#!/bin/bash
# Test PUT (full replace) and PATCH (selective update) for /clients/{id}

set -euo pipefail

SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"
CLIENT_ID="put-patch-test-client"

echo "🧪 Client PUT/PATCH Test"
echo "========================"
echo "Server: $SERVER_URL"

echo "📝 Registering client..."
REG=$(curl -s -X POST "$SERVER_URL/register" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"client_name\": \"PUT PATCH Test\",
    \"client_secret\": \"original-secret-value-123456\",
    \"redirect_uris\": [\"https://example.com/cb1\"],
    \"grant_types\": [\"authorization_code\", \"refresh_token\"],
    \"response_types\": [\"code\"],
    \"scope\": \"openid profile\",
    \"token_endpoint_auth_method\": \"client_secret_basic\"
  }")
echo "Registration: $REG"
echo "$REG" | grep -q "\"client_id\":\"$CLIENT_ID\"" || { echo "❌ register failed"; exit 1; }

echo "🔧 PATCH redirect_uris only..."
PATCH=$(curl -s -X PATCH "$SERVER_URL/clients/$CLIENT_ID" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "redirect_uris": ["https://example.com/cb1", "https://example.com/cb2"]
  }')
echo "PATCH response: $PATCH"
echo "$PATCH" | grep -q 'https://example.com/cb2' || { echo "❌ PATCH did not update redirect_uris"; exit 1; }
echo "$PATCH" | grep -q 'authorization_code' || { echo "❌ PATCH cleared grant_types"; exit 1; }

echo "🔁 Verifying PATCH persisted (secret unchanged, redirects updated)..."
GET1=$(curl -s -X GET "$SERVER_URL/clients/$CLIENT_ID" -H "X-API-Key: $API_KEY")
echo "GET after PATCH: $GET1"
echo "$GET1" | grep -q 'https://example.com/cb2' || { echo "❌ GET missing patched redirect"; exit 1; }
echo "$GET1" | grep -q 'original-secret-value-123456' || { echo "❌ secret changed after PATCH"; exit 1; }

echo "📦 PUT full replace with new secret..."
PUT=$(curl -s -X PUT "$SERVER_URL/clients/$CLIENT_ID" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"client_name\": \"Fully Replaced Client\",
    \"client_secret\": \"rotated-secret-value-654321\",
    \"redirect_uris\": [\"https://example.com/only\"],
    \"grant_types\": [\"client_credentials\"],
    \"response_types\": [],
    \"scope\": \"openid\",
    \"token_endpoint_auth_method\": \"client_secret_post\"
  }")
echo "PUT response: $PUT"
echo "$PUT" | grep -q 'https://example.com/only' || { echo "❌ PUT redirect missing"; exit 1; }
echo "$PUT" | grep -q 'client_credentials' || { echo "❌ PUT grant_types missing"; exit 1; }
echo "$PUT" | grep -q 'rotated-secret-value-654321' || { echo "❌ PUT did not return rotated secret"; exit 1; }
echo "$PUT" | grep -q 'https://example.com/cb2' && { echo "❌ PUT did not replace old redirects"; exit 1; }

echo "🔐 Token with rotated secret..."
TOKEN2=$(curl -s -X POST "$SERVER_URL/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=rotated-secret-value-654321&scope=openid")
echo "Token response: $TOKEN2"
echo "$TOKEN2" | grep -q '"access_token"' || { echo "❌ token with rotated secret failed"; exit 1; }

echo "🚫 Old secret should fail..."
TOKEN_OLD=$(curl -s -X POST "$SERVER_URL/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=original-secret-value-123456&scope=openid")
echo "Old secret response: $TOKEN_OLD"
echo "$TOKEN_OLD" | grep -q '"access_token"' && { echo "❌ old secret still works"; exit 1; }

echo "🧹 Cleanup..."
curl -s -X DELETE "$SERVER_URL/clients/$CLIENT_ID" -H "X-API-Key: $API_KEY" >/dev/null

echo "✅ Client PUT/PATCH tests passed"
