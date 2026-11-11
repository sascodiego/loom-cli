package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the per-project router configuration. Credentials never live here:
// they come from the environment.
type Config struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
}

// Environment variables the Cloudflare provider reads at call time.
const (
	EnvCloudflareToken   = "CLOUDFLARE_API_TOKEN"
	EnvCloudflareAccount = "CLOUDFLARE_ACCOUNT_ID"
)

// AccountPlaceholder is the token an endpoint may use instead of embedding the
// account id, so the tracked config never has to carry it.
const AccountPlaceholder = "{account_id}"

// ConfigPath returns .router/config.json for a repository root.
func ConfigPath(root string) string {
	return filepath.Join(RouterDir(root), "config.json")
}

// LoadConfig reads and validates .router/config.json. A missing file is a
// validation error that names the path, because decide cannot run without it.
func LoadConfig(root string) (*Config, error) {
	path := ConfigPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no router config at %s (write %s)", path, filepath.Join(".router", "config.json"))
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate requires every field the provider needs.
func (c *Config) Validate() error {
	switch {
	case c.Provider == "":
		return fmt.Errorf("config: missing provider")
	case c.Model == "":
		return fmt.Errorf("config: missing model")
	case c.Endpoint == "":
		return fmt.Errorf("config: missing endpoint")
	}
	if strings.Contains(c.Endpoint, "\n") || strings.Contains(c.Endpoint, " ") {
		return fmt.Errorf("config: endpoint must be a single URL")
	}
	return nil
}

// ResolveEndpoint substitutes the account placeholder from the environment.
// An endpoint without the placeholder never consults it, so the variable stays
// optional exactly as documented; a placeholder with the variable unset is an
// error naming it, because the URL cannot be built.
func (c *Config) ResolveEndpoint(getenv func(string) string) (string, error) {
	if !strings.Contains(c.Endpoint, AccountPlaceholder) {
		return c.Endpoint, nil
	}
	account := strings.TrimSpace(getenv(EnvCloudflareAccount))
	if account == "" {
		return "", fmt.Errorf("endpoint needs %s but it is not set in the environment", EnvCloudflareAccount)
	}
	return strings.ReplaceAll(c.Endpoint, AccountPlaceholder, account), nil
}
