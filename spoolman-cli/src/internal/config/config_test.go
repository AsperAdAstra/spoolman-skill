package config

import (
	"os"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"http://localhost:7912", "http://localhost:7912/api/v1"},
		{"http://localhost:7912/", "http://localhost:7912/api/v1"},
		{"http://spoolman.lan:7912", "http://spoolman.lan:7912/api/v1"},
		{"http://spoolman.lan/api/v1", "http://spoolman.lan/api/v1"},
		{"http://spoolman.lan/custom/path", "http://spoolman.lan/custom/path"},
		{"https://spoolman.example.com", "https://spoolman.example.com/api/v1"},
	}
	for _, tc := range cases {
		got := normalizeURL(tc.input)
		if got != tc.want {
			t.Errorf("normalizeURL(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestLoadResolutionOrder(t *testing.T) {
	// Clean env
	os.Unsetenv("SPOOLMAN_URL")
	os.Unsetenv("SPOOLMAN_TIMEOUT")
	os.Unsetenv("SPOOLMAN_INSECURE")
	os.Unsetenv("SPOOLMAN_CA_CERT")

	t.Run("flag wins", func(t *testing.T) {
		cfg, err := Load("http://flag.host")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Source != "flag" {
			t.Errorf("source = %q; want flag", cfg.Source)
		}
		if cfg.ServerURL != "http://flag.host/api/v1" {
			t.Errorf("URL = %q; want http://flag.host/api/v1", cfg.ServerURL)
		}
	})

	t.Run("env wins over default", func(t *testing.T) {
		os.Setenv("SPOOLMAN_URL", "http://env.host")
		defer os.Unsetenv("SPOOLMAN_URL")
		cfg, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Source != "env" {
			t.Errorf("source = %q; want env", cfg.Source)
		}
		if cfg.ServerURL != "http://env.host/api/v1" {
			t.Errorf("URL = %q; want http://env.host/api/v1", cfg.ServerURL)
		}
	})

	t.Run("default when nothing set", func(t *testing.T) {
		cfg, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		want := DefaultServerURL + DefaultAPIPath
		if cfg.ServerURL != want {
			t.Errorf("URL = %q; want %q", cfg.ServerURL, want)
		}
	})

	t.Run("timeout from env", func(t *testing.T) {
		os.Setenv("SPOOLMAN_TIMEOUT", "30s")
		defer os.Unsetenv("SPOOLMAN_TIMEOUT")
		cfg, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Timeout != 30*time.Second {
			t.Errorf("timeout = %v; want 30s", cfg.Timeout)
		}
	})

	t.Run("insecure from env", func(t *testing.T) {
		os.Setenv("SPOOLMAN_INSECURE", "1")
		defer os.Unsetenv("SPOOLMAN_INSECURE")
		cfg, err := Load("")
		if err != nil {
			t.Fatal(err)
		}
		if !cfg.Insecure {
			t.Error("expected Insecure=true")
		}
	})

	t.Run("bad timeout rejected", func(t *testing.T) {
		os.Setenv("SPOOLMAN_TIMEOUT", "notaduration")
		defer os.Unsetenv("SPOOLMAN_TIMEOUT")
		_, err := Load("")
		if err == nil {
			t.Error("expected error for bad timeout")
		}
	})
}
