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

package mock_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	githubmock "github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

func TestMockServer(t *testing.T) {
	t.Parallel()

	testRepoDataEndpoint := func(
		t *testing.T,
		server *githubmock.Server,
		setup func(*githubmock.Server),
		endpoint string,
		expectedLen int,
	) {
		t.Helper()

		setup(server)

		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			server.URL()+endpoint,
			http.NoBody,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = resp.Body.Close()
		})

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var items []json.RawMessage
		require.NoError(t, json.Unmarshal(body, &items))
		require.Len(t, items, expectedLen)
	}

	t.Run("NewServer creates a server with valid URL", func(t *testing.T) {
		t.Parallel()

		server := githubmock.NewServer()
		t.Cleanup(server.Close)

		require.NotNil(t, server)
		require.NotEmpty(t, server.URL())
		require.NotNil(t, server.HTTPServer)
	})

	t.Run("AddTags adds tags for a repository", func(t *testing.T) {
		t.Parallel()

		server := githubmock.NewServer()
		t.Cleanup(server.Close)

		testRepoDataEndpoint(
			t,
			server,
			func(s *githubmock.Server) { s.AddTags("golang", "go", []string{"go1.20.0", "go1.21.0"}) },
			"/repos/golang/go/git/refs/tags",
			2,
		)
	})

	t.Run("AddReleases adds releases for a repository", func(t *testing.T) {
		t.Parallel()

		server := githubmock.NewServer()
		t.Cleanup(server.Close)

		testRepoDataEndpoint(
			t,
			server,
			func(s *githubmock.Server) { s.AddReleases("kubernetes", "kubernetes", []string{"v1.28.0", "v1.29.0"}) },
			"/repos/kubernetes/kubernetes/releases",
			2,
		)
	})

	t.Run("HTTP endpoints", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			expected int
		}{
			{
				name:     "returns 404 for unknown paths",
				path:     "/unknown/path",
				expected: http.StatusNotFound,
			},
			{
				name:     "returns 404 for repos without tags",
				path:     "/repos/unknown/repo/git/refs/tags",
				expected: http.StatusNotFound,
			},
			{
				name:     "returns 404 for repos without releases",
				path:     "/repos/unknown/repo/releases",
				expected: http.StatusNotFound,
			},
		}

		for i := range tests {
			tt := tests[i]
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				server := githubmock.NewServer()
				t.Cleanup(server.Close)

				req, err := http.NewRequestWithContext(
					t.Context(),
					http.MethodGet,
					server.URL()+tt.path,
					http.NoBody,
				)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				t.Cleanup(func() {
					_ = resp.Body.Close()
				})

				require.Equal(t, tt.expected, resp.StatusCode)
			})
		}
	})

	t.Run("extractRepoPath extracts owner/repo from path", func(t *testing.T) {
		t.Parallel()

		require.Equal(
			t,
			"golang/go",
			githubmock.ExtractRepoPathForTests("/repos/golang/go/git/refs/tags", "/git/refs/tags"),
		)
		require.Equal(
			t,
			"k8s/k8s",
			githubmock.ExtractRepoPathForTests("/repos/k8s/k8s/releases", "/releases"),
		)
		require.Equal(t, "/invalid", githubmock.ExtractRepoPathForTests("/invalid", ""))
	})
}
