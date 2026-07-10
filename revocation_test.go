package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// jtiOf must recover the token id from a signed token's payload (the value a
// hub denylists) and fail safe (empty) on garbage.
func TestJTIOf(t *testing.T) {
	body, _ := json.Marshal(LicensePayload{JTI: "abc123", Org: "acme", Email: "a@b.c"})
	tok := base64.RawURLEncoding.EncodeToString(body) + ".signature-part"
	if got := jtiOf(tok); got != "abc123" {
		t.Fatalf("jtiOf = %q, want abc123", got)
	}
	if jtiOf("not-a-token") != "" {
		t.Error("garbage token should yield empty jti")
	}
	if jtiOf("") != "" {
		t.Error("empty token should yield empty jti")
	}
}
