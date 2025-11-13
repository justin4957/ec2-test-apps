/*
# Module: types/api_types.go
External API request and response data structures.

## Linked Modules
(None - types package has no dependencies)

## Tags
data-types, api-client

## Exports
PerplexityRequest, PerplexityMessage, PerplexityResponse, TwilioWebhook

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "types/api_types.go" ;
    code:description "External API request and response data structures" ;
    code:exports :PerplexityRequest, :PerplexityMessage, :PerplexityResponse, :TwilioWebhook ;
    code:tags "data-types", "api-client" .
<!-- End LinkedDoc RDF -->
*/
package types

// PerplexityRequest represents a request to Perplexity API
type PerplexityRequest struct {
	Model    string              `json:"model"`
	Messages []PerplexityMessage `json:"messages"`
}

// PerplexityMessage represents a message in Perplexity API format
type PerplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PerplexityResponse represents response from Perplexity API
type PerplexityResponse struct {
	Choices []struct {
		Message PerplexityMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// TwilioWebhook represents incoming SMS data from Twilio
type TwilioWebhook struct {
	MessageSid string `json:"MessageSid"`
	Body       string `json:"Body"`
	From       string `json:"From"`
	To         string `json:"To"`
}
