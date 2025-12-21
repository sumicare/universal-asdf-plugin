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

package github_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	github "github.com/sumicare/universal-asdf-plugin/plugins/github"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

type staticHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (c *staticHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

type errReadCloser struct{}

var errReadFailed = errors.New("read failed")

func (errReadCloser) Read(
	[]byte,
) (int, error) {
	return 0, errReadFailed
}
func (errReadCloser) Close() error { return nil }

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "rate limit exceeded") ||
		strings.Contains(msg, "stream error") ||
		strings.Contains(msg, "CANCEL") ||
		strings.Contains(msg, "HTTP request failed: 429") ||
		strings.Contains(msg, "HTTP request failed: 5")
}

func TestGitHubClient(t *testing.T) {
	t.Parallel()

	client := github.NewClientWithToken("my-token")
	require.Equal(t, "my-token", client.GetToken())
}

func TestNewClientFromEnv(t *testing.T) {
	// Cannot be parallel as it modifies environment variables
	t.Run("uses GITHUB_TOKEN", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "token1")

		client := github.NewClient()
		require.Equal(t, "token1", client.GetToken())
	})

	t.Run("uses GITHUB_API_TOKEN fallback", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GITHUB_API_TOKEN", "token2")

		client := github.NewClient()
		require.Equal(t, "token2", client.GetToken())
	})

	t.Run("prefers GITHUB_TOKEN over GITHUB_API_TOKEN", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "token1")
		t.Setenv("GITHUB_API_TOKEN", "token2")

		client := github.NewClient()
		require.Equal(t, "token1", client.GetToken())
	})
}

func TestGitHubClient2(t *testing.T) {
	t.Parallel()

	t.Run("NewClientWithHTTP creates client with custom HTTP client", func(t *testing.T) {
		t.Parallel()

		httpClient := &http.Client{Timeout: 5 * time.Second}
		client := github.NewClientWithHTTP(httpClient, "https://custom.api.com")
		require.NotNil(t, client)
	})

	t.Run("SetToken sets the authentication token", func(t *testing.T) {
		t.Parallel()

		client := github.NewClient()
		client.SetToken("new-token")
		require.Equal(t, "new-token", client.GetToken())
	})

	t.Run("GetOwnerRepo", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name          string
			url           string
			expectedOwner string
			expectedRepo  string
			expectError   bool
		}{
			{
				name:          "HTTPS URL",
				url:           "https://github.com/golang/go",
				expectedOwner: "golang",
				expectedRepo:  "go",
			},
			{
				name:          "HTTPS URL with .git",
				url:           "https://github.com/golang/go.git",
				expectedOwner: "golang",
				expectedRepo:  "go",
			},
			{
				name:          "SSH URL",
				url:           "git@github.com:golang/go.git",
				expectedOwner: "golang",
				expectedRepo:  "go",
			},
			{name: "invalid URL", url: "invalid", expectError: true},
			{name: "too many parts", url: "https://github.com/a/b/c", expectError: true},
		}

		for i := range tests {
			tc := tests[i]
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				owner, repo, err := github.GetOwnerRepo(tc.url)
				if tc.expectError {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)
				require.Equal(t, tc.expectedOwner, owner)
				require.Equal(t, tc.expectedRepo, repo)
			})
		}
	})

	t.Run("GetTags with mock server", func(t *testing.T) {
		t.Parallel()

		t.Run("fetches tags from mock server", func(t *testing.T) {
			t.Parallel()

			server := githubmock.NewServer()
			t.Cleanup(server.Close)

			server.AddTags("golang", "go", []string{"go1.20.0", "go1.21.0", "go1.22.0"})

			client := github.NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
			tags, err := client.GetTags(t.Context(), "https://github.com/golang/go")
			require.NoError(t, err)
			require.ElementsMatch(t, []string{"go1.20.0", "go1.21.0", "go1.22.0"}, tags)
		})

		t.Run("returns error for invalid URL", func(t *testing.T) {
			t.Parallel()

			client := github.NewClient()
			_, err := client.GetTags(t.Context(), "invalid")
			require.Error(t, err)
		})

		t.Run("propagates fetch errors", func(t *testing.T) {
			t.Parallel()

			client := github.NewClientForTests(
				&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader("boom")),
					}, nil
				}},
				"https://api.github.com",
				"",
			)

			_, err := client.GetTags(t.Context(), "https://github.com/golang/go")
			require.Error(t, err)
			require.Contains(t, err.Error(), "fetching tags")
		})
	})

	t.Run("GetReleases with mock server", func(t *testing.T) {
		t.Parallel()

		server := githubmock.NewServer()
		t.Cleanup(server.Close)

		server.AddReleases("kubernetes", "kubernetes", []string{"v1.28.0", "v1.29.0"})

		client := github.NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
		releases, err := client.GetReleases(t.Context(), "https://github.com/kubernetes/kubernetes")
		require.NoError(t, err)
		require.Len(t, releases, 2)
	})

	t.Run("GetReleases returns error for invalid URL", func(t *testing.T) {
		t.Parallel()

		client := github.NewClient()
		_, err := client.GetReleases(t.Context(), "invalid")
		require.Error(t, err)
	})

	t.Run("GetReleases propagates fetch errors", func(t *testing.T) {
		t.Parallel()

		client := github.NewClientForTests(
			&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("do failed") //nolint:err113 // test error
			}},
			"https://api.github.com",
			"",
		)

		_, err := client.GetReleases(t.Context(), "https://github.com/kubernetes/kubernetes")
		require.Error(t, err)
		require.Contains(t, err.Error(), "fetching releases")
	})

	t.Run("fetchJSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns error when request cannot be created", func(t *testing.T) {
			t.Parallel()

			client := github.NewClientForTests(
				&http.Client{Timeout: time.Second},
				"https://api.github.com",
				"",
			)

			var out any

			err := client.FetchJSONForTests(t.Context(), "://bad-url", &out)
			require.Error(t, err)
			require.Contains(t, err.Error(), "creating request")
		})

		t.Run("returns error when http client fails", func(t *testing.T) {
			t.Parallel()

			client := github.NewClientForTests(
				&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
					return nil, errors.New("do failed") //nolint:err113 // test error
				}},
				"https://api.github.com",
				"",
			)

			var out any

			err := client.FetchJSONForTests(t.Context(), "https://example.invalid", &out)
			require.Error(t, err)
			require.Contains(t, err.Error(), "http request")
		})

		t.Run("returns ErrHTTPRequest when status is not OK", func(t *testing.T) {
			t.Parallel()

			client := github.NewClientForTests(
				&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader("boom")),
					}, nil
				}},
				"https://api.github.com",
				"",
			)

			var out any

			err := client.FetchJSONForTests(t.Context(), "https://example.invalid", &out)
			require.Error(t, err)
			require.ErrorIs(t, err, github.ErrHTTPRequest)
			require.Contains(t, err.Error(), "boom")
		})

		t.Run(
			"returns ErrHTTPRequest when status is not OK and reading body fails",
			func(t *testing.T) {
				t.Parallel()

				client := github.NewClientForTests(
					&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusInternalServerError,
							Body:       errReadCloser{},
						}, nil
					}},
					"https://api.github.com",
					"",
				)

				var out any

				err := client.FetchJSONForTests(t.Context(), "https://example.invalid", &out)
				require.Error(t, err)
				require.ErrorIs(t, err, github.ErrHTTPRequest)
				require.Contains(t, err.Error(), "failed to read body")
			},
		)

		t.Run("returns error when response is invalid JSON", func(t *testing.T) {
			t.Parallel()

			client := github.NewClientForTests(
				&staticHTTPClient{do: func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("{")),
					}, nil
				}},
				"https://api.github.com",
				"",
			)

			var out any

			err := client.FetchJSONForTests(t.Context(), "https://example.invalid", &out)
			require.Error(t, err)
			require.Contains(t, err.Error(), "decode response")
		})

		t.Run("sets Authorization header when auth token is present", func(t *testing.T) {
			t.Parallel()

			seen := false
			client := github.NewClientForTests(
				&staticHTTPClient{do: func(req *http.Request) (*http.Response, error) {
					seen = true

					require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("[]")),
					}, nil
				}},
				"https://api.github.com",
				"test-token",
			)

			var out []github.TagResponse

			err := client.FetchJSONForTests(t.Context(), "https://example.invalid", &out)
			require.NoError(t, err)
			require.True(t, seen)
		})
	})

	t.Run("ParseGitTagsOutput parses git ls-remote output", func(t *testing.T) {
		t.Parallel()

		output := "abc123\trefs/tags/go1.20.0\ndef456\trefs/tags/go1.21.0\nghi789\trefs/tags/go1.21.0^{}"
		tags := github.ParseGitTagsOutput(output)
		require.Len(t, tags, 2)
		require.ElementsMatch(t, []string{"go1.20.0", "go1.21.0"}, tags)
	})

	t.Run("ParseGitTagsOutput ignores non-tag lines and de-dupes", func(t *testing.T) {
		t.Parallel()

		output := "abc123\trefs/tags/go1.20.0\nnot a tag\nzzz\trefs/tags/go1.20.0\n"
		tags := github.ParseGitTagsOutput(output)
		require.Len(t, tags, 1)
		require.Equal(t, []string{"go1.20.0"}, tags)
	})

	t.Run("online tests", func(t *testing.T) {
		t.Parallel()

		t.Run("fetches real tags from GitHub", func(t *testing.T) {
			t.Parallel()

			client := github.NewClient()

			tags, err := client.GetTags(t.Context(), "https://github.com/golang/go")
			if err != nil && isTransientError(err) {
				t.Skipf("Skipping online test due to transient error: %v", err)
			}

			require.NoError(t, err)
			require.NotEmpty(t, tags)

			found := false

			for _, tag := range tags {
				if strings.Contains(tag, "go1.21") {
					found = true

					break
				}
			}

			require.True(t, found, "expected to find go1.21.x tag")
		})

		t.Run("fetches real releases from GitHub", func(t *testing.T) {
			t.Parallel()

			client := github.NewClient()

			releases, err := client.GetReleases(
				t.Context(),
				"https://github.com/kubernetes/kubernetes",
			)
			if err != nil && isTransientError(err) {
				t.Skipf("Skipping online test due to transient error: %v", err)
			}

			require.NoError(t, err)
			require.NotEmpty(t, releases)
		})
	})
}

// TestParseGitTagsOutputGoldie tests git tags output parsing with golden files.
func TestParseGitTagsOutputGoldie(t *testing.T) {
	t.Parallel()

	output := `abc123	refs/tags/go1.20.0
def456	refs/tags/go1.21.0
ghi789	refs/tags/go1.21.0^{}`

	tags := github.ParseGitTagsOutput(output)

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "git_tags_output", []byte(strings.Join(tags, "\n")))
}
