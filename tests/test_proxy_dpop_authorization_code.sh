#!/bin/bash
# Proxy authorization_code flow with RFC 9449 DPoP required (DPOP_REQUIRED=true).
# Verifies:
#  1) Token exchange without DPoP is rejected
#  2) Authorize binding via dpop_jkt + matching token-endpoint proof succeeds
#  3) Issued token is DPoP-bound (token_type / cnf.jkt)
#  4) UserInfo rejects Bearer and accepts Authorization: DPoP + ath

set -euo pipefail

OAUTH2_SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
MOCK_PROVIDER_URL="${MOCK_PROVIDER_URL:-http://localhost:9999}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"
TEST_REDIRECT_URI="http://127.0.0.1:34567/callback"
TEST_SCOPE="openid profile email"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEY_FILE="${TMPDIR:-/tmp}/dpop-proxy-key.$$"
trap 'rm -f "$KEY_FILE"' EXIT

echo "🧪 Proxy Authorization Code + DPoP (DPOP_REQUIRED) Test"
echo "========================================================"
echo "Server: $OAUTH2_SERVER_URL"

# --- Preconditions ---
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
  echo "$DISC" | jq .
  exit 1
fi
echo "✅ Discovery lists DPoP algs"

# Stable DPoP key + jkt (for authorize binding and token proofs)
HTU="${OAUTH2_SERVER_URL%/}/token"
PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE")
EXPECTED_JKT=$(echo "$PROOF_OUT" | sed -n '2p')
echo "DPoP jkt: $EXPECTED_JKT"

# --- Register client ---
echo ""
echo "📋 Step 2: Register client..."
CLIENT_ID="dpop-proxy-$(date +%s)"
CLIENT_SECRET="dpop-proxy-secret-$(date +%s)"
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

# Helper: run proxy authorize (± dpop_jkt) and return code from final redirect URL
obtain_auth_code() {
  local with_jkt="$1"
  local state="dpop-state-$(date +%s)-$RANDOM"
  local nonce="dpop-nonce-$(date +%s)-$RANDOM"
  local encoded_scope
  encoded_scope=$(echo "$TEST_SCOPE" | sed 's/ /%20/g')
  local auth_url="${OAUTH2_SERVER_URL}/authorize?client_id=${CLIENT_ID}&redirect_uri=${TEST_REDIRECT_URI}&response_type=code&scope=${encoded_scope}&state=${state}&nonce=${nonce}"
  if [ "$with_jkt" = "yes" ]; then
    auth_url="${auth_url}&dpop_jkt=${EXPECTED_JKT}"
  fi
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

# --- Negative: token without DPoP must fail when DPOP_REQUIRED ---
echo ""
echo "📋 Step 3: Auth-code token exchange without DPoP must fail..."
CODE_NO_DPOP=$(obtain_auth_code "no")
NO_DPOP_BODY=$(mktemp)
NO_DPOP_CODE=$(curl -s -o "$NO_DPOP_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=$CODE_NO_DPOP&redirect_uri=$TEST_REDIRECT_URI")
echo "   HTTP $NO_DPOP_CODE: $(cat "$NO_DPOP_BODY")"
if [ "$NO_DPOP_CODE" = "200" ]; then
  echo "❌ Token endpoint issued tokens without DPoP while DPOP_REQUIRED=true"
  exit 1
fi
if ! grep -Eqi 'invalid_request|DPoP|dpop' "$NO_DPOP_BODY"; then
  echo "❌ Expected DPoP-related error, got: $(cat "$NO_DPOP_BODY")"
  exit 1
fi
echo "✅ Token exchange without DPoP rejected (HTTP $NO_DPOP_CODE)"
rm -f "$NO_DPOP_BODY"

# --- Positive: authorize with dpop_jkt + matching proof ---
echo ""
echo "📋 Step 4: Authorize with dpop_jkt and exchange with matching DPoP proof..."
AUTH_CODE=$(obtain_auth_code "yes")
echo "   Authorization code: ${AUTH_CODE:0:24}..."

PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE")
PROOF=$(echo "$PROOF_OUT" | sed -n '1p')
JKT=$(echo "$PROOF_OUT" | sed -n '2p')
if [ "$JKT" != "$EXPECTED_JKT" ]; then
  echo "❌ Key file jkt drifted: $JKT vs $EXPECTED_JKT"
  exit 1
fi

TOKEN_HEADERS=$(mktemp)
TOKEN_BODY=$(mktemp)
trap 'rm -f "$KEY_FILE" "$TOKEN_HEADERS" "$TOKEN_BODY"' EXIT

HTTP_CODE=$(curl -s -D "$TOKEN_HEADERS" -o "$TOKEN_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF" \
  -d "grant_type=authorization_code&code=$AUTH_CODE&redirect_uri=$TEST_REDIRECT_URI")

TOKEN_RESP=$(cat "$TOKEN_BODY")
echo "   Token response (HTTP $HTTP_CODE): $TOKEN_RESP"
ACCESS_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty')
TOKEN_TYPE=$(echo "$TOKEN_RESP" | jq -r '.token_type // empty')
RESP_JKT=$(echo "$TOKEN_RESP" | jq -r '.cnf.jkt // empty')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
  echo "❌ Failed to obtain access token with DPoP"
  exit 1
fi
if [ "$TOKEN_TYPE" != "DPoP" ] && [ "$TOKEN_TYPE" != "dpop" ]; then
  echo "❌ Expected token_type=DPoP, got: $TOKEN_TYPE"
  exit 1
fi
if [ "$RESP_JKT" != "$EXPECTED_JKT" ]; then
  echo "❌ cnf.jkt mismatch: got $RESP_JKT want $EXPECTED_JKT"
  exit 1
fi
echo "✅ Received DPoP-bound proxy access token"

# --- Binding mismatch: different key must fail ---
echo ""
echo "📋 Step 5: Token proof with different key must fail (binding)..."
AUTH_CODE2=$(obtain_auth_code "yes")
WRONG_KEY="${TMPDIR:-/tmp}/dpop-proxy-wrong.$$"
PROOF_WRONG=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$WRONG_KEY" | sed -n '1p')
rm -f "$WRONG_KEY"
MISMATCH_BODY=$(mktemp)
MISMATCH_CODE=$(curl -s -o "$MISMATCH_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF_WRONG" \
  -d "grant_type=authorization_code&code=$AUTH_CODE2&redirect_uri=$TEST_REDIRECT_URI")
echo "   HTTP $MISMATCH_CODE: $(cat "$MISMATCH_BODY")"
if [ "$MISMATCH_CODE" = "200" ]; then
  echo "❌ Accepted DPoP proof that does not match dpop_jkt binding"
  exit 1
fi
echo "✅ Mismatched DPoP key rejected (HTTP $MISMATCH_CODE)"
rm -f "$MISMATCH_BODY"

# --- UserInfo: Bearer rejected, DPoP+ath accepted ---
echo ""
echo "📋 Step 6: UserInfo requires DPoP for bound token..."
BEARER_CODE=$(curl -s -o /tmp/dpop-proxy-ui.out -w "%{http_code}" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$OAUTH2_SERVER_URL/userinfo")
if [ "$BEARER_CODE" = "200" ]; then
  echo "❌ UserInfo accepted Bearer for DPoP-bound token"
  cat /tmp/dpop-proxy-ui.out
  exit 1
fi
echo "✅ UserInfo rejected Bearer (HTTP $BEARER_CODE)"

ATH=$(python3 - <<PY
import hashlib, base64
tok = """$ACCESS_TOKEN"""
print(base64.urlsafe_b64encode(hashlib.sha256(tok.encode()).digest()).decode().rstrip("="))
PY
)
UI_HTU="${OAUTH2_SERVER_URL%/}/userinfo"
UI_PROOF=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm GET -htu "$UI_HTU" -key "$KEY_FILE" -ath "$ATH" | sed -n '1p')
USERINFO=$(curl -s -w "\nHTTP:%{http_code}" \
  -H "Authorization: DPoP $ACCESS_TOKEN" \
  -H "DPoP: $UI_PROOF" \
  "$OAUTH2_SERVER_URL/userinfo")
UI_HTTP=$(echo "$USERINFO" | sed -n 's/^HTTP://p')
UI_BODY=$(echo "$USERINFO" | sed '/^HTTP:/d')
echo "   UserInfo (HTTP $UI_HTTP): $UI_BODY"
if [ "$UI_HTTP" != "200" ]; then
  echo "❌ UserInfo with DPoP proof failed"
  exit 1
fi
SUB=$(echo "$UI_BODY" | jq -r '.sub // empty')
if [ -z "$SUB" ]; then
  echo "❌ UserInfo missing sub"
  exit 1
fi
echo "✅ UserInfo accepted DPoP-bound access (sub=$SUB)"

echo ""
echo "✅ Proxy authorization_code + DPOP_REQUIRED test completed successfully"
