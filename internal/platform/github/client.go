package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gogithub "github.com/google/go-github/v69/github"
)

// NewAppClient creates a GitHub App-level client authenticated with the App's
// private key (PEM-encoded RSA key). Use this for App-level API calls such as
// listing installations or getting installation info.
func NewAppClient(appID int64, privateKey []byte) (*gogithub.Client, error) {
	atr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("create app transport: %w", err)
	}
	httpClient := &http.Client{Transport: atr}
	return gogithub.NewClient(httpClient), nil
}

// NewInstallationClient creates a GitHub installation-level client that
// automatically manages token exchange and refresh for the given installation.
// privateKey must be a PEM-encoded RSA private key.
func NewInstallationClient(appID, installationID int64, privateKey []byte) (*gogithub.Client, error) {
	itr, err := ghinstallation.New(http.DefaultTransport, appID, installationID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("create installation transport: %w", err)
	}
	httpClient := &http.Client{Transport: itr}
	return gogithub.NewClient(httpClient), nil
}

// ListOrgRepos lists all repositories for the given organization, paginating
// through all pages. Returns the combined list of all repositories.
func ListOrgRepos(ctx context.Context, client *gogithub.Client, org string) ([]*gogithub.Repository, error) {
	var allRepos []*gogithub.Repository

	opts := &gogithub.RepositoryListByOrgOptions{
		ListOptions: gogithub.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("list org repos (page %d): %w", opts.Page, err)
		}
		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}
