package state

import (
	"strings"
	"testing"
)

// The endpoint may name the account instead of embedding it, so a config file
// can be committed without the account id while the environment supplies it.
func TestResolveEndpointWithoutPlaceholderIgnoresTheEnvironment(t *testing.T) {
	c := &Config{Provider: "cloudflare", Model: "m", Endpoint: "https://api.example/v1/run"}
	got, err := c.ResolveEndpoint(func(string) string {
		t.Fatal("the environment must not be consulted when the endpoint has no placeholder")
		return ""
	})
	if err != nil {
		t.Fatalf("ResolveEndpoint: %v", err)
	}
	if got != c.Endpoint {
		t.Fatalf("endpoint = %q, want it unchanged", got)
	}
}

func TestResolveEndpointSubstitutesTheAccountID(t *testing.T) {
	c := &Config{
		Provider: "cloudflare",
		Model:    "m",
		Endpoint: "https://api.example/client/v4/accounts/{account_id}/ai/run/@cf/cloudflare/clef",
	}
	got, err := c.ResolveEndpoint(func(name string) string {
		if name != EnvCloudflareAccount {
			t.Fatalf("looked up %q, want %q", name, EnvCloudflareAccount)
		}
		return "abc123"
	})
	if err != nil {
		t.Fatalf("ResolveEndpoint: %v", err)
	}
	if strings.Contains(got, "{account_id}") {
		t.Fatalf("endpoint = %q, placeholder left in place", got)
	}
	if !strings.Contains(got, "/accounts/abc123/") {
		t.Fatalf("endpoint = %q, want the account id substituted", got)
	}
}

func TestResolveEndpointFailsWhenTheAccountIDIsMissing(t *testing.T) {
	c := &Config{Provider: "cloudflare", Model: "m", Endpoint: "https://api.example/accounts/{account_id}/run"}
	_, err := c.ResolveEndpoint(func(string) string { return "" })
	if err == nil || !strings.Contains(err.Error(), EnvCloudflareAccount) {
		t.Fatalf("ResolveEndpoint = %v, want an error naming %s", err, EnvCloudflareAccount)
	}
}
