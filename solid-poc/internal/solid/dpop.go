package solid

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// DPoPTokenClaims represents the claims in a DPoP token JWT
type DPoPTokenClaims struct {
	JTI string `json:"jti"` // Unique identifier
	HTM string `json:"htm"` // HTTP method
	HTU string `json:"htu"` // HTTP URI
	IAT int64  `json:"iat"` // Issued at
}

// ValidateDPoPToken validates a DPoP JWT token structure.
// Note: Full cryptographic validation would require the public key from JWK header.
// This is a simplified validation for PoC purposes.
func ValidateDPoPToken(token string) (bool, error) {
	if token == "" {
		return false, fmt.Errorf("token is empty")
	}

	// JWT format: header.payload.signature
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	// Decode header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false, fmt.Errorf("failed to parse header: %w", err)
	}

	// Check for DPoP-specific header fields
	typ, ok := header["typ"].(string)
	if !ok || typ != "dpop+jwt" {
		return false, fmt.Errorf("invalid typ header: expected 'dpop+jwt', got '%v'", typ)
	}

	// Check for JWK (JSON Web Key) in header
	if _, ok := header["jwk"]; !ok {
		return false, fmt.Errorf("missing jwk in header")
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims DPoPTokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return false, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Basic claims validation
	if claims.JTI == "" {
		return false, fmt.Errorf("missing jti claim")
	}
	if claims.HTM == "" {
		return false, fmt.Errorf("missing htm claim")
	}
	if claims.HTU == "" {
		return false, fmt.Errorf("missing htu claim")
	}

	// Note: In production, you would:
	// 1. Verify the signature using the public key from JWK header
	// 2. Check token expiration (iat + reasonable time window)
	// 3. Verify htm/htu match the actual HTTP request
	// 4. Check for token replay (jti should be unique)

	return true, nil
}

// ExtractDPoPClaims extracts claims from a DPoP token without full validation.
// Useful for debugging and logging.
func ExtractDPoPClaims(token string) (*DPoPTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims DPoPTokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}
