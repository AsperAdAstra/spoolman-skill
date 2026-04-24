package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) ListSpools(filamentID string, archived bool) (json.RawMessage, error) {
	q := url.Values{}
	if filamentID != "" {
		q.Set("filament_id", filamentID)
	}
	if archived {
		q.Set("allow_archived", "true")
	}
	return c.RawGet("/spool", q)
}

func (c *Client) GetSpool(id int) (json.RawMessage, error) {
	return c.RawGet(fmt.Sprintf("/spool/%d", id), nil)
}

func (c *Client) CreateSpool(body SpoolCreate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.post("/spool", body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) UpdateSpool(id int, body SpoolUpdate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.patch(fmt.Sprintf("/spool/%d", id), body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) DeleteSpool(id int) error {
	return c.delete(fmt.Sprintf("/spool/%d", id))
}

func (c *Client) UseSpool(id int, body SpoolUse) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.put(fmt.Sprintf("/spool/%d/use", id), body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) MeasureSpool(id int, body SpoolMeasure) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.put(fmt.Sprintf("/spool/%d/measure", id), body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetSpoolTyped returns a decoded Spool struct (used internally for context).
func (c *Client) GetSpoolTyped(id int) (*Spool, error) {
	var s Spool
	if err := c.get(fmt.Sprintf("/spool/%d", id), nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSpoolsTyped returns decoded Spool slice (used internally for context).
func (c *Client) ListSpoolsTyped(filamentID string, archived bool) ([]Spool, error) {
	q := url.Values{}
	if filamentID != "" {
		q.Set("filament_id", filamentID)
	}
	if archived {
		q.Set("allow_archived", "true")
	}
	var spools []Spool
	if err := c.get("/spool", q, &spools); err != nil {
		return nil, err
	}
	return spools, nil
}
