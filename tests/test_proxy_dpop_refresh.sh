#!/bin/bash
# Proxy refresh_token flow with RFC 9449 DPoP (DPOP_REQUIRED=true).
# Verifies DPoP binding is preserved across refresh:
#  1) Initial auth-code mint with dpop_jkt + proof
#  2) Refresh without DPoP is rejected
#  3) Refresh with wrong key is rejected
#  4) Refresh with same key succeeds (token_type=DPoP, same cnf.jkt)

set -euo pipefail

OAUTH2_SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
MOCK_PROVIDER_URL="${MOCK_PROVIDER_URL:-http://localhost:9999}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"
TEST_REDIRECT_URI="http://127.0.0.1:34567/callback"
TEST_SCOPE="openid profile email"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEY_FILE="${TMPDIR:-/tmp}/dpop-proxy-refresh-key.$$"
trap 'rm -f "$KEY_FILE"' EXIT

echo "🧪 Proxy Refresh Token + DPoP (DPOP_REQUIRED) Test"
echo "==================================================="
echo "Server: $OAUTH2_SERVER_URL"

echo ""
echo "📋 Step 1: Check services and DPoP discovery..."
if ! curl -s -f "$MOCK_PROVIDER_URL/health" >/dev/null 2>&1; then
  echo "❌ Mock provider not responding at $MOCK_PROVIDER_URL"
  exit 1
fi
if ! curl -s -f "$OAUTH2_SERVER_URL/health" >/dev/null 2>&1; then
  echo "❌ OAuth2 server not responding at $OAUTH2_SERVER_URL"
  exit 1
fi

DISC=$(curl -s "$OAUTH2_SERVER_URL/.well-known/oauth-authorization-server")
ALGS=$(echo "$DISC" | jq -r '.dpop_signing_alg_values_supported // empty')
if [ -z "$ALGS" ] || [ "$ALGS" = "null" ]; then
  echo "❌ dpop_signing_alg_values_supported missing — is DPOP_ENABLED=true?"
  exit 1
fi
echo "✅ Discovery lists DPoP algs"

HTU="${OAUTH2_SERVER_URL%/}/token"
PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE")
EXPECTED_JKT=$(echo "$PROOF_OUT" | sed -n '2p')
echo "DPoP jkt: $EXPECTED_JKT"

echo ""
echo "📋 Step 2: Register client..."
CLIENT_ID="dpop-refresh-$(date +%s)"
CLIENT_SECRET="dpop-refresh-secret-$(date +%s)"
REGISTER_RESPONSE=$(curl -s -X POST "$OAUTH2_SERVER_URL/register" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"client_secret\": \"$CLIENT_SECRET\",
    \"redirect_uris\": [\"$TEST_REDIRECT_URI\"],
    \"grant_types\": [\"authorization_code\", \"refresh_token\"],
    \"response_types\": [\"code\"],
    \"scope\": \"$TEST_SCOPE\",
    \"token_endpoint_auth_method\": \"client_secret_basic\"
  }")
if ! echo "$REGISTER_RESPONSE" | grep -q "client_id"; then
  echo "❌ Client registration failed: $REGISTER_RESPONSE"
  exit 1
fi
echo "✅ Client registered: $CLIENT_ID"

obtain_auth_code() {
  local state="dpop-rf-state-$(date +%s)-$RANDOM"
  local nonce="dpop-rf-nonce-$(date +%s)-$RANDOM"
  local encoded_scope
  encoded_scope=$(echo "$TEST_SCOPE" | sed 's/ /%20/g')
  local auth_url="${OAUTH2_SERVER_URL}/authorize?client_id=${CLIENT_ID}&redirect_uri=${TEST_REDIRECT_URI}&response_type=code&scope=${encoded_scope}&state=${state}&nonce=${nonce}&dpop_jkt=${EXPECTED_JKT}"
  local out
  out=$(timeout 15 curl -s -L --connect-timeout 3 --max-time 5 \
    -w "FINAL_URL:%{url_effective}\n" \
    "$auth_url" 2>/dev/null || true)
  local final
  final=$(echo "$out" | grep "FINAL_URL:" | sed 's/FINAL_URL://')
  if ! echo "$final" | grep -q "code="; then
    echo "❌ No authorization code in redirect: $final" >&2
    return 1
  fi
  echo "$final" | sed 's/.*code=\([^&]*\).*/\1/'
}

echo ""
echo "📋 Step 3: Mint DPoP-bound tokens via authorization_code..."
AUTH_CODE=$(obtain_auth_code)
PROOF=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE" | sed -n '1p')
TOKEN_RESP=$(curl -s -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF" \
  -d "grant_type=authorization_code&code=$AUTH_CODE&redirect_uri=$TEST_REDIRECT_URI")
echo "   Initial token response: $(echo "$TOKEN_RESP" | jq -c '{token_type, cnf, has_rt: (.refresh_token != null)}')"
ACCESS_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty')
REFRESH_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.refresh_token // empty')
TOKEN_TYPE=$(echo "$TOKEN_RESP" | jq -r '.token_type // empty')
RESP_JKT=$(echo "$TOKEN_RESP" | jq -r '.cnf.jkt // empty')

if [ -z "$ACCESS_TOKEN" ] || [ -z "$REFRESH_TOKEN" ] || [ "$REFRESH_TOKEN" = "null" ]; then
  echo "❌ Failed to obtain access/refresh tokens"
  echo "$TOKEN_RESP"
  exit 1
fi
if [ "$TOKEN_TYPE" != "DPoP" ] && [ "$TOKEN_TYPE" != "dpop" ]; then
  echo "❌ Expected token_type=DPoP, got: $TOKEN_TYPE"
  exit 1
fi
if [ "$RESP_JKT" != "$EXPECTED_JKT" ]; then
  echo "❌ cnf.jkt mismatch on initial mint"
  exit 1
fi
echo "✅ Initial DPoP-bound tokens issued (refresh present)"

echo ""
echo "📋 Step 4: Refresh without DPoP must fail..."
NO_BODY=$(mktemp)
NO_CODE=$(curl -s -o "$NO_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=refresh_token&refresh_token=$REFRESH_TOKEN")
echo "   HTTP $NO_CODE: $(cat "$NO_BODY")"
if [ "$NO_CODE" = "200" ]; then
  echo "❌ Refresh succeeded without DPoP while DPOP_REQUIRED=true"
  exit 1
fi
echo "✅ Refresh without DPoP rejected"
rm -f "$NO_BODY"

echo ""
echo "📋 Step 5: Refresh with mismatched DPoP key must fail..."
WRONG_KEY="${TMPDIR:-/tmp}/dpop-proxy-refresh-wrong.$$"
PROOF_WRONG=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$WRONG_KEY" | sed -n '1p')
rm -f "$WRONG_KEY"
MIS_BODY=$(mktemp)
MIS_CODE=$(curl -s -o "$MIS_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF_WRONG" \
  -d "grant_type=refresh_token&refresh_token=$REFRESH_TOKEN")
echo "   HTTP $MIS_CODE: $(cat "$MIS_BODY")"
if [ "$MIS_CODE" = "200" ]; then
  echo "❌ Refresh accepted mismatched DPoP key"
  exit 1
fi
echo "✅ Refresh with wrong key rejected"
rm -f "$MIS_BODY"

echo ""
echo "📋 Step 6: Refresh with matching DPoP key must succeed..."
PROOF_RF=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE" | sed -n '1p')
RF_RESP=$(curl -s -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF_RF" \
  -d "grant_type=refresh_token&refresh_token=$REFRESH_TOKEN")
echo "   Refresh response: $(echo "$RF_RESP" | jq -c '{token_type, cnf, has_at: (.access_token != null)}')"
NEW_AT=$(echo "$RF_RESP" | jq -r '.access_token // empty')
NEW_TYPE=$(echo "$RF_RESP" | jq -r '.token_type // empty')
NEW_JKT=$(echo "$RF_RESP" | jq -r '.cnf.jkt // empty')
NEW_RT=$(echo "$RF_RESP" | jq -r '.refresh_token // empty')

if [ -z "$NEW_AT" ] || [ "$NEW_AT" = "null" ]; then
  echo "❌ Refresh with matching DPoP failed"
  echo "$RF_RESP"
  exit 1
fi
if [ "$NEW_TYPE" != "DPoP" ] && [ "$NEW_TYPE" != "dpop" ]; then
  echo "❌ Expected refreshed token_type=DPoP, got: $NEW_TYPE"
  exit 1
fi
if [ "$NEW_JKT" != "$EXPECTED_JKT" ]; then
  echo "❌ Refresh did not preserve cnf.jkt (got $NEW_JKT want $EXPECTED_JKT)"
  exit 1
fi
echo "✅ Refresh preserved DPoP binding (jkt matches)"
if [ -n "$NEW_RT" ] && [ "$NEW_RT" != "null" ]; then
  echo "✅ New refresh token issued"
fi

echo ""
echo "✅ Proxy refresh_token + DPOP_REQUIRED test completed successfully"
