#!/bin/bash
# RFC 9449 DPoP smoke test (opt-in via DPOP_ENABLED)
# Issues a DPoP-bound client_credentials token (with nonce challenge when
# DPOP_NONCE_REQUIRED=true), checks discovery + introspection, and verifies
# UserInfo rejects a bound token used as Bearer.

set -euo pipefail

BASE_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
CLIENT_ID="${DPOP_TEST_CLIENT_ID:-backend-client}"
CLIENT_SECRET="${DPOP_TEST_CLIENT_SECRET:-backend-client-secret}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
KEY_FILE="${TMPDIR:-/tmp}/dpop-key.$$"
trap 'rm -f "$KEY_FILE"' EXIT

echo "🧪 RFC 9449 DPoP Test"
echo "====================="
echo "Server: $BASE_URL"

# --- Discovery metadata ---
echo ""
echo "📋 Step 1: Check discovery advertises DPoP algorithms..."
DISC=$(curl -s "$BASE_URL/.well-known/oauth-authorization-server")
ALGS=$(echo "$DISC" | jq -r '.dpop_signing_alg_values_supported // empty')
if [ -z "$ALGS" ] || [ "$ALGS" = "null" ]; then
  echo "❌ dpop_signing_alg_values_supported missing — is DPOP_ENABLED=true?"
  echo "$DISC" | jq .
  exit 1
fi
echo "✅ Discovery lists DPoP algs: $(echo "$DISC" | jq -c '.dpop_signing_alg_values_supported')"

NONCE_REQUIRED=$(echo "$DISC" | jq -r '.dpop_nonce_required // false')

# --- Generate DPoP proof (stable key for nonce retry) ---
echo ""
echo "📋 Step 2: Generate DPoP proof for token endpoint..."
HTU="${BASE_URL%/}/token"
PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE")
PROOF=$(echo "$PROOF_OUT" | sed -n '1p')
EXPECTED_JKT=$(echo "$PROOF_OUT" | sed -n '2p')
echo "Expected jkt: $EXPECTED_JKT"

# --- Token request with DPoP (nonce challenge when required) ---
echo ""
echo "📋 Step 3: Request token with DPoP header..."
TOKEN_HEADERS=$(mktemp)
TOKEN_BODY=$(mktemp)
trap 'rm -f "$KEY_FILE" "$TOKEN_HEADERS" "$TOKEN_BODY"' EXIT

HTTP_CODE=$(curl -s -D "$TOKEN_HEADERS" -o "$TOKEN_BODY" -w "%{http_code}" -X POST "$BASE_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "DPoP: $PROOF" \
  -d "grant_type=client_credentials&scope=openid profile")

if [ "$HTTP_CODE" = "400" ] && grep -qi 'use_dpop_nonce' "$TOKEN_BODY"; then
  echo "ℹ️ Server challenged for DPoP nonce (use_dpop_nonce)"
  SERVER_NONCE=$(grep -i '^DPoP-Nonce:' "$TOKEN_HEADERS" | awk '{print $2}' | tr -d '\r')
  if [ -z "$SERVER_NONCE" ]; then
    echo "❌ use_dpop_nonce without DPoP-Nonce header"
    cat "$TOKEN_HEADERS" "$TOKEN_BODY"
    exit 1
  fi
  echo "Retrying with nonce: $SERVER_NONCE"
  PROOF_OUT=$(go run "$ROOT_DIR/scripts/dpop-proof" -htm POST -htu "$HTU" -key "$KEY_FILE" -nonce "$SERVER_NONCE")
  PROOF=$(echo "$PROOF_OUT" | sed -n '1p')
  EXPECTED_JKT=$(echo "$PROOF_OUT" | sed -n '2p')
  HTTP_CODE=$(curl -s -D "$TOKEN_HEADERS" -o "$TOKEN_BODY" -w "%{http_code}" -X POST "$BASE_URL/token" \
    -u "$CLIENT_ID:$CLIENT_SECRET" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "DPoP: $PROOF" \
    -d "grant_type=client_credentials&scope=openid profile")
elif [ "$NONCE_REQUIRED" = "true" ]; then
  echo "❌ Expected use_dpop_nonce challenge when dpop_nonce_required=true"
  cat "$TOKEN_HEADERS" "$TOKEN_BODY"
  exit 1
fi

TOKEN_RESP=$(cat "$TOKEN_BODY")
echo "Token response (HTTP $HTTP_CODE): $TOKEN_RESP"
ACCESS_TOKEN=$(echo "$TOKEN_RESP" | jq -r '.access_token // empty')
TOKEN_TYPE=$(echo "$TOKEN_RESP" | jq -r '.token_type // empty')
RESP_JKT=$(echo "$TOKEN_RESP" | jq -r '.cnf.jkt // empty')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
  echo "❌ Failed to obtain access token"
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
echo "✅ Received DPoP-bound token (jkt matches)"

RESP_NONCE=$(grep -i '^DPoP-Nonce:' "$TOKEN_HEADERS" | awk '{print $2}' | tr -d '\r' || true)
if [ -n "$RESP_NONCE" ]; then
  echo "✅ Success response included DPoP-Nonce for subsequent proofs"
fi

# --- Introspection includes cnf ---
echo ""
echo "📋 Step 4: Introspect token for cnf.jkt..."
INTROSPECT=$(curl -s -X POST "$BASE_URL/introspect" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "token=$ACCESS_TOKEN")
echo "Introspection: $INTROSPECT"
INT_JKT=$(echo "$INTROSPECT" | jq -r '.cnf.jkt // empty')
ACTIVE=$(echo "$INTROSPECT" | jq -r '.active // false')
if [ "$ACTIVE" != "true" ]; then
  echo "❌ Token not active on introspection"
  exit 1
fi
if [ "$INT_JKT" != "$EXPECTED_JKT" ]; then
  echo "❌ Introspection cnf.jkt mismatch"
  exit 1
fi
echo "✅ Introspection includes matching cnf.jkt"

# --- Bearer must fail for DPoP-bound token on userinfo ---
echo ""
echo "📋 Step 5: UserInfo with Bearer must fail for DPoP-bound token..."
HTTP_CODE=$(curl -s -o /tmp/dpop-ui.out -w "%{http_code}" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  "$BASE_URL/userinfo")
if [ "$HTTP_CODE" = "200" ]; then
  echo "❌ UserInfo accepted Bearer for DPoP-bound token"
  cat /tmp/dpop-ui.out
  exit 1
fi
echo "✅ UserInfo rejected Bearer for DPoP-bound token (HTTP $HTTP_CODE)"

# --- Without DPoP, classic Bearer tokens still work ---
echo ""
echo "📋 Step 6: Classic Bearer issuance still works without DPoP header..."
BEARER_RESP=$(curl -s -X POST "$BASE_URL/token" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&scope=openid profile")
BEARER_TYPE=$(echo "$BEARER_RESP" | jq -r '.token_type // empty' | tr '[:upper:]' '[:lower:]')
BEARER_TOKEN=$(echo "$BEARER_RESP" | jq -r '.access_token // empty')
if [ -z "$BEARER_TOKEN" ] || [ "$BEARER_TOKEN" = "null" ]; then
  echo "❌ Bearer token issuance failed"
  echo "$BEARER_RESP"
  exit 1
fi
if [ "$BEARER_TYPE" != "bearer" ]; then
  echo "❌ Expected token_type=bearer without DPoP, got: $BEARER_TYPE"
  exit 1
fi
echo "✅ Non-DPoP clients still receive Bearer tokens"

echo ""
echo "✅ RFC 9449 DPoP test completed successfully"
