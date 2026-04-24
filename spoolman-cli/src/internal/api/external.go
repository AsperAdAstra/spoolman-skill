package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ListExternalFilaments fetches /external/filament from the Spoolman server.
func (c *Client) ListExternalFilaments(manufacturer, material string, diameter float64) (json.RawMessage, error) {
	q := url.Values{}
	if manufacturer != "" {
		q.Set("manufacturer", manufacturer)
	}
	if material != "" {
		q.Set("material", material)
	}
	if diameter > 0 {
		q.Set("diameter", fmt.Sprintf("%.2f", diameter))
	}
	return c.RawGet("/external/filament", q)
}

// ListExternalMaterials fetches /external/material from the Spoolman server.
func (c *Client) ListExternalMaterials() (json.RawMessage, error) {
	return c.RawGet("/external/material", nil)
}

// GetInfo fetches /info.
func (c *Client) GetInfo() (*Info, error) {
	var info Info
	if err := c.get("/info", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetHealth fetches /health.
func (c *Client) GetHealth() (*HealthCheck, error) {
	var hc HealthCheck
	if err := c.get("/health", nil, &hc); err != nil {
		return nil, err
	}
	return &hc, nil
}

// ListVendorsTyped returns decoded vendor slice.
func (c *Client) ListVendorsTyped() ([]Vendor, error) {
	var vendors []Vendor
	if err := c.get("/vendor", nil, &vendors); err != nil {
		return nil, err
	}
	return vendors, nil
}

// ListFilamentsTyped returns decoded filament slice.
func (c *Client) ListFilamentsTyped() ([]Filament, error) {
	var filaments []Filament
	if err := c.get("/filament", nil, &filaments); err != nil {
		return nil, err
	}
	return filaments, nil
}
