package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// computeSignature is a test helper that computes the expected sha256 HMAC signature.
func computeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		payload   []byte
		sigHeader string
		want      bool
	}{
		{
			name:      "valid signature",
			secret:    "my-secret",
			payload:   []byte(`{"action":"created"}`),
			sigHeader: computeSignature("my-secret", []byte(`{"action":"created"}`)),
			want:      true,
		},
		{
			name:      "wrong secret produces false",
			secret:    "my-secret",
			payload:   []byte(`{"action":"created"}`),
			sigHeader: computeSignature("wrong-secret", []byte(`{"action":"created"}`)),
			want:      false,
		},
		{
			name:      "empty sigHeader returns false",
			secret:    "my-secret",
			payload:   []byte(`{"action":"created"}`),
			sigHeader: "",
			want:      false,
		},
		{
			name:      "sigHeader without sha256= prefix returns false",
			secret:    "my-secret",
			payload:   []byte(`{"action":"created"}`),
			sigHeader: hex.EncodeToString([]byte("some-hmac")),
			want:      false,
		},
		{
			name:      "empty payload with valid signature is true",
			secret:    "my-secret",
			payload:   []byte{},
			sigHeader: computeSignature("my-secret", []byte{}),
			want:      true,
		},
		{
			name:      "tampered payload returns false",
			secret:    "my-secret",
			payload:   []byte(`{"action":"tampered"}`),
			sigHeader: computeSignature("my-secret", []byte(`{"action":"original"}`)),
			want:      false,
		},
		{
			name:      "empty secret with valid signature is true",
			secret:    "",
			payload:   []byte(`{"action":"created"}`),
			sigHeader: computeSignature("", []byte(`{"action":"created"}`)),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyWebhookSignature(tt.secret, tt.payload, tt.sigHeader)
			if got != tt.want {
				t.Errorf("VerifyWebhookSignature(%q, payload, %q) = %v, want %v",
					tt.secret, tt.sigHeader, got, tt.want)
			}
		})
	}
}
