// Package solid provides a Go client library for interacting with Solid Pods.
// It includes HTTP operations with DPoP token support and RDF serialization.
package solid

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Solid Pod HTTP client with DPoP authentication.
type Client struct {
	httpClient *http.Client
	dPopToken  string
	userAgent  string
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// NewClient creates a new Solid Pod client with the given DPoP token.
//
// The DPoP token should be obtained from the frontend after Solid authentication.
// This client validates the token and uses it for authenticated requests.
func NewClient(dPopToken string, opts ...ClientOption) (*Client, error) {
	if dPopToken == "" {
		return nil, fmt.Errorf("dPoP token is required")
	}

	client := &Client{
		dPopToken: dPopToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Solid-PoC/1.0",
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// GetResource fetches a resource from a Solid Pod.
//
// The URL should be a full Pod resource URL (e.g., https://alice.solid.net/private/data.ttl).
// Returns the resource data and content type.
func (c *Client) GetResource(ctx context.Context, resourceURL string) (data []byte, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add DPoP header
	req.Header.Set("Authorization", fmt.Sprintf("DPoP %s", c.dPopToken))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType = resp.Header.Get("Content-Type")
	return data, contentType, nil
}

// PutResource writes a resource to a Solid Pod.
//
// The URL should be a full Pod resource URL (e.g., https://alice.solid.net/private/data.ttl).
// The contentType should match the data format (e.g., "text/turtle", "application/ld+json").
func (c *Client) PutResource(ctx context.Context, resourceURL string, data []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, resourceURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add DPoP header
	req.Header.Set("Authorization", fmt.Sprintf("DPoP %s", c.dPopToken))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to write resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteResource deletes a resource from a Solid Pod.
func (c *Client) DeleteResource(ctx context.Context, resourceURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, resourceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add DPoP header
	req.Header.Set("Authorization", fmt.Sprintf("DPoP %s", c.dPopToken))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HeadResource checks if a resource exists and returns its metadata.
func (c *Client) HeadResource(ctx context.Context, resourceURL string) (exists bool, contentType string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, resourceURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add DPoP header
	req.Header.Set("Authorization", fmt.Sprintf("DPoP %s", c.dPopToken))
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to check resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	contentType = resp.Header.Get("Content-Type")
	return true, contentType, nil
}

// CreateContainer creates a new container (directory) in the Pod.
func (c *Client) CreateContainer(ctx context.Context, containerURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, containerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add DPoP header
	req.Header.Set("Authorization", fmt.Sprintf("DPoP %s", c.dPopToken))
	req.Header.Set("Content-Type", "text/turtle")
	req.Header.Set("Link", `<http://www.w3.org/ns/ldp#BasicContainer>; rel="type"`)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
