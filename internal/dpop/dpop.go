// Package dpop implements RFC 9449 OAuth 2.0 Demonstrating Proof of Possession (DPoP).
package dpop

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
)

const (
	// HeaderName is the HTTP header carrying a DPoP proof JWT.
	HeaderName = "DPoP"
	// TokenType is the OAuth access token type for DPoP-bound tokens.
	TokenType = "DPoP"
	// AuthScheme is the Authorization scheme for DPoP-bound access tokens.
	AuthScheme = "DPoP"
	// JWTType is the required JWT typ header value.
	JWTType = "dpop+jwt"
)

var (
	ErrMissingProof   = errors.New("dpop: missing DPoP proof")
	ErrInvalidProof   = errors.New("dpop: invalid DPoP proof")
	ErrReplay         = errors.New("dpop: proof replay detected")
	ErrHTMMismatch    = errors.New("dpop: htm claim mismatch")
	ErrHTUMismatch    = errors.New("dpop: htu claim mismatch")
	ErrATHMismatch    = errors.New("dpop: ath claim mismatch")
	ErrJKTMismatch    = errors.New("dpop: jkt mismatch")
	ErrUnsupportedAlg = errors.New("dpop: unsupported signing algorithm")
	ErrMissingATH     = errors.New("dpop: ath claim required for protected resource access")
	ErrClockSkew      = errors.New("dpop: iat outside acceptable window")
	ErrUseNonce       = errors.New("dpop: use_dpop_nonce")
	ErrInvalidNonce   = errors.New("dpop: invalid or expired nonce")
)

// SupportedAlgs are the asymmetric algorithms advertised and accepted for DPoP proofs.
var SupportedAlgs = []string{"ES256", "ES384", "ES512", "RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "EdDSA"}

// Proof holds validated claims from a DPoP proof JWT.
type Proof struct {
	JKT   string
	JTI   string
	HTM   string
	HTU   string
	IAT   time.Time
	ATH   string
	Nonce string
}

// NonceStore issues and consumes single-use DPoP nonces (RFC 9449 §8).
type NonceStore struct {
	ttl  time.Duration
	data sync.Map // nonce -> expiry unix
}

// NewNonceStore creates a nonce store with the given TTL (default 5 minutes).
func NewNonceStore(ttl time.Duration) *NonceStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &NonceStore{ttl: ttl}
}

// Issue creates a new opaque nonce.
func (s *NonceStore) Issue() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	s.data.Store(nonce, time.Now().Add(s.ttl).Unix())
	return nonce
}

// Consume validates and invalidates a nonce. Returns ErrInvalidNonce if unknown/expired.
func (s *NonceStore) Consume(nonce string) error {
	if nonce == "" {
		return ErrInvalidNonce
	}
	v, ok := s.data.LoadAndDelete(nonce)
	if !ok {
		return ErrInvalidNonce
	}
	exp, _ := v.(int64)
	if time.Now().Unix() > exp {
		return ErrInvalidNonce
	}
	return nil
}

// Verifier validates DPoP proofs and tracks jti values for replay protection.
type Verifier struct {
	MaxSkew    time.Duration
	Nonces     *NonceStore
	RequireNonce bool
	jtiTTL     time.Duration
	seen       sync.Map // jti -> expiry unix
	cleanOnce  sync.Once
}

// NewVerifier creates a DPoP verifier with the given clock-skew tolerance.
func NewVerifier(maxSkew time.Duration) *Verifier {
	if maxSkew <= 0 {
		maxSkew = 2 * time.Minute
	}
	v := &Verifier{
		MaxSkew: maxSkew,
		jtiTTL:  10 * time.Minute,
		Nonces:  NewNonceStore(5 * time.Minute),
	}
	return v
}

// VerifyOptions controls optional proof checks.
type VerifyOptions struct {
	AccessToken  string
	RequireNonce bool
}

// Verify validates a DPoP proof JWT for the given HTTP method and target URI.
// When accessToken is non-empty, the ath claim is required and must match.
func (v *Verifier) Verify(proofJWT, method, htu, accessToken string) (*Proof, error) {
	return v.VerifyWithOptions(proofJWT, method, htu, VerifyOptions{AccessToken: accessToken, RequireNonce: v.RequireNonce})
}

// VerifyWithOptions validates a DPoP proof with explicit options.
func (v *Verifier) VerifyWithOptions(proofJWT, method, htu string, opts VerifyOptions) (*Proof, error) {
	v.cleanOnce.Do(func() { go v.cleanupLoop() })

	if strings.TrimSpace(proofJWT) == "" {
		return nil, ErrMissingProof
	}

	joseAlgs := supportedJoseAlgs()
	parsed, err := jose.ParseSigned(proofJWT, joseAlgs)
	if err != nil {
		return nil, fmt.Errorf("%w: parse failed: %v", ErrInvalidProof, err)
	}
	if len(parsed.Signatures) == 0 {
		return nil, fmt.Errorf("%w: missing protected header", ErrInvalidProof)
	}
	hdr := parsed.Signatures[0].Header

	typ, _ := hdr.ExtraHeaders[jose.HeaderKey("typ")].(string)
	if !strings.EqualFold(typ, JWTType) {
		return nil, fmt.Errorf("%w: typ must be %s", ErrInvalidProof, JWTType)
	}
	if hdr.Algorithm == "" || strings.EqualFold(string(hdr.Algorithm), "none") {
		return nil, ErrUnsupportedAlg
	}
	if !isSupportedAlg(string(hdr.Algorithm)) {
		return nil, ErrUnsupportedAlg
	}
	if hdr.JSONWebKey == nil {
		return nil, fmt.Errorf("%w: jwk header required", ErrInvalidProof)
	}
	if !hdr.JSONWebKey.IsPublic() {
		return nil, fmt.Errorf("%w: jwk must be a public key", ErrInvalidProof)
	}

	payload, err := parsed.Verify(hdr.JSONWebKey)
	if err != nil {
		return nil, fmt.Errorf("%w: signature verification failed: %v", ErrInvalidProof, err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("%w: invalid claims JSON: %v", ErrInvalidProof, err)
	}

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return nil, fmt.Errorf("%w: jti required", ErrInvalidProof)
	}
	htm, _ := claims["htm"].(string)
	if !strings.EqualFold(htm, method) {
		return nil, ErrHTMMismatch
	}
	htuClaim, _ := claims["htu"].(string)
	if !htuEqual(htuClaim, htu) {
		return nil, ErrHTUMismatch
	}

	iat, err := claimAsTime(claims["iat"])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid iat: %v", ErrInvalidProof, err)
	}
	now := time.Now().UTC()
	if iat.After(now.Add(v.MaxSkew)) || iat.Before(now.Add(-v.MaxSkew)) {
		return nil, ErrClockSkew
	}

	nonce, _ := claims["nonce"].(string)
	requireNonce := opts.RequireNonce || v.RequireNonce
	if requireNonce {
		if nonce == "" {
			return nil, ErrUseNonce
		}
		if v.Nonces == nil {
			return nil, ErrInvalidNonce
		}
		if err := v.Nonces.Consume(nonce); err != nil {
			return nil, ErrUseNonce
		}
	} else if nonce != "" && v.Nonces != nil {
		// Optional nonce present: must still be valid if provided
		if err := v.Nonces.Consume(nonce); err != nil {
			return nil, ErrInvalidNonce
		}
	}

	ath, _ := claims["ath"].(string)
	if opts.AccessToken != "" {
		if ath == "" {
			return nil, ErrMissingATH
		}
		expected := AccessTokenHash(opts.AccessToken)
		if ath != expected {
			return nil, ErrATHMismatch
		}
	}

	jkt, err := Thumbprint(hdr.JSONWebKey)
	if err != nil {
		return nil, fmt.Errorf("%w: jkt computation failed: %v", ErrInvalidProof, err)
	}

	if _, loaded := v.seen.LoadOrStore(jti, now.Add(v.jtiTTL).Unix()); loaded {
		return nil, ErrReplay
	}

	return &Proof{
		JKT:   jkt,
		JTI:   jti,
		HTM:   strings.ToUpper(htm),
		HTU:   htuClaim,
		IAT:   iat,
		ATH:   ath,
		Nonce: nonce,
	}, nil
}

// ExtractJKTFromProof parses a DPoP proof enough to return the jwk thumbprint
// without enforcing htm/htu (used at authorization / PAR binding time).
func ExtractJKTFromProof(proofJWT string) (string, error) {
	if strings.TrimSpace(proofJWT) == "" {
		return "", ErrMissingProof
	}
	parsed, err := jose.ParseSigned(proofJWT, supportedJoseAlgs())
	if err != nil {
		return "", fmt.Errorf("%w: parse failed: %v", ErrInvalidProof, err)
	}
	if len(parsed.Signatures) == 0 || parsed.Signatures[0].Header.JSONWebKey == nil {
		return "", fmt.Errorf("%w: jwk header required", ErrInvalidProof)
	}
	hdr := parsed.Signatures[0].Header
	typ, _ := hdr.ExtraHeaders[jose.HeaderKey("typ")].(string)
	if !strings.EqualFold(typ, JWTType) {
		return "", fmt.Errorf("%w: typ must be %s", ErrInvalidProof, JWTType)
	}
	if _, err := parsed.Verify(hdr.JSONWebKey); err != nil {
		return "", fmt.Errorf("%w: signature verification failed: %v", ErrInvalidProof, err)
	}
	return Thumbprint(hdr.JSONWebKey)
}

// VerifyHeader extracts and verifies the DPoP header from an HTTP request.
func (v *Verifier) VerifyHeader(r *http.Request, htu, accessToken string) (*Proof, error) {
	return v.Verify(r.Header.Get(HeaderName), r.Method, htu, accessToken)
}

// AccessTokenHash returns the base64url-encoded SHA-256 hash of the access token (RFC 9449 ath).
func AccessTokenHash(accessToken string) string {
	sum := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// Thumbprint returns the JWK thumbprint (RFC 7638) for use as cnf.jkt.
func Thumbprint(jwk *jose.JSONWebKey) (string, error) {
	if jwk == nil {
		return "", fmt.Errorf("nil jwk")
	}
	tp, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tp), nil
}

// HTTPURI builds the htu value for a request: absolute URI without query or fragment.
func HTTPURI(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}

// HTTPURIFromRequest builds htu from the request as received (scheme + host + path),
// without query or fragment, per RFC 9449 Section 4.2.
func HTTPURIFromRequest(baseURL string, r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	// Fall back to configured base URL host if request host is empty
	if host == "" && baseURL != "" {
		return HTTPURI(baseURL, r.URL.Path)
	}
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	return scheme + "://" + host + path
}

// ParseAuthorization extracts the scheme and token from an Authorization header.
func ParseAuthorization(header string) (scheme, token string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ConfirmationClaim builds the cnf object for introspection / token responses.
func ConfirmationClaim(jkt string) map[string]interface{} {
	return map[string]interface{}{
		"jkt": jkt,
	}
}

func isSupportedAlg(alg string) bool {
	for _, a := range SupportedAlgs {
		if strings.EqualFold(a, alg) {
			return true
		}
	}
	return false
}

func supportedJoseAlgs() []jose.SignatureAlgorithm {
	out := make([]jose.SignatureAlgorithm, 0, len(SupportedAlgs))
	for _, a := range SupportedAlgs {
		out = append(out, jose.SignatureAlgorithm(a))
	}
	return out
}

func claimAsTime(v interface{}) (time.Time, error) {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0).UTC(), nil
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(i, 0).UTC(), nil
	case int64:
		return time.Unix(t, 0).UTC(), nil
	case int:
		return time.Unix(int64(t), 0).UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("unexpected iat type %T", v)
	}
}

func htuEqual(claim, expected string) bool {
	c, err1 := url.Parse(claim)
	e, err2 := url.Parse(expected)
	if err1 != nil || err2 != nil {
		return strings.TrimRight(claim, "/") == strings.TrimRight(expected, "/")
	}
	// Compare without query/fragment; normalize trailing slash on path
	c.RawQuery, c.Fragment = "", ""
	e.RawQuery, e.Fragment = "", ""
	c.Path = strings.TrimRight(c.Path, "/")
	e.Path = strings.TrimRight(e.Path, "/")
	if c.Path == "" {
		c.Path = ""
	}
	if e.Path == "" {
		e.Path = ""
	}
	return strings.EqualFold(c.Scheme, e.Scheme) &&
		strings.EqualFold(c.Host, e.Host) &&
		c.Path == e.Path
}

func (v *Verifier) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now().Unix()
		v.seen.Range(func(key, value interface{}) bool {
			if exp, ok := value.(int64); ok && exp < now {
				v.seen.Delete(key)
			}
			return true
		})
	}
}
