package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultServerURL = "http://localhost:7912"
	DefaultAPIPath   = "/api/v1"
	DefaultTimeout   = 10 * time.Second
	TestedVersion    = "0.23.1"
)

type Config struct {
	ServerURL string
	Timeout   time.Duration
	Insecure  bool
	CACert    string
	Source    string // how ServerURL was resolved
}

type fileConfig struct {
	Server string `toml:"server"`
}

func Load(flagServer string) (*Config, error) {
	cfg := &Config{Timeout: DefaultTimeout}

	if d := os.Getenv("SPOOLMAN_TIMEOUT"); d != "" {
		t, err := time.ParseDuration(d)
		if err != nil {
			return nil, fmt.Errorf("SPOOLMAN_TIMEOUT: %w", err)
		}
		cfg.Timeout = t
	}
	if os.Getenv("SPOOLMAN_INSECURE") == "1" {
		cfg.Insecure = true
	}
	if v := os.Getenv("SPOOLMAN_CA_CERT"); v != "" {
		cfg.CACert = v
	}

	switch {
	case flagServer != "":
		cfg.ServerURL = flagServer
		cfg.Source = "flag"
	case os.Getenv("SPOOLMAN_URL") != "":
		cfg.ServerURL = os.Getenv("SPOOLMAN_URL")
		cfg.Source = "env"
	default:
		if url, err := loadFromFile(); err == nil && url != "" {
			cfg.ServerURL = url
			cfg.Source = "config-file"
		} else {
			cfg.ServerURL = DefaultServerURL
			cfg.Source = "default"
		}
	}

	cfg.ServerURL = normalizeURL(cfg.ServerURL)
	return cfg, nil
}

func normalizeURL(raw string) string {
	// Strip trailing slash
	raw = strings.TrimRight(raw, "/")
	// If no path component beyond host, append /api/v1
	if !strings.Contains(stripScheme(raw), "/") {
		raw += DefaultAPIPath
	}
	return raw
}

func stripScheme(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		return u[i+3:]
	}
	return u
}

func loadFromFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", "spoolctl", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return "", err
	}
	return fc.Server, nil
}

func ConfigFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "spoolctl", "config.toml")
}

func (c *Config) HTTPClient() (*http.Client, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: c.Insecure} //nolint:gosec
	if c.CACert != "" {
		pem, err := os.ReadFile(c.CACert)
		if err != nil {
			return nil, fmt.Errorf("reading CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse CA cert %s", c.CACert)
		}
		tlsCfg.RootCAs = pool
	}
	return &http.Client{
		Timeout:   c.Timeout,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}, nil
}
