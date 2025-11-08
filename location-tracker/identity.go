package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UserMetadata contains information used to generate anonymous IDs
type UserMetadata struct {
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Timestamp    time.Time `json:"timestamp"`
	SessionToken string    `json:"session_token,omitempty"`
}

// UserIdentityManager handles anonymous ID generation and reversal
type UserIdentityManager struct {
	encryptionKey []byte // 32-byte AES-256 key
}

// NewUserIdentityManager creates a new identity manager with encryption key
func NewUserIdentityManager(key []byte) (*UserIdentityManager, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256")
	}
	return &UserIdentityManager{encryptionKey: key}, nil
}

// GenerateAnonymousID creates a reversible anonymous ID from request metadata
func (uim *UserIdentityManager) GenerateAnonymousID(r *http.Request) (hash string, encryptedMetadata string, err error) {
	// Collect metadata
	metadata := UserMetadata{
		IPAddress:    getClientIP(r),
		UserAgent:    r.UserAgent(),
		Timestamp:    time.Now(),
		SessionToken: r.Header.Get("X-Session-Token"),
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Encrypt metadata with AES-256-GCM
	encryptedData, err := uim.encrypt(metadataJSON)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	// Base64 encode encrypted data
	encryptedMetadata = base64.StdEncoding.EncodeToString(encryptedData)

	// Generate display hash (first 12 chars of SHA-256)
	hashBytes := sha256.Sum256([]byte(encryptedMetadata))
	hash = fmt.Sprintf("user_%x", hashBytes[:6]) // 12 hex chars from 6 bytes

	return hash, encryptedMetadata, nil
}

// ReverseHash decrypts metadata to reveal original user information (admin only)
func (uim *UserIdentityManager) ReverseHash(encryptedMetadata string) (*UserMetadata, error) {
	// Decode base64
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decrypt AES
	decryptedJSON, err := uim.decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	// Parse metadata
	var metadata UserMetadata
	if err := json.Unmarshal(decryptedJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// encrypt encrypts data using AES-256-GCM
func (uim *UserIdentityManager) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(uim.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM
func (uim *UserIdentityManager) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(uim.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// getClientIP extracts the real client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take first IP if multiple
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}
