package config

import (
	"strings"
	"testing"
)

func TestParseInitialClientsJSON_Array(t *testing.T) {
	raw := `[
  {
    "id": "webapp",
    "secret": "s3cret",
    "redirect_uris": ["https://app.example.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "token_endpoint_auth_method": "client_secret_basic",
    "scopes": ["openid", "profile"]
  },
  {
    "client_id": "pkjwt",
    "token_endpoint_auth_method": "private_key_jwt",
    "token_endpoint_auth_signing_alg": "ES256",
    "grant_types": ["client_credentials"],
    "jwks": {
      "keys": [{
        "kty": "EC",
        "crv": "P-256",
        "x": "AAAA",
        "y": "BBBB",
        "kid": "k1",
        "use": "sig",
        "alg": "ES256"
      }]
    }
  }
]`
	clients, err := ParseInitialClientsJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("got %d clients", len(clients))
	}
	if clients[0].ID != "webapp" || clients[0].Secret != "s3cret" {
		t.Fatalf("webapp client: %+v", clients[0])
	}
	if clients[1].ID != "pkjwt" {
		t.Fatalf("pkjwt id: %s", clients[1].ID)
	}
	if clients[1].TokenEndpointAuthMethod != "private_key_jwt" {
		t.Fatalf("auth method: %s", clients[1].TokenEndpointAuthMethod)
	}
	if !strings.Contains(clients[1].Jwks, `"kid":"k1"`) && !strings.Contains(clients[1].Jwks, `"kid": "k1"`) {
		t.Fatalf("jwks not preserved: %s", clients[1].Jwks)
	}
}

func TestParseInitialClientsJSON_ObjectMap(t *testing.T) {
	raw := `{
  "spa": {
    "public": true,
    "redirect_uris": ["https://spa.example.com/cb"],
    "grant_types": ["authorization_code"]
  },
  "svc": {
    "secret": "svc-secret",
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_post"
  }
}`
	clients, err := ParseInitialClientsJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("got %d clients", len(clients))
	}
	byID := map[string]ClientConfig{}
	for _, c := range clients {
		byID[c.ID] = c
	}
	if !byID["spa"].Public || byID["spa"].TokenEndpointAuthMethod != "none" {
		t.Fatalf("spa: %+v", byID["spa"])
	}
	if byID["svc"].Secret != "svc-secret" {
		t.Fatalf("svc: %+v", byID["svc"])
	}
}

func TestParseInitialClientsJSON_SingleObject(t *testing.T) {
	raw := `{
  "id": "lonely",
  "secret": "x",
  "grant_types": ["client_credentials"]
}`
	clients, err := ParseInitialClientsJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(clients) != 1 || clients[0].ID != "lonely" {
		t.Fatalf("got %+v", clients)
	}
}

func TestMergeClientsByID(t *testing.T) {
	existing := []ClientConfig{
		{ID: "a", Secret: "old"},
		{ID: "b", Secret: "keep"},
	}
	incoming := []ClientConfig{
		{ID: "a", Secret: "new"},
		{ID: "c", Secret: "added"},
	}
	merged := mergeClientsByID(existing, incoming)
	if len(merged) != 3 {
		t.Fatalf("len=%d", len(merged))
	}
	if merged[0].Secret != "new" {
		t.Fatalf("a not replaced: %+v", merged[0])
	}
	if merged[1].Secret != "keep" {
		t.Fatalf("b changed: %+v", merged[1])
	}
	if merged[2].ID != "c" {
		t.Fatalf("c missing: %+v", merged)
	}
}

func TestLoadInitialClientsFromEnv(t *testing.T) {
	t.Setenv("INITIAL_CLIENTS", `[
	  {"id":"env-client","secret":"env-secret","grant_types":["client_credentials"],"scopes":["openid"]}
	]`)
	cfg := &Config{
		Clients: []ClientConfig{{ID: "yaml-client", Secret: "yaml"}},
	}
	if err := cfg.loadInitialClientsFromEnv(); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Clients) != 2 {
		t.Fatalf("clients=%d", len(cfg.Clients))
	}
	found := false
	for _, c := range cfg.Clients {
		if c.ID == "env-client" && c.Secret == "env-secret" {
			found = true
		}
	}
	if !found {
		t.Fatalf("env client missing: %+v", cfg.Clients)
	}
}
