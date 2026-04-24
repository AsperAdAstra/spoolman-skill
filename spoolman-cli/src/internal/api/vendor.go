package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) ListVendors(name string) (json.RawMessage, error) {
	q := url.Values{}
	if name != "" {
		q.Set("name", name)
	}
	return c.RawGet("/vendor", q)
}

func (c *Client) GetVendor(id int) (json.RawMessage, error) {
	return c.RawGet(fmt.Sprintf("/vendor/%d", id), nil)
}

func (c *Client) CreateVendor(body VendorCreate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.post("/vendor", body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) UpdateVendor(id int, body VendorUpdate) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.patch(fmt.Sprintf("/vendor/%d", id), body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) DeleteVendor(id int) error {
	return c.delete(fmt.Sprintf("/vendor/%d", id))
}
