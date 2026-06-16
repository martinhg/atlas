package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gogithub "github.com/google/go-github/v69/github"
)

// generateTestRSAKey creates a fresh RSA private key for use in tests.
func generateTestRSAKey(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(block)
}

func TestNewAppClient_success(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	client, err := NewAppClient(12345, privateKey)
	if err != nil {
		t.Fatalf("NewAppClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("NewAppClient returned nil client")
	}
}

func TestNewAppClient_invalidPrivateKey(t *testing.T) {
	_, err := NewAppClient(12345, []byte("not-a-valid-pem-key"))
	if err == nil {
		t.Error("expected error for invalid private key, got nil")
	}
}

func TestNewInstallationClient_success(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	client, err := NewInstallationClient(12345, 67890, privateKey)
	if err != nil {
		t.Fatalf("NewInstallationClient returned error: %v", err)
	}
	if client == nil {
		t.Fatal("NewInstallationClient returned nil client")
	}
}

func TestNewInstallationClient_invalidPrivateKey(t *testing.T) {
	_, err := NewInstallationClient(12345, 67890, []byte("not-valid-pem"))
	if err == nil {
		t.Error("expected error for invalid private key, got nil")
	}
}

func TestListInstallationRepos_paginates(t *testing.T) {
	page1 := []*gogithub.Repository{
		{ID: gogithub.Ptr(int64(1)), Name: gogithub.Ptr("repo-1")},
		{ID: gogithub.Ptr(int64(2)), Name: gogithub.Ptr("repo-2")},
	}
	page2 := []*gogithub.Repository{
		{ID: gogithub.Ptr(int64(3)), Name: gogithub.Ptr("repo-3")},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			w.Header().Set("Link", fmt.Sprintf(`<%s?page=2&per_page=100>; rel="next"`, "http://"+r.Host+r.URL.Path))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count":  3,
				"repositories": page1,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count":  3,
				"repositories": page2,
			})
		}
	}))
	defer server.Close()

	httpClient := &http.Client{}
	client := gogithub.NewClient(httpClient).WithAuthToken("test-token")

	serverURL := server.URL + "/"
	client, err := client.WithEnterpriseURLs(serverURL, serverURL)
	if err != nil {
		t.Fatalf("failed to set enterprise URLs: %v", err)
	}

	repos, err := ListInstallationRepos(context.Background(), client)
	if err != nil {
		t.Fatalf("ListInstallationRepos returned error: %v", err)
	}

	if len(repos) != 3 {
		t.Errorf("ListInstallationRepos returned %d repos, want 3", len(repos))
	}
}

func TestListInstallationRepos_empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_count":  0,
			"repositories": []*gogithub.Repository{},
		})
	}))
	defer server.Close()

	httpClient := &http.Client{}
	client := gogithub.NewClient(httpClient).WithAuthToken("test-token")

	serverURL := server.URL + "/"
	client, err := client.WithEnterpriseURLs(serverURL, serverURL)
	if err != nil {
		t.Fatalf("failed to set enterprise URLs: %v", err)
	}

	repos, err := ListInstallationRepos(context.Background(), client)
	if err != nil {
		t.Fatalf("ListInstallationRepos returned error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("ListInstallationRepos returned %d repos, want 0", len(repos))
	}
}
