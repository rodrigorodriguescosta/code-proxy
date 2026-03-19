package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// GeneratePKCE generates a code_verifier and code_challenge for OAuth PKCE (RFC 7636)
func GeneratePKCE() (verifier, challenge string, err error) {
	// Code verifier: 128 caracteres URL-safe random
	b := make([]byte, 96) // 96 bytes → 128 chars base64url
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64URLEncode(b)

	// Code challenge: SHA256 do verifier, base64url encoded
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64URLEncode(hash[:])

	return verifier, challenge, nil
}

// GenerateState generates a random state string for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}

// base64URLEncode base64-encodes in a URL-safe way without padding
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
