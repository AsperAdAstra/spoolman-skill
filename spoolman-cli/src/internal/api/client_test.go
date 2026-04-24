package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vibecoder/spoolctl/internal/config"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cfg := &config.Config{
		ServerURL: srv.URL,
		Timeout:   5 * time.Second,
	}
	c, err := NewClient(cfg, false)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}
	return c, srv
}

func TestGetVendor(t *testing.T) {
	fixture := `{"id":1,"registered":"2026-01-01T00:00:00Z","name":"Bambu Lab","extra":{}}`
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vendor/1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fixture))
	}))
	defer srv.Close()

	raw, err := c.GetVendor(1)
	if err != nil {
		t.Fatal(err)
	}
	var v Vendor
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if v.ID != 1 || v.Name != "Bambu Lab" {
		t.Errorf("unexpected vendor: %+v", v)
	}
}

func TestListVendors(t *testing.T) {
	fixture := `[{"id":1,"registered":"2026-01-01T00:00:00Z","name":"Bambu Lab","extra":{}}]`
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vendor" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fixture))
	}))
	defer srv.Close()

	raw, err := c.ListVendors("")
	if err != nil {
		t.Fatal(err)
	}
	var vendors []Vendor
	if err := json.Unmarshal(raw, &vendors); err != nil {
		t.Fatal(err)
	}
	if len(vendors) != 1 {
		t.Errorf("expected 1 vendor, got %d", len(vendors))
	}
}

func TestAPIError(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"Vendor not found"}`))
	}))
	defer srv.Close()

	_, err := c.GetVendor(999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 404 {
		t.Errorf("expected status 404, got %d", apiErr.Status)
	}
}

func TestCreateVendor(t *testing.T) {
	response := `{"id":2,"registered":"2026-01-01T00:00:00Z","name":"Polymaker","extra":{}}`
	var receivedBody map[string]interface{}
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(response))
	}))
	defer srv.Close()

	raw, err := c.CreateVendor(VendorCreate{Name: "Polymaker"})
	if err != nil {
		t.Fatal(err)
	}
	var v Vendor
	json.Unmarshal(raw, &v)
	if v.Name != "Polymaker" {
		t.Errorf("unexpected name: %s", v.Name)
	}
	if receivedBody["name"] != "Polymaker" {
		t.Errorf("request body did not include name")
	}
}

func TestGetHealth(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer srv.Close()

	hc, err := c.GetHealth()
	if err != nil {
		t.Fatal(err)
	}
	if hc.Status != "healthy" {
		t.Errorf("unexpected status: %s", hc.Status)
	}
}

func TestGetFilament(t *testing.T) {
	fixture := `{"id":1,"registered":"2026-01-01T00:00:00Z","density":1.24,"diameter":1.75,"extra":{}}`
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fixture))
	}))
	defer srv.Close()

	raw, err := c.GetFilament(1)
	if err != nil {
		t.Fatal(err)
	}
	var f Filament
	json.Unmarshal(raw, &f)
	if f.Density != 1.24 || f.Diameter != 1.75 {
		t.Errorf("unexpected filament: %+v", f)
	}
}

func TestDeleteSpool(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := c.DeleteSpool(1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUseSpool(t *testing.T) {
	response := `{"id":1,"registered":"2026-01-01T00:00:00Z","filament":{"id":1,"registered":"2026-01-01T00:00:00Z","density":1.24,"diameter":1.75,"extra":{}},"used_weight":42,"used_length":0,"archived":false,"extra":{}}`
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer srv.Close()

	w := 42.0
	raw, err := c.UseSpool(1, SpoolUse{UseWeight: &w})
	if err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		t.Error("expected non-nil response")
	}
}
