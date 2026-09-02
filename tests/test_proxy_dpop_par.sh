#!/bin/bash
# Proxy PAR (RFC 9126) + DPoP key binding (RFC 9449 §10.1) with DPOP_REQUIRED=true.
# Verifies dpop_jkt pushed via PAR survives request_uri → authorize → token:
#  1) PAR with dpop_jkt
#  2) Authorize using request_uri
#  3) Token with matching proof succeeds
#  4) Token with mismatched proof fails

set -euo pipefail

OAUTH2_SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
MOCK_PROVIDER_URL="${MOCK_PROVIDER_URL:-http://localhost:9999}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"
TEST_REDIRECT_URI="http://127.0.0.1:34567/callback"
TEST_SCOPE="openid profile email"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEY_FILE="${TMPDIR:-/tmp}/dpop-proxy-par-key.$$"
trap 'rm -f "$KEY_FILE"' EXIT

echo "🧪 Proxy PAR + DPoP Binding (DPOP_REQUIRED) Test"
echo "================================================="
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
PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE" 2>/dev/null)
EXPECTED_JKT=$(echo "$PROOF_OUT" | sed -n '2p' | tr -d '[:space:]')
if [ -z "$EXPECTED_JKT" ]; then
  echo "❌ Failed to generate DPoP proof/jkt (is Go available?)"
  echo "$PROOF_OUT"
  exit 1
fi
echo "DPoP jkt: $EXPECTED_JKT"

echo ""
echo "📋 Step 2: Register client..."
CLIENT_ID="dpop-par-$(date +%s)-$RANDOM"
CLIENT_SECRET="dpop-par-secret-$(date +%s)-$RANDOM"
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

# PAR → authorize(request_uri) → code
par_then_code() {
  local state="dpop-par-state-$(date +%s)-$RANDOM"
  local par_resp
  par_resp=$(curl -s -X POST "$OAUTH2_SERVER_URL/authorize" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    --data-urlencode "client_id=${CLIENT_ID}" \
    --data-urlencode "response_type=code" \
    --data-urlencode "scope=openid profile email" \
    --data-urlencode "redirect_uri=${TEST_REDIRECT_URI}" \
    --data-urlencode "state=${state}" \
    --data-urlencode "dpop_jkt=${EXPECTED_JKT}")
  local request_uri
  request_uri=$(echo "$par_resp" | jq -r '.request_uri // empty')
  if [ -z "$request_uri" ] || [ "$request_uri" = "null" ]; then
    echo "❌ PAR failed: $par_resp" >&2
    return 1
  fi
  echo "   PAR request_uri: $request_uri" >&2

  local encoded_uri
  encoded_uri=$(printf '%s' "$request_uri" | jq -sRr @uri)
  local auth_url="${OAUTH2_SERVER_URL}/authorize?request_uri=${encoded_uri}&client_id=${CLIENT_ID}"
  local out
  # Prefer GNU timeout; fall back to curl --max-time only
  if command -v timeout >/dev/null 2>&1; then
    out=$(timeout 15 curl -s -L --connect-timeout 3 --max-time 5 \
      -w "FINAL_URL:%{url_effective}\n" \
      "$auth_url" 2>/dev/null || true)
  else
    out=$(curl -s -L --connect-timeout 3 --max-time 15 \
      -w "FINAL_URL:%{url_effective}\n" \
      "$auth_url" 2>/dev/null || true)
  fi
  local final
  final=$(echo "$out" | grep "FINAL_URL:" | sed 's/FINAL_URL://')
  if ! echo "$final" | grep -q "code="; then
    echo "❌ No authorization code after PAR authorize: $final" >&2
    return 1
  fi
  echo "$final" | sed 's/.*code=\([^&]*\).*/\1/'
}

echo ""
echo "📋 Step 3: PAR with dpop_jkt → authorize → matching DPoP token..."
AUTH_CODE=$(par_then_code)
echo "   Authorization code: ${AUTH_CODE:0:24}..."

PROOF=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE" 2>/dev/null | sed -n '1p')
TOKEN_RESP=$(curl -s -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF" \
  -d "grant_type=authorization_code&code=$AUTH_CODE&redirect_uri=$TEST_REDIRECT_URI")
if echo "$TOKEN_RESP" | jq -e . >/dev/null 2>&1; then
  echo "   Token response: $(echo "$TOKEN_RESP" | jq -c '{token_type, cnf}')"
else
  echo "   Token response (non-JSON): $TOKEN_RESP"
fi
ACCESS_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty' 2>/dev/null || true)
TOKEN_TYPE=$(echo "$TOKEN_RESP" | jq -r '.token_type // empty' 2>/dev/null || true)
RESP_JKT=$(echo "$TOKEN_RESP" | jq -r '.cnf.jkt // empty' 2>/dev/null || true)

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
  echo "❌ Token exchange with PAR-bound DPoP failed"
  echo "$TOKEN_RESP"
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
echo "✅ PAR dpop_jkt binding honored at token endpoint"

echo ""
echo "📋 Step 4: PAR-bound code + mismatched DPoP key must fail..."
AUTH_CODE2=$(par_then_code)
WRONG_KEY="${TMPDIR:-/tmp}/dpop-proxy-par-wrong.$$"
PROOF_WRONG=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$WRONG_KEY" 2>/dev/null | sed -n '1p')
rm -f "$WRONG_KEY"
MIS_BODY=$(mktemp)
MIS_CODE=$(curl -s -o "$MIS_BODY" -w "%{http_code}" -X POST "$OAUTH2_SERVER_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF_WRONG" \
  -d "grant_type=authorization_code&code=$AUTH_CODE2&redirect_uri=$TEST_REDIRECT_URI")
echo "   HTTP $MIS_CODE: $(cat "$MIS_BODY")"
if [ "$MIS_CODE" = "200" ]; then
  echo "❌ Accepted mismatched key for PAR-bound authorization"
  echo "   Hint: ensure the server was started with DPOP_ENABLED=true and includes PAR→callback dpop_jkt binding (rebuild with make build)."
  exit 1
fi
if ! grep -Eqi 'invalid_grant|dpop_jkt|binding|DPoP' "$MIS_BODY"; then
  echo "❌ Expected binding error, got: $(cat "$MIS_BODY")"
  exit 1
fi
echo "✅ PAR binding rejected mismatched DPoP key"
rm -f "$MIS_BODY"

echo ""
echo "✅ Proxy PAR + DPoP binding test completed successfully"
