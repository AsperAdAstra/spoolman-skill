package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) ListFilaments(vendorID string, material string) (json.RawMessage, error) {
	q := url.Values{}
	if vendorID != "" {
		q.Set("vendor_id", vendorID)
	}
	if material != "" {
		q.Set("material", material)
	}
	return c.RawGet("/filament", q)
}

func (c *Client) GetFilament(id int) (json.RawMessage, error) {
	return c.RawGet(fmt.Sprintf("/filament/%d", id), nil)
}

func (c *Client) CreateFilament(body FilamentCreate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.post("/filament", body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) UpdateFilament(id int, body FilamentUpdate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.patch(fmt.Sprintf("/filament/%d", id), body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) DeleteFilament(id int) error {
	return c.delete(fmt.Sprintf("/filament/%d", id))
}
