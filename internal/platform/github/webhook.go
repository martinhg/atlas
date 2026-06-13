package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifyWebhookSignature verifies that the given GitHub webhook payload matches
// the provided HMAC-SHA256 signature header.
//
// sigHeader is the value of the X-Hub-Signature-256 header (e.g. "sha256=abc123...").
// Returns false if sigHeader is empty or missing the "sha256=" prefix.
// Uses constant-time comparison to prevent timing attacks.
func VerifyWebhookSignature(secret string, payload []byte, sigHeader string) bool {
	if sigHeader == "" || !strings.HasPrefix(sigHeader, "sha256=") {
		return false
	}

	hexSig := strings.TrimPrefix(sigHeader, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	actual, err := hex.DecodeString(hexSig)
	if err != nil {
		return false
	}

	return hmac.Equal(expected, actual)
}
