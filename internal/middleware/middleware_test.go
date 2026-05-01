package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestNormalizeEscapedQuerySeparators_ReplacesEscapedSeparators(t *testing.T) {
	req := httptest.NewRequest("GET", "/authorize?client_id=unime&amp;request_uri=urn:test", nil)

	changed := NormalizeEscapedQuerySeparators(req)
	if !changed {
		t.Fatalf("expected query normalization to report changes")
	}

	got := req.URL.RawQuery
	want := "client_id=unime&request_uri=urn:test"
	if got != want {
		t.Fatalf("unexpected normalized query: got %q, want %q", got, want)
	}
}

func TestNormalizeEscapedQuerySeparators_ReplacesEscapedUnicodePattern(t *testing.T) {
	req := httptest.NewRequest("GET", "/authorize?client_id=unime\\u0026amp;request_uri=urn:test", nil)

	changed := NormalizeEscapedQuerySeparators(req)
	if !changed {
		t.Fatalf("expected query normalization to report changes")
	}

	got := req.URL.RawQuery
	want := "client_id=unime&request_uri=urn:test"
	if got != want {
		t.Fatalf("unexpected normalized query: got %q, want %q", got, want)
	}
}

func TestNormalizeEscapedQuerySeparators_NoChangeForValidQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/authorize?client_id=unime&request_uri=urn:test", nil)

	changed := NormalizeEscapedQuerySeparators(req)
	if changed {
		t.Fatalf("expected query normalization to report no changes")
	}

	got := req.URL.RawQuery
	want := "client_id=unime&request_uri=urn:test"
	if got != want {
		t.Fatalf("unexpected query mutation: got %q, want %q", got, want)
	}
}

func TestNormalizeEscapedQuerySeparators_HandlesNilRequest(t *testing.T) {
	if NormalizeEscapedQuerySeparators(nil) {
		t.Fatalf("expected nil request to return false")
	}
}
