package dpop

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
)

func newECProof(t *testing.T, claims map[string]interface{}) (proof string, jkt string, priv *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	jwk := &jose.JSONWebKey{Key: &priv.PublicKey, Algorithm: string(jose.ES256), Use: "sig"}
	jkt, err = Thumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint: %v", err)
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: priv}, &jose.SignerOptions{
		ExtraHeaders: map[jose.HeaderKey]interface{}{
			"typ": JWTType,
			"jwk": jwk,
		},
	})
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	obj, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	serialized, err := obj.CompactSerialize()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return serialized, jkt, priv
}

func TestVerifyValidProof(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	htu := "https://as.example.com/token"
	proof, jkt, _ := newECProof(t, map[string]interface{}{
		"jti": "proof-1",
		"htm": http.MethodPost,
		"htu": htu,
		"iat": time.Now().Unix(),
	})

	got, err := v.Verify(proof, http.MethodPost, htu, "")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.JKT != jkt {
		t.Fatalf("jkt = %s, want %s", got.JKT, jkt)
	}
}

func TestVerifyReplayRejected(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	htu := "https://as.example.com/token"
	proof, _, _ := newECProof(t, map[string]interface{}{
		"jti": "replay-jti",
		"htm": http.MethodPost,
		"htu": htu,
		"iat": time.Now().Unix(),
	})
	if _, err := v.Verify(proof, http.MethodPost, htu, ""); err != nil {
		t.Fatalf("first verify: %v", err)
	}
	if _, err := v.Verify(proof, http.MethodPost, htu, ""); err == nil {
		t.Fatal("expected replay error")
	}
}

func TestVerifyHTMMismatch(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	htu := "https://as.example.com/token"
	proof, _, _ := newECProof(t, map[string]interface{}{
		"jti": "htm-1",
		"htm": http.MethodPost,
		"htu": htu,
		"iat": time.Now().Unix(),
	})
	if _, err := v.Verify(proof, http.MethodGet, htu, ""); err == nil {
		t.Fatal("expected htm mismatch")
	}
}

func TestVerifyATH(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	htu := "https://as.example.com/userinfo"
	token := "access-token-value"
	good, _, _ := newECProof(t, map[string]interface{}{
		"jti": "ath-good",
		"htm": http.MethodGet,
		"htu": htu,
		"iat": time.Now().Unix(),
		"ath": AccessTokenHash(token),
	})
	if _, err := v.Verify(good, http.MethodGet, htu, token); err != nil {
		t.Fatalf("verify with ath: %v", err)
	}
	bad, _, _ := newECProof(t, map[string]interface{}{
		"jti": "ath-bad",
		"htm": http.MethodGet,
		"htu": htu,
		"iat": time.Now().Unix(),
		"ath": AccessTokenHash(token),
	})
	if _, err := v.Verify(bad, http.MethodGet, htu, "other-token"); err == nil {
		t.Fatal("expected ath mismatch for wrong token")
	}
}

func TestVerifyHeaderAndHTTPURI(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	base := "https://as.example.com"
	htu := HTTPURI(base, "/token")
	proof, _, _ := newECProof(t, map[string]interface{}{
		"jti": "hdr-1",
		"htm": http.MethodPost,
		"htu": htu,
		"iat": time.Now().Unix(),
	})
	req := httptest.NewRequest(http.MethodPost, "https://as.example.com/token?x=1", nil)
	req.Header.Set(HeaderName, proof)
	got, err := v.VerifyHeader(req, HTTPURIFromRequest(base, req), "")
	if err != nil {
		t.Fatalf("VerifyHeader: %v", err)
	}
	if got.HTM != http.MethodPost {
		t.Fatalf("htm = %s", got.HTM)
	}
}

func TestParseAuthorization(t *testing.T) {
	scheme, token, ok := ParseAuthorization("DPoP abc.def.ghi")
	if !ok || scheme != "DPoP" || token != "abc.def.ghi" {
		t.Fatalf("got %s %s %v", scheme, token, ok)
	}
	if _, _, ok := ParseAuthorization("Bearer"); ok {
		t.Fatal("expected failure")
	}
}

func TestAccessTokenHashStable(t *testing.T) {
	h1 := AccessTokenHash("tok")
	h2 := AccessTokenHash("tok")
	if h1 != h2 || h1 == "" {
		t.Fatalf("hash unstable: %s %s", h1, h2)
	}
	// Sanity: known encoding length for sha256 base64url without padding
	if len(h1) != base64.RawURLEncoding.EncodedLen(32) {
		t.Fatalf("unexpected hash length %d", len(h1))
	}
}

func TestNonceRequired(t *testing.T) {
	v := NewVerifier(2 * time.Minute)
	v.RequireNonce = true
	htu := "https://as.example/token"
	proof, _, _ := newECProof(t, map[string]interface{}{
		"jti": "n1",
		"htm": "POST",
		"htu": htu,
		"iat": float64(time.Now().Unix()),
	})
	_, err := v.Verify(proof, "POST", htu, "")
	if err != ErrUseNonce {
		t.Fatalf("expected ErrUseNonce, got %v", err)
	}

	nonce := v.Nonces.Issue()
	proof2, _, _ := newECProof(t, map[string]interface{}{
		"jti":   "n2",
		"htm":   "POST",
		"htu":   htu,
		"iat":   float64(time.Now().Unix()),
		"nonce": nonce,
	})
	if _, err := v.Verify(proof2, "POST", htu, ""); err != nil {
		t.Fatalf("expected success with nonce: %v", err)
	}

	// Replay of same nonce must fail
	proof3, _, _ := newECProof(t, map[string]interface{}{
		"jti":   "n3",
		"htm":   "POST",
		"htu":   htu,
		"iat":   float64(time.Now().Unix()),
		"nonce": nonce,
	})
	_, err = v.Verify(proof3, "POST", htu, "")
	if err != ErrUseNonce {
		t.Fatalf("expected ErrUseNonce on reused nonce, got %v", err)
	}
}

func TestExtractJKTFromProof(t *testing.T) {
	proof, jkt, _ := newECProof(t, map[string]interface{}{
		"jti": "e1",
		"htm": "GET",
		"htu": "https://as.example/authorize",
		"iat": float64(time.Now().Unix()),
	})
	got, err := ExtractJKTFromProof(proof)
	if err != nil {
		t.Fatal(err)
	}
	if got != jkt {
		t.Fatalf("jkt mismatch: got %s want %s", got, jkt)
	}
}
