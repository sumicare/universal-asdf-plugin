//
// Copyright (c) 2025 Sumicare
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// API configuration constants.
const (
	// APIVersion is the GitHub REST API version header value.
	APIVersion = "2022-11-28"

	// httpTimeout is the default timeout for HTTP requests.
	httpTimeout = 30 * time.Second
)

// Sentinel errors for GitHub API operations.
var (
	// ErrInvalidURL indicates the provided URL is not a valid GitHub repository URL.
	ErrInvalidURL = errors.New("invalid GitHub repository URL")

	// ErrHTTPRequest indicates an HTTP request to the GitHub API failed.
	ErrHTTPRequest = errors.New("HTTP request failed")
)

type (
	// HTTPClient interface for HTTP operations (allows mocking).
	HTTPClient interface {
		// Do sends an HTTP request and returns an HTTP response.
		Do(req *http.Request) (*http.Response, error)
	}

	// Client provides methods to interact with the GitHub REST API.
	Client struct {
		httpClient HTTPClient
		apiURL     string
		authToken  string
	}

	// TagResponse represents a tag from the GitHub API.
	TagResponse struct {
		Ref string `json:"ref"`
	}

	// ReleaseResponse represents a release from the GitHub API.
	ReleaseResponse struct {
		TagName string `json:"tag_name"`
	}
)

// NewClient creates a new GitHub API client with default settings.
// It automatically uses GITHUB_TOKEN or GITHUB_API_TOKEN environment variable if set.
func NewClient() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_API_TOKEN")
	}

	return &Client{
		httpClient: &http.Client{Timeout: httpTimeout},
		apiURL:     "https://api.github.com",
		authToken:  token,
	}
}

// NewClientWithHTTP creates a new GitHub client with a custom HTTP client.
func NewClientWithHTTP(httpClient HTTPClient, apiURL string) *Client {
	return &Client{
		httpClient: httpClient,
		apiURL:     apiURL,
		authToken:  os.Getenv("GITHUB_TOKEN"),
	}
}

// NewClientWithToken creates a new GitHub client with explicit token.
func NewClientWithToken(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: httpTimeout},
		apiURL:     "https://api.github.com",
		authToken:  token,
	}
}

// SetToken sets the authentication token.
func (client *Client) SetToken(token string) {
	client.authToken = token
}

// GetOwnerRepo extracts owner and repository name from a GitHub URL.
func GetOwnerRepo(url string) (string, string, error) {
	const expectedParts = 2

	cleaned := strings.Replace(url, "git@github.com:", "", 1)

	cleaned = strings.Replace(cleaned, "https://github.com/", "", 1)

	parts := strings.Split(cleaned, "/")
	if len(parts) != expectedParts {
		return "", "", ErrInvalidURL
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")

	return owner, repo, nil
}

// GetTags fetches all tags from a GitHub repository using the API.
func (client *Client) GetTags(ctx context.Context, repoURL string) ([]string, error) {
	owner, repo, err := GetOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/git/refs/tags", client.apiURL, owner, repo)

	var tags []TagResponse
	if err := client.fetchJSON(ctx, url, &tags); err != nil {
		return nil, fmt.Errorf("fetching tags: %w", err)
	}

	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		version := strings.TrimPrefix(tag.Ref, "refs/tags/")

		versions = append(versions, version)
	}

	return versions, nil
}

// GetReleases fetches all releases from a GitHub repository.
func (client *Client) GetReleases(ctx context.Context, repoURL string) ([]string, error) {
	owner, repo, err := GetOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100", client.apiURL, owner, repo)

	var releases []ReleaseResponse
	if err := client.fetchJSON(ctx, url, &releases); err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}

	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		versions = append(versions, release.TagName)
	}

	return versions, nil
}

// fetchJSON fetches JSON from a URL and decodes it into the result.
func (client *Client) fetchJSON(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-Github-Api-Version", APIVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	if client.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+client.authToken)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%w: %d (failed to read body: %w)", ErrHTTPRequest, resp.StatusCode, err)
		}

		return fmt.Errorf("%w: %d %s", ErrHTTPRequest, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// ParseGitTagsOutput parses git ls-remote style output into tag names.
// This is useful for parsing cached or pre-fetched tag data.
func ParseGitTagsOutput(output string) []string {
	tags := make([]string, 0, 10)

	seen := make(map[string]bool)

	re := regexp.MustCompile(`refs/tags/([^\s^{}]+)`)

	for line := range strings.SplitSeq(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		tag := matches[1]
		if seen[tag] {
			continue
		}

		seen[tag] = true
		tags = append(tags, tag)
	}

	return tags
}

// GetToken returns the current auth token (for testing).
func (client *Client) GetToken() string {
	return client.authToken
}
