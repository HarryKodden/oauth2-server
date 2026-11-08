#!/bin/bash
# RFC 7523 private_key_jwt client authentication test

set -euo pipefail

SERVER_URL="${OAUTH2_SERVER_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-super-secure-random-api-key-change-in-production-32-chars-minimum}"
TEST_SCOPE="${TEST_SCOPE:-openid profile}"
CLIENT_ID="rfc7523-pkjwt-client"
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

echo "🧪 RFC 7523 private_key_jwt Test"
echo "================================"
echo "Server: $SERVER_URL"

python3 - <<'PY' "$WORKDIR"
import json, sys, time, uuid
from pathlib import Path
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.backends import default_backend
import base64

workdir = Path(sys.argv[1])
key = ec.generate_private_key(ec.SECP256R1(), default_backend())
priv_pem = key.private_bytes(
    encoding=serialization.Encoding.PEM,
    format=serialization.PrivateFormat.PKCS8,
    encryption_algorithm=serialization.NoEncryption(),
)
(workdir / "ec-private.pem").write_bytes(priv_pem)

pub = key.public_key().public_numbers()
def b64u(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()

def int_to_b64u(n: int, length: int) -> str:
    return b64u(n.to_bytes(length, "big"))

jwks = {
    "keys": [{
        "kty": "EC",
        "crv": "P-256",
        "x": int_to_b64u(pub.x, 32),
        "y": int_to_b64u(pub.y, 32),
        "kid": "rfc7523-test-key",
        "use": "sig",
        "alg": "ES256",
    }]
}
(workdir / "jwks.json").write_text(json.dumps(jwks))
print("ok")
PY

JWKS=$(cat "$WORKDIR/jwks.json")

echo "📝 Registering private_key_jwt client..."
REG=$(curl -s -X POST "$SERVER_URL/register" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"client_id\": \"$CLIENT_ID\",
    \"client_name\": \"RFC7523 private_key_jwt client\",
    \"grant_types\": [\"client_credentials\"],
    \"response_types\": [],
    \"token_endpoint_auth_method\": \"private_key_jwt\",
    \"token_endpoint_auth_signing_alg\": \"ES256\",
    \"scope\": \"$TEST_SCOPE\",
    \"redirect_uris\": [],
    \"jwks\": $JWKS
  }")

echo "Registration: $REG"
echo "$REG" | grep -q "\"client_id\":\"$CLIENT_ID\"" || { echo "❌ registration failed"; exit 1; }
echo "$REG" | grep -q "private_key_jwt" || { echo "❌ auth method missing"; exit 1; }

TOKEN_URL="$SERVER_URL/token"
NOW=$(date +%s)
EXP=$((NOW + 300))
JTI=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)

python3 - <<'PY' "$WORKDIR" "$CLIENT_ID" "$TOKEN_URL" "$NOW" "$EXP" "$JTI"
import base64, hashlib, json, sys
from pathlib import Path
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec, utils
from cryptography.hazmat.backends import default_backend

workdir, client_id, aud, iat, exp, jti = sys.argv[1:7]
priv = serialization.load_pem_private_key((Path(workdir)/"ec-private.pem").read_bytes(), password=None, backend=default_backend())

def b64u(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()

header = {"alg": "ES256", "typ": "JWT", "kid": "rfc7523-test-key"}
payload = {"iss": client_id, "sub": client_id, "aud": aud, "iat": int(iat), "exp": int(exp), "jti": jti}
signing_input = f"{b64u(json.dumps(header, separators=(',', ':')).encode())}.{b64u(json.dumps(payload, separators=(',', ':')).encode())}".encode()
sig = priv.sign(signing_input, ec.ECDSA(hashes.SHA256()))
r, s = utils.decode_dss_signature(sig)
sig_b64 = b64u(r.to_bytes(32, "big") + s.to_bytes(32, "big"))
token = signing_input.decode() + "." + sig_b64
(Path(workdir)/"assertion.jwt").write_text(token)
print(token[:40] + "...")
PY

ASSERTION=$(cat "$WORKDIR/assertion.jwt")
ENCODED=$(printf '%s' "$ASSERTION" | jq -sRr @uri)

echo "🔐 Requesting token with private_key_jwt..."
TOKEN_RESP=$(curl -s -X POST "$TOKEN_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&scope=$TEST_SCOPE&client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer&client_assertion=$ENCODED")

echo "Token response: $TOKEN_RESP"
echo "$TOKEN_RESP" | grep -q '"access_token"' || { echo "❌ token request failed"; exit 1; }

echo "🔁 Replaying same assertion (should fail)..."
REPLAY=$(curl -s -X POST "$TOKEN_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&scope=$TEST_SCOPE&client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer&client_assertion=$ENCODED")
echo "Replay response: $REPLAY"
if echo "$REPLAY" | grep -q '"access_token"'; then
  echo "❌ replay was accepted"
  exit 1
fi

echo "✅ RFC 7523 private_key_jwt tests passed"
