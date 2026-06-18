package vuln

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// osvChunkSize is the maximum number of packages per querybatch request.
	// OSV.dev rejects larger batches, so callers MUST chunk.
	osvChunkSize = 100
	// osvDefaultBaseURL is the OSV.dev API root.
	osvDefaultBaseURL = "https://api.osv.dev"
)

// OSVClient queries the OSV.dev API for known vulnerabilities.
//
// The OSV querybatch endpoint returns only vulnerability IDs per package, so
// QueryBatch performs a second hydration step (GET /v1/vulns/{id}) to fetch the
// full advisory record (aliases, severity, affected ranges) needed downstream.
type OSVClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewOSVClient constructs an OSVClient pointed at the public OSV.dev API.
func NewOSVClient() *OSVClient {
	return &OSVClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    osvDefaultBaseURL,
	}
}

// OSVResult holds the vulnerabilities matched for a single queried dependency.
// Results are index-aligned with the packages slice passed to QueryBatch.
type OSVResult struct {
	Dep   DepPair
	Vulns []OSVVuln
}

// OSVVuln is the parsed OSV advisory record returned by GET /v1/vulns/{id}.
// Severity/CVSS derivation and affected-range extraction are performed by the
// service layer, not here.
type OSVVuln struct {
	ID        string        `json:"id"`
	Aliases   []string      `json:"aliases"`
	Summary   string        `json:"summary"`
	Details   string        `json:"details"`
	Published *time.Time    `json:"published"`
	Modified  *time.Time    `json:"modified"`
	Severity  []OSVSeverity `json:"severity"`
	Affected  []OSVAffected `json:"affected"`
}

// OSVSeverity is a single severity entry; Type is e.g. "CVSS_V3" and Score is
// the OSV-provided score (numeric string or CVSS vector).
type OSVSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// OSVAffected describes one affected package and its version ranges.
type OSVAffected struct {
	Package OSVPackage `json:"package"`
	Ranges  []OSVRange `json:"ranges"`
}

// OSVPackage identifies a package within an OSV advisory.
type OSVPackage struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// OSVRange is a version range; Type is e.g. "SEMVER" or "ECOSYSTEM".
type OSVRange struct {
	Type   string     `json:"type"`
	Events []OSVEvent `json:"events"`
}

// OSVEvent is a single range boundary; exactly one field is set per event.
type OSVEvent struct {
	Introduced string `json:"introduced"`
	Fixed      string `json:"fixed"`
}

// osvBatchRequest is the querybatch request body.
type osvBatchRequest struct {
	Queries []osvBatchQuery `json:"queries"`
}

type osvBatchQuery struct {
	Package OSVPackage `json:"package"`
}

// osvBatchResponse is the querybatch response: results are index-aligned with
// the request queries and carry only vulnerability IDs.
type osvBatchResponse struct {
	Results []struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	} `json:"results"`
}

// QueryBatch queries OSV.dev for vulnerabilities affecting the given packages.
// It chunks requests at osvChunkSize, then hydrates each unique vulnerability ID
// into a full OSVVuln record. Returned results are index-aligned with packages.
func (c *OSVClient) QueryBatch(ctx context.Context, packages []DepPair) ([]OSVResult, error) {
	if len(packages) == 0 {
		return nil, nil
	}

	results := make([]OSVResult, len(packages))
	for i := range packages {
		results[i] = OSVResult{Dep: packages[i]}
	}

	// Phase 1: batch query for vulnerability IDs per package.
	idsByIndex := make([][]string, len(packages))
	for start := 0; start < len(packages); start += osvChunkSize {
		end := min(start+osvChunkSize, len(packages))
		chunk := packages[start:end]

		resp, err := c.queryChunk(ctx, chunk)
		if err != nil {
			return nil, err
		}
		for i, res := range resp.Results {
			ids := make([]string, 0, len(res.Vulns))
			for _, v := range res.Vulns {
				ids = append(ids, v.ID)
			}
			idsByIndex[start+i] = ids
		}
	}

	// Phase 2: hydrate each unique vulnerability ID once.
	cache := make(map[string]OSVVuln)
	for i, ids := range idsByIndex {
		for _, id := range ids {
			vuln, ok := cache[id]
			if !ok {
				fetched, err := c.fetchVuln(ctx, id)
				if err != nil {
					return nil, err
				}
				vuln = fetched
				cache[id] = vuln
			}
			results[i].Vulns = append(results[i].Vulns, vuln)
		}
	}

	return results, nil
}

// queryChunk POSTs a single querybatch request for up to osvChunkSize packages.
func (c *OSVClient) queryChunk(ctx context.Context, chunk []DepPair) (*osvBatchResponse, error) {
	reqBody := osvBatchRequest{Queries: make([]osvBatchQuery, len(chunk))}
	for i, p := range chunk {
		reqBody.Queries[i] = osvBatchQuery{Package: OSVPackage{Ecosystem: p.Ecosystem, Name: p.Name}}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/querybatch", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv querybatch: unexpected status %d", resp.StatusCode)
	}

	var out osvBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// fetchVuln hydrates a single vulnerability by ID via GET /v1/vulns/{id}.
func (c *OSVClient) fetchVuln(ctx context.Context, id string) (OSVVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/vulns/"+id, nil)
	if err != nil {
		return OSVVuln{}, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return OSVVuln{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return OSVVuln{}, fmt.Errorf("osv vulns/%s: unexpected status %d: %s", id, resp.StatusCode, body)
	}

	var v OSVVuln
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return OSVVuln{}, err
	}
	return v, nil
}
