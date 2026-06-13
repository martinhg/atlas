package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-jwt-secret-value"

func TestIssueTokenPair_success(t *testing.T) {
	userID := uuid.New()

	pair, err := IssueTokenPair(testSecret, userID)
	if err != nil {
		t.Fatalf("IssueTokenPair returned error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}

	// Round-trip: validate the access token and check UserID matches.
	claims, err := ValidateToken(testSecret, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("got UserID %v, want %v", claims.UserID, userID)
	}
}

func TestIssueTokenPair_differentUsers(t *testing.T) {
	idA := uuid.New()
	idB := uuid.New()

	pairA, err := IssueTokenPair(testSecret, idA)
	if err != nil {
		t.Fatalf("IssueTokenPair(A) error: %v", err)
	}
	pairB, err := IssueTokenPair(testSecret, idB)
	if err != nil {
		t.Fatalf("IssueTokenPair(B) error: %v", err)
	}

	if pairA.AccessToken == pairB.AccessToken {
		t.Error("different users produced identical access tokens")
	}
	if pairA.RefreshToken == pairB.RefreshToken {
		t.Error("different users produced identical refresh tokens")
	}

	// Validate each token carries the right user.
	claimsA, _ := ValidateToken(testSecret, pairA.AccessToken)
	claimsB, _ := ValidateToken(testSecret, pairB.AccessToken)

	if claimsA.UserID != idA {
		t.Errorf("token A: got UserID %v, want %v", claimsA.UserID, idA)
	}
	if claimsB.UserID != idB {
		t.Errorf("token B: got UserID %v, want %v", claimsB.UserID, idB)
	}
}

func TestValidateToken_validToken(t *testing.T) {
	userID := uuid.New()
	pair, err := IssueTokenPair(testSecret, userID)
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}

	claims, err := ValidateToken(testSecret, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("got UserID %v, want %v", claims.UserID, userID)
	}
}

func TestValidateToken_invalidSecret(t *testing.T) {
	userID := uuid.New()
	pair, err := IssueTokenPair(testSecret, userID)
	if err != nil {
		t.Fatalf("IssueTokenPair error: %v", err)
	}

	_, err = ValidateToken("wrong-secret", pair.AccessToken)
	if err == nil {
		t.Error("expected error for wrong secret, got nil")
	}
}

func TestValidateToken_malformedToken(t *testing.T) {
	_, err := ValidateToken(testSecret, "this.is.garbage")
	if err == nil {
		t.Error("expected error for malformed token, got nil")
	}
}

func TestValidateToken_expiredToken(t *testing.T) {
	userID := uuid.New()

	// Build a token that is already expired by crafting claims directly.
	past := time.Now().Add(-1 * time.Hour)
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(past.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(past),
		},
		UserID: userID,
	}
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	_, err = ValidateToken(testSecret, tokenStr)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}
