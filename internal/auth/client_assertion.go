package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"oauth2-server/internal/store"
	"oauth2-server/internal/store/types"

	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/ory/fosite"
)

const clientAssertionJWTBearerType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// Default max assertion lifetime (RFC 7523 recommends short-lived assertions).
const defaultMaxAssertionLifetime = time.Hour

// AssertionVerifier validates RFC 7523 client_assertion JWTs for private_key_jwt
// and client_secret_jwt authentication.
type AssertionVerifier struct {
	Storage       store.Storage
	SecretManager *store.SecretManager
	// AcceptedAudiences are AS identities that may appear in aud (token URL, issuer, introspect URL).
	AcceptedAudiences []string
	MaxLifetime       time.Duration
	HTTPClient        *http.Client

	jwksCache sync.Map // uri -> cachedJWKS
	cacheTTL  time.Duration
}

type cachedJWKS struct {
	set       *jose.JSONWebKeySet
	fetchedAt time.Time
}

// NewAssertionVerifier constructs a verifier for RFC 7523 client assertions.
func NewAssertionVerifier(storage store.Storage, secretManager *store.SecretManager, audiences []string) *AssertionVerifier {
	return &AssertionVerifier{
		Storage:           storage,
		SecretManager:     secretManager,
		AcceptedAudiences: audiences,
		MaxLifetime:       defaultMaxAssertionLifetime,
		HTTPClient:        &http.Client{Timeout: 10 * time.Second},
		cacheTTL:          time.Hour,
	}
}

// Authenticate validates client_assertion / client_assertion_type and returns the client.
func (v *AssertionVerifier) Authenticate(ctx context.Context, assertionType, assertion, formClientID string) (fosite.Client, error) {
	if assertionType != clientAssertionJWTBearerType {
		return nil, fosite.ErrInvalidRequest.WithHintf("Unknown client_assertion_type '%s'.", assertionType)
	}
	if assertion == "" {
		return nil, fosite.ErrInvalidRequest.WithHintf("The client_assertion request parameter must be set when using client_assertion_type of '%s'.", clientAssertionJWTBearerType)
	}
	if len(v.AcceptedAudiences) == 0 {
		return nil, fosite.ErrServerError.WithHint("Authorization server audience is not configured for JWT client authentication.")
	}

	parsed, err := jwt.ParseSigned(assertion)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unable to parse client_assertion JWT.").WithWrap(err).WithDebug(err.Error())
	}

	var unsafeClaims jwt.Claims
	if err := parsed.UnsafeClaimsWithoutVerification(&unsafeClaims); err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unable to read client_assertion claims.").WithWrap(err).WithDebug(err.Error())
	}

	clientID := formClientID
	if clientID == "" {
		clientID = unsafeClaims.Subject
	}
	if clientID == "" {
		return nil, fosite.ErrInvalidClient.WithHint("The claim 'sub' from the client_assertion JSON Web Token is undefined.")
	}

	client, err := v.Storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unknown client").WithWrap(err).WithDebug(err.Error())
	}

	oidcClient, ok := client.(fosite.OpenIDConnectClient)
	if !ok {
		return nil, fosite.ErrInvalidRequest.WithHint("The server configuration does not support OpenID Connect specific authentication methods.")
	}

	authMethod := oidcClient.GetTokenEndpointAuthMethod()
	switch authMethod {
	case "private_key_jwt", "client_secret_jwt":
		// ok
	case "none":
		return nil, fosite.ErrInvalidClient.WithHint("This requested OAuth 2.0 client does not support client authentication, however 'client_assertion' was provided in the request.")
	case "attest_jwt_client_auth", "attest_tls_client_auth":
		return nil, fosite.ErrInvalidClient.WithHintf("Client authentication method '%s' must use the attestation authentication path.", authMethod)
	default:
		return nil, fosite.ErrInvalidClient.WithHintf("This requested OAuth 2.0 client only supports client authentication method '%s', however 'client_assertion' was provided in the request.", authMethod)
	}

	alg := ""
	if len(parsed.Headers) > 0 {
		alg = parsed.Headers[0].Algorithm
	}
	expectedAlg := oidcClient.GetTokenEndpointAuthSigningAlgorithm()
	if expectedAlg != "" && alg != expectedAlg {
		return nil, fosite.ErrInvalidClient.WithHintf("The 'client_assertion' uses signing algorithm '%s' but the requested OAuth 2.0 Client enforces signing algorithm '%s'.", alg, expectedAlg)
	}

	var key interface{}
	switch authMethod {
	case "private_key_jwt":
		key, err = v.resolvePublicKey(ctx, oidcClient, parsed)
		if err != nil {
			return nil, err
		}
	case "client_secret_jwt":
		key, err = v.resolveHMACKey(ctx, clientID, alg)
		if err != nil {
			return nil, err
		}
	}

	var claims jwt.Claims
	if err := parsed.Claims(key, &claims); err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unable to verify the integrity of the 'client_assertion' value.").WithWrap(err).WithDebug(err.Error())
	}

	now := time.Now()
	if err := claims.Validate(jwt.Expected{
		Issuer:  clientID,
		Subject: clientID,
		Time:    now,
	}); err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unable to verify client_assertion claims (iss/sub/exp/nbf).").WithWrap(err).WithDebug(err.Error())
	}

	if !audienceAccepted(claims.Audience, v.AcceptedAudiences) {
		return nil, fosite.ErrInvalidClient.WithHintf(
			"Claim 'audience' from 'client_assertion' must match the authorization server identity ('%s').",
			strings.Join(v.AcceptedAudiences, "' or '"))
	}

	if claims.Expiry == nil {
		return nil, fosite.ErrInvalidClient.WithHint("Claim 'exp' from 'client_assertion' must be set.")
	}

	maxLifetime := v.MaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = defaultMaxAssertionLifetime
	}
	issuedAt := now
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time()
	}
	if claims.Expiry.Time().Sub(issuedAt) > maxLifetime {
		return nil, fosite.ErrInvalidClient.WithHintf("The 'client_assertion' lifetime exceeds the maximum allowed duration of %s.", maxLifetime)
	}

	if claims.ID == "" {
		return nil, fosite.ErrInvalidClient.WithHint("Claim 'jti' from 'client_assertion' must be set but is not.")
	}
	if err := v.Storage.ClientAssertionJWTValid(ctx, claims.ID); err != nil {
		return nil, fosite.ErrJTIKnown.WithHint("Claim 'jti' from 'client_assertion' MUST only be used once.")
	}
	if err := v.Storage.SetClientAssertionJWT(ctx, claims.ID, claims.Expiry.Time()); err != nil {
		return nil, fosite.ErrServerError.WithWrap(err).WithDebug(err.Error())
	}

	return client, nil
}

func audienceAccepted(assertionAud jwt.Audience, accepted []string) bool {
	for _, want := range accepted {
		if want == "" {
			continue
		}
		for _, got := range assertionAud {
			if got == want {
				return true
			}
		}
	}
	return false
}

func (v *AssertionVerifier) resolveHMACKey(ctx context.Context, clientID, alg string) (interface{}, error) {
	switch alg {
	case string(jose.HS256), string(jose.HS384), string(jose.HS512), "":
		// ok
	default:
		return nil, fosite.ErrInvalidClient.WithHintf("client_secret_jwt requires an HMAC signing algorithm, got '%s'.", alg)
	}
	if v.SecretManager == nil {
		return nil, fosite.ErrServerError.WithHint("Client secret decryption is not configured.")
	}
	encrypted, err := v.Storage.GetClientSecret(ctx, clientID)
	if err != nil || encrypted == "" {
		return nil, fosite.ErrInvalidClient.WithHint("No client secret available for client_secret_jwt authentication.")
	}
	secret, err := v.SecretManager.DecryptSecret(encrypted)
	if err != nil {
		return nil, fosite.ErrServerError.WithHint("Failed to decrypt client secret.").WithWrap(err).WithDebug(err.Error())
	}
	return []byte(secret), nil
}

func (v *AssertionVerifier) resolvePublicKey(ctx context.Context, client fosite.OpenIDConnectClient, token *jwt.JSONWebToken) (interface{}, error) {
	set := client.GetJSONWebKeys()
	if set == nil || len(set.Keys) == 0 {
		if uri := client.GetJSONWebKeysURI(); uri != "" {
			fetched, err := v.fetchJWKS(ctx, uri)
			if err != nil {
				return nil, fosite.ErrInvalidClient.WithHint("Failed to fetch client JWKS.").WithWrap(err).WithDebug(err.Error())
			}
			set = fetched
		}
	}
	if set == nil || len(set.Keys) == 0 {
		return nil, fosite.ErrInvalidClient.WithHint("The OAuth 2.0 Client has no JSON Web Keys set registered, but they are needed to complete the request.")
	}

	kid := ""
	alg := ""
	if len(token.Headers) > 0 {
		kid = token.Headers[0].KeyID
		alg = token.Headers[0].Algorithm
	}

	keys := set.Keys
	if kid != "" {
		keys = set.Key(kid)
	}
	if len(keys) == 0 {
		return nil, fosite.ErrInvalidClient.WithHintf("The JSON Web Token uses signing key with kid '%s', which could not be found.", kid)
	}

	for _, k := range keys {
		if !k.Valid() {
			continue
		}
		switch alg {
		case string(jose.RS256), string(jose.RS384), string(jose.RS512),
			string(jose.PS256), string(jose.PS384), string(jose.PS512):
			if _, ok := k.Key.(*rsa.PublicKey); ok {
				return k.Key, nil
			}
		case string(jose.ES256), string(jose.ES384), string(jose.ES512):
			if _, ok := k.Key.(*ecdsa.PublicKey); ok {
				return k.Key, nil
			}
		default:
			// Fall back to first valid key if alg not constrained above.
			switch k.Key.(type) {
			case *rsa.PublicKey, *ecdsa.PublicKey:
				return k.Key, nil
			}
		}
	}

	return nil, fosite.ErrInvalidClient.WithHint("Unable to find a suitable public key to verify the client_assertion.")
}

func (v *AssertionVerifier) fetchJWKS(ctx context.Context, uri string) (*jose.JSONWebKeySet, error) {
	if v.cacheTTL > 0 {
		if cached, ok := v.jwksCache.Load(uri); ok {
			entry := cached.(cachedJWKS)
			if time.Since(entry.fetchedAt) < v.cacheTTL {
				return entry.set, nil
			}
		}
	}

	client := v.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks_uri returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, err
	}
	v.jwksCache.Store(uri, cachedJWKS{set: &set, fetchedAt: time.Now()})
	return &set, nil
}

// ParseClientJWKS parses an inline JWKS document from registration metadata.
func ParseClientJWKS(raw json.RawMessage) (*jose.JSONWebKeySet, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	// Allow a JSON string containing JWKS JSON.
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		raw = json.RawMessage(asString)
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, fmt.Errorf("invalid jwks: %w", err)
	}
	if len(set.Keys) == 0 {
		return nil, fmt.Errorf("jwks must contain at least one key")
	}
	return &set, nil
}

// AsOIDCClient returns the client as CustomClient when possible.
func AsOIDCClient(client fosite.Client) (*types.CustomClient, bool) {
	cc, ok := client.(*types.CustomClient)
	return cc, ok
}
