package handlers

import (
	"encoding/json"
	"net/url"
	"strings"

	"oauth2-server/pkg/config"
)

type authorizationDetailsItem struct {
	Type                      string `json:"type"`
	CredentialConfigurationID string `json:"credential_configuration_id"`
}

func getAuthorizationDetailsJSON(q url.Values) (string, bool) {
	// Spec param name is "authorization_details".
	// Some clients may send "authorisation_details" (variant spelling), so accept it too.
	if v := strings.TrimSpace(q.Get("authorization_details")); v != "" {
		return v, true
	}
	if v := strings.TrimSpace(q.Get("authorisation_details")); v != "" {
		return v, true
	}
	return "", false
}

func isCustomFlow(effectiveScopes []string, authorizationDetailsJSON string, policy config.UpstreamPromptPolicyConfig) bool {
	customScope := strings.TrimSpace(policy.CustomScope)
	if customScope != "" {
		for _, s := range effectiveScopes {
			if s == customScope {
				return true
			}
		}
	}

	authorizationDetailsJSON = strings.TrimSpace(authorizationDetailsJSON)
	if authorizationDetailsJSON == "" {
		return false
	}

	var details []authorizationDetailsItem
	if err := json.Unmarshal([]byte(authorizationDetailsJSON), &details); err != nil {
		// If we can't parse it, don't guess.
		return false
	}

	wantType := strings.TrimSpace(policy.CustomAuthorizationDetailsType)
	wantCredID := strings.TrimSpace(policy.CustomCredentialConfigurationID)
	if wantType == "" || wantCredID == "" {
		return false
	}

	for _, d := range details {
		if d.Type == wantType && d.CredentialConfigurationID == wantCredID {
			return true
		}
	}

	return false
}

func parseAuthorizationDetailsItems(authorizationDetailsJSON string) ([]authorizationDetailsItem, bool) {
	authorizationDetailsJSON = strings.TrimSpace(authorizationDetailsJSON)
	if authorizationDetailsJSON == "" {
		return nil, false
	}
	var details []authorizationDetailsItem
	if err := json.Unmarshal([]byte(authorizationDetailsJSON), &details); err != nil {
		return nil, false
	}
	return details, true
}

// policyRuleMatches implements AND logic across all configured matchers:
// - If a matcher value is empty, it is ignored.
// - If it is non-empty, it must match for the policy to match.
func policyRuleMatches(effectiveScopes []string, authorizationDetailsJSON string, rule config.UpstreamPromptPolicyRule) bool {
	// At least one matcher should be configured; otherwise it's too easy to accidentally match everything.
	hasAnyMatcher := strings.TrimSpace(rule.MatchScope) != "" ||
		strings.TrimSpace(rule.MatchAuthorizationDetailsType) != "" ||
		strings.TrimSpace(rule.MatchCredentialConfigurationID) != ""
	if !hasAnyMatcher {
		return false
	}

	// Scope matcher
	if want := strings.TrimSpace(rule.MatchScope); want != "" {
		found := false
		for _, s := range effectiveScopes {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// authorization_details matcher
	wantType := strings.TrimSpace(rule.MatchAuthorizationDetailsType)
	wantCredID := strings.TrimSpace(rule.MatchCredentialConfigurationID)
	if wantType != "" || wantCredID != "" {
		details, ok := parseAuthorizationDetailsItems(authorizationDetailsJSON)
		if !ok {
			return false
		}

		itemMatched := false
		for _, d := range details {
			if wantType != "" && d.Type != wantType {
				continue
			}
			if wantCredID != "" && d.CredentialConfigurationID != wantCredID {
				continue
			}
			itemMatched = true
			break
		}
		if !itemMatched {
			return false
		}
	}

	return true
}
