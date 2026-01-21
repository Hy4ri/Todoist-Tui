package api

import "fmt"

// GetLabels returns all personal labels.
func (c *Client) GetLabels() ([]Label, error) {
	var labels []Label
	if err := c.Get("/labels", &labels); err != nil {
		return nil, fmt.Errorf("failed to get labels: %w", err)
	}
	return labels, nil
}

// GetLabel returns a single label by ID.
func (c *Client) GetLabel(id string) (*Label, error) {
	var label Label
	if err := c.Get("/labels/"+id, &label); err != nil {
		return nil, fmt.Errorf("failed to get label %s: %w", id, err)
	}
	return &label, nil
}

// CreateLabel creates a new personal label.
func (c *Client) CreateLabel(req CreateLabelRequest) (*Label, error) {
	var label Label
	if err := c.Post("/labels", req, &label); err != nil {
		return nil, fmt.Errorf("failed to create label: %w", err)
	}
	return &label, nil
}

// UpdateLabel updates an existing label.
func (c *Client) UpdateLabel(id string, req UpdateLabelRequest) (*Label, error) {
	var label Label
	if err := c.Post("/labels/"+id, req, &label); err != nil {
		return nil, fmt.Errorf("failed to update label %s: %w", id, err)
	}
	return &label, nil
}

// DeleteLabel deletes a label.
func (c *Client) DeleteLabel(id string) error {
	if err := c.Delete("/labels/" + id); err != nil {
		return fmt.Errorf("failed to delete label %s: %w", id, err)
	}
	return nil
}
