package handlers

import (
	"testing"

	"oauth2-server/pkg/config"
)

func TestIsCustomFlow_ScopeMatch(t *testing.T) {
	p := config.UpstreamPromptPolicyConfig{
		CustomScope:                     "eduID",
		CustomAuthorizationDetailsType:  "openid_credential",
		CustomCredentialConfigurationID: "eduID",
	}

	if !isCustomFlow([]string{"openid", "eduID"}, "", p) {
		t.Fatalf("expected custom flow when scope contains eduID")
	}
	if isCustomFlow([]string{"openid", "profile"}, "", p) {
		t.Fatalf("did not expect custom flow without custom scope or authorization_details")
	}
}

func TestIsCustomFlow_AuthorizationDetailsMatch(t *testing.T) {
	p := config.UpstreamPromptPolicyConfig{
		CustomScope:                     "eduID",
		CustomAuthorizationDetailsType:  "openid_credential",
		CustomCredentialConfigurationID: "eduID",
	}

	jsonStr := `[{"type":"openid_credential","credential_configuration_id":"eduID"}]`
	if !isCustomFlow([]string{"openid"}, jsonStr, p) {
		t.Fatalf("expected custom flow when authorization_details matches")
	}

	nonMatch := `[{"type":"openid_credential","credential_configuration_id":"other"}]`
	if isCustomFlow([]string{"openid"}, nonMatch, p) {
		t.Fatalf("did not expect custom flow when credential_configuration_id does not match")
	}
}

func TestIsCustomFlow_AuthorizationDetailsInvalidJSON(t *testing.T) {
	p := config.UpstreamPromptPolicyConfig{
		CustomScope:                     "eduID",
		CustomAuthorizationDetailsType:  "openid_credential",
		CustomCredentialConfigurationID: "eduID",
	}

	if isCustomFlow([]string{"openid"}, `not-json`, p) {
		t.Fatalf("did not expect custom flow on invalid authorization_details JSON")
	}
}

func TestPolicyRuleMatches_ANDLogic(t *testing.T) {
	rule := config.UpstreamPromptPolicyRule{
		Name:                           "EDUID",
		Enabled:                        true,
		MatchScope:                     "eduID",
		MatchAuthorizationDetailsType:  "openid_credential",
		MatchCredentialConfigurationID: "eduID",
	}

	// Missing authz_details should fail (AND logic).
	if policyRuleMatches([]string{"openid", "eduID"}, "", rule) {
		t.Fatalf("expected no match when authz_details matcher configured but missing")
	}

	// Missing scope should fail (AND logic).
	jsonStr := `[{"type":"openid_credential","credential_configuration_id":"eduID"}]`
	if policyRuleMatches([]string{"openid"}, jsonStr, rule) {
		t.Fatalf("expected no match when scope matcher configured but missing")
	}

	// Both match should succeed.
	if !policyRuleMatches([]string{"openid", "eduID"}, jsonStr, rule) {
		t.Fatalf("expected match when both scope and authz_details match")
	}
}

func TestPolicyRuleMatches_RequiresAtLeastOneMatcher(t *testing.T) {
	rule := config.UpstreamPromptPolicyRule{Name: "EMPTY", Enabled: true}
	if policyRuleMatches([]string{"openid"}, "", rule) {
		t.Fatalf("expected no match when no matchers configured")
	}
}
