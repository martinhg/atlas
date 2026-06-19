package vuln

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// testOSVClient builds an OSVClient pointed at a test server base URL.
func testOSVClient(base string) *OSVClient {
	return &OSVClient{httpClient: http.DefaultClient, baseURL: base}
}

// TestQueryBatch_postsExpectedBody asserts the querybatch request body matches
// the OSV contract: { "queries": [ { "package": { "name", "ecosystem" } } ] }.
func TestQueryBatch_postsExpectedBody(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/querybatch" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []any{
				map[string]any{"vulns": []any{}},
				map[string]any{"vulns": []any{}},
			},
		})
	}))
	defer srv.Close()

	c := testOSVClient(srv.URL)
	pkgs := []DepPair{
		{DepID: uuid.New(), Ecosystem: "npm", Name: "express", Version: "4.0.0"},
		{DepID: uuid.New(), Ecosystem: "npm", Name: "lodash", Version: "4.17.0"},
	}
	if _, err := c.QueryBatch(context.Background(), pkgs); err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}

	queries, ok := gotBody["queries"].([]any)
	if !ok || len(queries) != 2 {
		t.Fatalf("expected 2 queries in body, got %#v", gotBody["queries"])
	}
	pkg := queries[0].(map[string]any)["package"].(map[string]any)
	if pkg["name"] != "express" || pkg["ecosystem"] != "npm" {
		t.Errorf("unexpected first query package: %#v", pkg)
	}
}

// TestQueryBatch_chunksAt100 asserts 101 packages produce 2 querybatch requests.
func TestQueryBatch_chunksAt100(t *testing.T) {
	var batchCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/querybatch" {
			http.NotFound(w, r)
			return
		}
		batchCalls++
		var body struct {
			Queries []json.RawMessage `json:"queries"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		results := make([]map[string]any, len(body.Queries))
		for i := range results {
			results[i] = map[string]any{"vulns": []any{}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	defer srv.Close()

	c := testOSVClient(srv.URL)
	pkgs := make([]DepPair, 101)
	for i := range pkgs {
		pkgs[i] = DepPair{DepID: uuid.New(), Ecosystem: "npm", Name: fmt.Sprintf("pkg-%d", i), Version: "1.0.0"}
	}
	results, err := c.QueryBatch(context.Background(), pkgs)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if batchCalls != 2 {
		t.Errorf("expected 2 querybatch requests for 101 packages, got %d", batchCalls)
	}
	if len(results) != 101 {
		t.Errorf("expected 101 results aligned to input, got %d", len(results))
	}
}

// TestQueryBatch_emptyInput returns no results without hitting the network.
func TestQueryBatch_emptyInput(t *testing.T) {
	c := testOSVClient("http://invalid.invalid")
	results, err := c.QueryBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestQueryBatch_serverError returns an error on 5xx and 429.
func TestQueryBatch_serverError(t *testing.T) {
	for _, code := range []int{http.StatusInternalServerError, http.StatusTooManyRequests} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", code)
		}))
		c := testOSVClient(srv.URL)
		_, err := c.QueryBatch(context.Background(), []DepPair{
			{DepID: uuid.New(), Ecosystem: "npm", Name: "x", Version: "1.0.0"},
		})
		if err == nil {
			t.Errorf("expected error for status %d, got nil", code)
		}
		srv.Close()
	}
}

// TestQueryBatch_hydratesVulnDetails asserts that vuln IDs from querybatch are
// hydrated via GET /v1/vulns/{id} into full OSV vuln records, aligned per package.
func TestQueryBatch_hydratesVulnDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/querybatch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{"vulns": []any{map[string]any{"id": "GHSA-aaaa"}}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/vulns/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      id,
				"aliases": []string{"CVE-2021-1234"},
				"summary": "test vuln",
				"severity": []any{
					map[string]any{"type": "CVSS_V3", "score": "9.8"},
				},
				"affected": []any{
					map[string]any{
						"package": map[string]any{"ecosystem": "npm", "name": "express"},
						"ranges": []any{
							map[string]any{
								"type": "SEMVER",
								"events": []any{
									map[string]any{"introduced": "0"},
									map[string]any{"fixed": "4.18.0"},
								},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := testOSVClient(srv.URL)
	depID := uuid.New()
	results, err := c.QueryBatch(context.Background(), []DepPair{
		{DepID: depID, Ecosystem: "npm", Name: "express", Version: "4.17.0"},
	})
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Dep.DepID != depID {
		t.Errorf("dep id mismatch: got %s want %s", results[0].Dep.DepID, depID)
	}
	if len(results[0].Vulns) != 1 {
		t.Fatalf("expected 1 vuln, got %d", len(results[0].Vulns))
	}
	v := results[0].Vulns[0]
	if v.ID != "GHSA-aaaa" {
		t.Errorf("vuln id = %q", v.ID)
	}
	if len(v.Aliases) != 1 || v.Aliases[0] != "CVE-2021-1234" {
		t.Errorf("aliases = %#v", v.Aliases)
	}
	if len(v.Severity) != 1 || v.Severity[0].Type != "CVSS_V3" || v.Severity[0].Score != "9.8" {
		t.Errorf("severity = %#v", v.Severity)
	}
	if len(v.Affected) != 1 || len(v.Affected[0].Ranges) != 1 {
		t.Errorf("affected = %#v", v.Affected)
	}
}

// TestQueryBatch_dedupesHydration asserts the same vuln ID across packages is
// fetched once, not per occurrence.
func TestQueryBatch_dedupesHydration(t *testing.T) {
	var vulnFetches int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/querybatch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []any{
					map[string]any{"vulns": []any{map[string]any{"id": "GHSA-shared"}}},
					map[string]any{"vulns": []any{map[string]any{"id": "GHSA-shared"}}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/vulns/"):
			vulnFetches++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "GHSA-shared"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := testOSVClient(srv.URL)
	results, err := c.QueryBatch(context.Background(), []DepPair{
		{DepID: uuid.New(), Ecosystem: "npm", Name: "a", Version: "1.0.0"},
		{DepID: uuid.New(), Ecosystem: "npm", Name: "b", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("QueryBatch: %v", err)
	}
	if vulnFetches != 1 {
		t.Errorf("expected 1 hydration fetch for shared id, got %d", vulnFetches)
	}
	if len(results) != 2 || len(results[0].Vulns) != 1 || len(results[1].Vulns) != 1 {
		t.Errorf("expected both packages to carry the shared vuln, got %#v", results)
	}
}
