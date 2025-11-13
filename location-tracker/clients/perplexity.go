/*
# Module: clients/perplexity.go
Perplexity AI API client for commercial real estate and context queries.

## Linked Modules
- [types/api_types](../types/api_types.go) - Perplexity API types

## Tags
api-client, perplexity, ai, llm

## Exports
PerplexityClient, NewPerplexityClient, ChatCompletion

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "clients/perplexity.go" ;
    code:description "Perplexity AI API client for commercial real estate and context queries" ;
    code:linksTo [
        code:name "types/api_types" ;
        code:path "../types/api_types.go" ;
        code:relationship "Perplexity API types"
    ] ;
    code:exports :PerplexityClient, :NewPerplexityClient, :ChatCompletion ;
    code:tags "api-client", "perplexity", "ai", "llm" .
<!-- End LinkedDoc RDF -->
*/
package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"location-tracker/types"
)

// PerplexityClient handles Perplexity AI API requests
type PerplexityClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewPerplexityClient creates a new Perplexity API client
func NewPerplexityClient(apiKey string) *PerplexityClient {
	return &PerplexityClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ChatCompletion sends a chat completion request to Perplexity API
func (c *PerplexityClient) ChatCompletion(model string, messages []types.PerplexityMessage) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("Perplexity API key not configured")
	}

	reqBody := types.PerplexityRequest{
		Model:    model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.perplexity.ai/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Perplexity API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Perplexity API error (status %d): %s", resp.StatusCode, string(body))
	}

	var perplexityResp types.PerplexityResponse
	if err := json.Unmarshal(body, &perplexityResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if perplexityResp.Error != nil {
		return "", fmt.Errorf("Perplexity API error: %s", perplexityResp.Error.Message)
	}

	if len(perplexityResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Perplexity")
	}

	return perplexityResp.Choices[0].Message.Content, nil
}
