package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"oauth2-server/internal/dpop"
	"strings"
	"sync"
	"time"

	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/handler/rfc8693"
	"github.com/ory/fosite/token/jwt"
)

var (
	dpopVerifierOnce sync.Once
	dpopVerifier     *dpop.Verifier
)

func sharedDPoPVerifier(maxSkewSeconds int, nonceRequired bool) *dpop.Verifier {
	dpopVerifierOnce.Do(func() {
		skew := 2 * time.Minute
		if maxSkewSeconds > 0 {
			skew = time.Duration(maxSkewSeconds) * time.Second
		}
		dpopVerifier = dpop.NewVerifier(skew)
		dpopVerifier.RequireNonce = nonceRequired
	})
	// Keep RequireNonce in sync if config changes after first init (test-friendly)
	if dpopVerifier != nil {
		dpopVerifier.RequireNonce = nonceRequired
	}
	return dpopVerifier
}

func (h *TokenHandler) getDPoPVerifier() *dpop.Verifier {
	skew := 120
	nonceRequired := false
	if h.Configuration != nil && h.Configuration.DPoP != nil {
		if h.Configuration.DPoP.MaxClockSkewSeconds > 0 {
			skew = h.Configuration.DPoP.MaxClockSkewSeconds
		}
		nonceRequired = h.Configuration.DPoP.NonceRequired
	}
	return sharedDPoPVerifier(skew, nonceRequired)
}

func (h *TokenHandler) dpopEnabled() bool {
	return h.Configuration != nil && h.Configuration.DPoP != nil && h.Configuration.DPoP.Enabled
}

func (h *TokenHandler) dpopRequired() bool {
	return h.dpopEnabled() && h.Configuration.DPoP.Required
}

func (h *TokenHandler) dpopNonceRequired() bool {
	return h.dpopEnabled() && h.Configuration.DPoP.NonceRequired
}

// writeUseDPoPNonce writes RFC 9449 §8 use_dpop_nonce error response.
func writeUseDPoPNonce(w http.ResponseWriter, verifier *dpop.Verifier) {
	nonce := verifier.Nonces.Issue()
	w.Header().Set("DPoP-Nonce", nonce)
	w.Header().Set("WWW-Authenticate", `DPoP error="use_dpop_nonce"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             "use_dpop_nonce",
		"error_description": "Authorization server requires nonce in DPoP proof",
	})
}

// processDPoPForTokenRequest validates an optional/required DPoP proof on the token endpoint
// and binds the resulting jkt onto the access request session when present.
// boundJKT, when non-empty, is an authorization-time binding that the proof must match.
func (h *TokenHandler) processDPoPForTokenRequest(r *http.Request, session fosite.Session, boundJKT string) (jkt string, err error) {
	proofHeader := r.Header.Get(dpop.HeaderName)
	needsProof := h.dpopRequired() || boundJKT != ""

	if proofHeader == "" {
		if needsProof {
			return "", fosite.ErrInvalidRequest.WithHint("DPoP proof is required (RFC 9449)")
		}
		return "", nil
	}
	if !h.dpopEnabled() {
		h.Log.Debugf("⚠️ DPoP header present but DPOP_ENABLED=false — ignoring")
		return "", nil
	}

	baseURL := h.Configuration.GetEffectiveBaseURL(r)
	htu := dpop.HTTPURIFromRequest(baseURL, r)
	verifier := h.getDPoPVerifier()
	proof, verr := verifier.VerifyWithOptions(proofHeader, r.Method, htu, dpop.VerifyOptions{
		RequireNonce: h.dpopNonceRequired(),
	})
	if verr != nil {
		if errors.Is(verr, dpop.ErrUseNonce) {
			return "", errUseDPoPNonce
		}
		h.Log.Errorf("❌ DPoP proof validation failed: %v", verr)
		return "", fosite.ErrInvalidRequest.WithHint("Invalid DPoP proof").WithDebug(verr.Error())
	}

	if boundJKT != "" && proof.JKT != boundJKT {
		return "", fosite.ErrInvalidGrant.WithHint("DPoP proof key does not match authorization binding (dpop_jkt)")
	}

	if r.FormValue("grant_type") == "refresh_token" {
		if existing := getDPoPJKTFromSession(session); existing != "" && existing != proof.JKT {
			return "", fosite.ErrInvalidGrant.WithHint("DPoP proof key does not match token binding")
		}
	}

	if err := storeDPoPJKTInSession(session, proof.JKT); err != nil {
		return "", fosite.ErrServerError.WithDebug(err.Error())
	}
	h.Log.Debugf("✅ DPoP proof accepted, jkt=%s", proof.JKT)
	return proof.JKT, nil
}

// errUseDPoPNonce is a sentinel checked by token handlers to emit the RFC nonce challenge.
var errUseDPoPNonce = errors.New("use_dpop_nonce")

func storeDPoPJKTInSession(session fosite.Session, jkt string) error {
	if jkt == "" {
		return nil
	}
	switch s := session.(type) {
	case *openid.DefaultSession:
		if s.Claims == nil {
			s.Claims = &jwt.IDTokenClaims{}
		}
		if s.Claims.Extra == nil {
			s.Claims.Extra = make(map[string]interface{})
		}
		s.Claims.Extra["dpop_jkt"] = jkt
		s.Claims.Extra["cnf"] = dpop.ConfirmationClaim(jkt)
		return nil
	case *rfc8693.TokenExchangeSession:
		if s.Extra == nil {
			s.Extra = make(map[string]interface{})
		}
		s.Extra["dpop_jkt"] = jkt
		s.Extra["cnf"] = dpop.ConfirmationClaim(jkt)
		return nil
	default:
		return fmt.Errorf("unsupported session type %T for DPoP binding", session)
	}
}

func getDPoPJKTFromSession(session fosite.Session) string {
	if session == nil {
		return ""
	}
	switch s := session.(type) {
	case *openid.DefaultSession:
		if s.Claims != nil && s.Claims.Extra != nil {
			if jkt, ok := s.Claims.Extra["dpop_jkt"].(string); ok {
				return jkt
			}
			if cnf, ok := s.Claims.Extra["cnf"].(map[string]interface{}); ok {
				if jkt, ok := cnf["jkt"].(string); ok {
					return jkt
				}
			}
		}
	case *rfc8693.TokenExchangeSession:
		if s.Extra != nil {
			if jkt, ok := s.Extra["dpop_jkt"].(string); ok {
				return jkt
			}
			if cnf, ok := s.Extra["cnf"].(map[string]interface{}); ok {
				if jkt, ok := cnf["jkt"].(string); ok {
					return jkt
				}
			}
		}
	}
	return ""
}

func applyDPoPToAccessResponse(accessResponse fosite.AccessResponder, jkt string) {
	if jkt == "" || accessResponse == nil {
		return
	}
	accessResponse.SetTokenType(dpop.TokenType)
	accessResponse.SetExtra("cnf", dpop.ConfirmationClaim(jkt))
}

// applyDPoPToProxyJSON mutates a proxy token JSON map when jkt is set.
func applyDPoPToProxyJSON(resp map[string]interface{}, jkt string) {
	if jkt == "" || resp == nil {
		return
	}
	resp["token_type"] = dpop.TokenType
	resp["cnf"] = dpop.ConfirmationClaim(jkt)
}

// extractAccessTokenFromAuth returns the access token from Bearer or DPoP Authorization schemes.
func extractAccessTokenFromAuth(authHeader string) (scheme, token string, ok bool) {
	return dpop.ParseAuthorization(authHeader)
}

// requireDPoPForBoundToken validates DPoP proof when the token is DPoP-bound.
func requireDPoPForBoundToken(r *http.Request, baseURL string, verifier *dpop.Verifier, session fosite.Session, scheme, token string) error {
	jkt := getDPoPJKTFromSession(session)
	if jkt == "" {
		if strings.EqualFold(scheme, dpop.AuthScheme) {
			return fmt.Errorf("token is not DPoP-bound")
		}
		return nil
	}
	if !strings.EqualFold(scheme, dpop.AuthScheme) {
		return fmt.Errorf("DPoP-bound token requires Authorization: DPoP")
	}
	htu := dpop.HTTPURIFromRequest(baseURL, r)
	proof, err := verifier.VerifyHeader(r, htu, token)
	if err != nil {
		return err
	}
	if proof.JKT != jkt {
		return dpop.ErrJKTMismatch
	}
	return nil
}

// extractAuthorizeDPoPJKT resolves a DPoP key thumbprint for authorization/PAR binding
// from either the dpop_jkt parameter or a DPoP proof header (RFC 9449 §10 / §10.1).
func extractAuthorizeDPoPJKT(r *http.Request) (string, error) {
	if jkt := strings.TrimSpace(r.FormValue("dpop_jkt")); jkt != "" {
		return jkt, nil
	}
	if jkt := strings.TrimSpace(r.URL.Query().Get("dpop_jkt")); jkt != "" {
		return jkt, nil
	}
	proof := r.Header.Get(dpop.HeaderName)
	if proof == "" {
		return "", nil
	}
	return dpop.ExtractJKTFromProof(proof)
}

// lookupBoundDPoPJKT returns authorization-time binding for an authorization_code grant.
func (h *TokenHandler) lookupBoundDPoPJKT(r *http.Request, session fosite.Session) string {
	if jkt := getDPoPJKTFromSession(session); jkt != "" {
		return jkt
	}
	if r.FormValue("grant_type") == "authorization_code" && h.AuthCodeToDPoPJKTMap != nil {
		if code := r.FormValue("code"); code != "" {
			if jkt, ok := (*h.AuthCodeToDPoPJKTMap)[code]; ok {
				return jkt
			}
		}
	}
	return ""
}

// applyProxyDPoP validates DPoP for proxy mint paths and returns the bound jkt.
// On error it writes the HTTP response and returns ok=false.
func (h *TokenHandler) applyProxyDPoP(w http.ResponseWriter, r *http.Request, session fosite.Session) (jkt string, ok bool) {
	boundJKT := h.lookupBoundDPoPJKT(r, session)
	jkt, err := h.processDPoPForTokenRequest(r, session, boundJKT)
	if err != nil {
		if errors.Is(err, errUseDPoPNonce) {
			writeUseDPoPNonce(w, h.getDPoPVerifier())
			return "", false
		}
		h.Log.Errorf("❌ DPoP processing failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusBadRequest
		code := "invalid_request"
		desc := "Invalid DPoP proof"
		var rfcErr *fosite.RFC6749Error
		if errors.As(err, &rfcErr) {
			if rfcErr.CodeField > 0 {
				status = rfcErr.CodeField
			}
			if rfcErr.ErrorField != "" {
				code = rfcErr.ErrorField
			}
			if rfcErr.HintField != "" {
				desc = rfcErr.HintField
			}
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             code,
			"error_description": desc,
		})
		return "", false
	}
	return jkt, true
}

// setDPoPNonceResponseHeader offers a fresh nonce for subsequent proofs (RFC 9449 §8.2).
func (h *TokenHandler) setDPoPNonceResponseHeader(w http.ResponseWriter, jkt string) {
	if jkt == "" {
		return
	}
	w.Header().Set("DPoP-Nonce", h.getDPoPVerifier().Nonces.Issue())
}
