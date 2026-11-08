package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"oauth2-server/internal/store"
	"oauth2-server/internal/store/storages"
	"oauth2-server/internal/store/types"
	"oauth2-server/internal/utils"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/ory/fosite"
	"github.com/ory/fosite/storage"
	"github.com/sirupsen/logrus"
)

func newTestStorage(t *testing.T) store.Storage {
	t.Helper()
	logger := logrus.New()
	mem := storages.NewMemoryStoreWrapper(storage.NewMemoryStore(), logger)
	return store.NewCustomStorage(mem, logger)
}

func signES256(t *testing.T, key *ecdsa.PrivateKey, claims map[string]interface{}) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"))
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	raw, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return raw
}

func signHS256(t *testing.T, secret string, claims map[string]interface{}) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte(secret)}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	raw, err := jwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return raw
}

func TestPrivateKeyJWTSuccessAndReplay(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage(t)
	sm := store.NewSecretManager([]byte("0123456789abcdef0123456789abcdef"))

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwks := &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &priv.PublicKey,
		KeyID:     "test-key",
		Algorithm: string(jose.ES256),
		Use:       "sig",
	}}}

	clientID := "pkjwt-client"
	client := &types.CustomClient{
		DefaultClient: &fosite.DefaultClient{
			ID:         clientID,
			GrantTypes: []string{"client_credentials"},
			Public:     false,
			Secret:     []byte("unused"),
		},
		JSONWebKeys:                       jwks,
		TokenEndpointAuthMethod:           "private_key_jwt",
		TokenEndpointAuthSigningAlgorithm: "ES256",
	}
	if err := storage.CreateClient(ctx, client); err != nil {
		t.Fatal(err)
	}

	tokenURL := "http://localhost:8080/token"
	verifier := NewAssertionVerifier(storage, sm, []string{tokenURL, "http://localhost:8080"})

	now := time.Now()
	claims := map[string]interface{}{
		"iss": clientID,
		"sub": clientID,
		"aud": tokenURL,
		"exp": now.Add(5 * time.Minute).Unix(),
		"iat": now.Unix(),
		"jti": "jti-1",
	}
	assertion := signES256(t, priv, claims)

	got, err := verifier.Authenticate(ctx, clientAssertionJWTBearerType, assertion, clientID)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if got.GetID() != clientID {
		t.Fatalf("client id = %s", got.GetID())
	}

	_, err = verifier.Authenticate(ctx, clientAssertionJWTBearerType, assertion, clientID)
	if err == nil {
		t.Fatal("expected replay rejection")
	}
}

func TestPrivateKeyJWTRejectsBadAudience(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage(t)
	sm := store.NewSecretManager([]byte("0123456789abcdef0123456789abcdef"))

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientID := "pkjwt-aud"
	client := &types.CustomClient{
		DefaultClient: &fosite.DefaultClient{
			ID:     clientID,
			Public: false,
			Secret: []byte("x"),
		},
		JSONWebKeys: &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key: &priv.PublicKey, KeyID: "test-key", Algorithm: string(jose.ES256), Use: "sig",
		}}},
		TokenEndpointAuthMethod:           "private_key_jwt",
		TokenEndpointAuthSigningAlgorithm: "ES256",
	}
	_ = storage.CreateClient(ctx, client)

	verifier := NewAssertionVerifier(storage, sm, []string{"http://localhost:8080/token"})
	now := time.Now()
	assertion := signES256(t, priv, map[string]interface{}{
		"iss": clientID,
		"sub": clientID,
		"aud": "https://evil.example/token",
		"exp": now.Add(time.Minute).Unix(),
		"iat": now.Unix(),
		"jti": "jti-aud",
	})
	if _, err := verifier.Authenticate(ctx, clientAssertionJWTBearerType, assertion, ""); err == nil {
		t.Fatal("expected audience rejection")
	}
}

func TestClientSecretJWTSuccess(t *testing.T) {
	ctx := context.Background()
	storage := newTestStorage(t)
	key := []byte("0123456789abcdef0123456789abcdef")
	sm := store.NewSecretManager(key)

	clientID := "hs-client"
	secret := "super-secret-client-value"
	hashed, err := storeHash(secret)
	if err != nil {
		t.Fatal(err)
	}
	client := &types.CustomClient{
		DefaultClient: &fosite.DefaultClient{
			ID:     clientID,
			Secret: hashed,
			Public: false,
		},
		TokenEndpointAuthMethod:           "client_secret_jwt",
		TokenEndpointAuthSigningAlgorithm: "HS256",
	}
	if err := storage.CreateClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	enc, err := sm.EncryptSecret(secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.StoreClientSecret(ctx, clientID, enc); err != nil {
		t.Fatal(err)
	}

	tokenURL := "http://localhost:8080/token"
	verifier := NewAssertionVerifier(storage, sm, []string{tokenURL})
	now := time.Now()
	assertion := signHS256(t, secret, map[string]interface{}{
		"iss": clientID,
		"sub": clientID,
		"aud": tokenURL,
		"exp": now.Add(2 * time.Minute).Unix(),
		"iat": now.Unix(),
		"jti": "jti-hs-1",
	})
	got, err := verifier.Authenticate(ctx, clientAssertionJWTBearerType, assertion, clientID)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if got.GetID() != clientID {
		t.Fatalf("unexpected client %s", got.GetID())
	}
}

func TestParseClientJWKS(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: &priv.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig",
	}}}
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseClientJWKS(raw)
	if err != nil || parsed == nil || len(parsed.Keys) != 1 {
		t.Fatalf("parse failed: %v %#v", err, parsed)
	}
}

func storeHash(secret string) ([]byte, error) {
	return utils.HashSecret(secret)
}
