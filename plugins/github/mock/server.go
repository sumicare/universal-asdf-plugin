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

package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
)

type (
	// Server provides a mock GitHub API server for testing.
	Server struct {
		HTTPServer *httptest.Server
		tags       map[string][]TagResponse
		releases   map[string][]ReleaseResponse
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

// NewServer creates a new mock GitHub API server.
func NewServer() *Server {
	mock := &Server{
		tags:     make(map[string][]TagResponse),
		releases: make(map[string][]ReleaseResponse),
	}

	mock.HTTPServer = httptest.NewServer(
		http.HandlerFunc(func(responseWriter http.ResponseWriter, req *http.Request) {
			path := req.URL.Path

			if strings.Contains(path, "/git/refs/tags") {
				repoPath := extractRepoPath(path, "/git/refs/tags")
				if tags, ok := mock.tags[repoPath]; ok {
					responseWriter.Header().Set("Content-Type", "application/json")

					_ = json.NewEncoder(responseWriter).Encode(tags)

					return
				}
			}

			if strings.Contains(path, "/releases") {
				repoPath := extractRepoPath(path, "/releases")
				if releases, ok := mock.releases[repoPath]; ok {
					responseWriter.Header().Set("Content-Type", "application/json")

					_ = json.NewEncoder(responseWriter).Encode(releases)

					return
				}
			}

			responseWriter.WriteHeader(http.StatusNotFound)
		}),
	)

	return mock
}

// extractRepoPath extracts "owner/repo" from a path like "/repos/owner/repo/...".
func extractRepoPath(path, suffix string) string {
	return strings.TrimSuffix(
		strings.TrimPrefix(path, "/repos/"),
		suffix,
	)
}

// URL returns the base URL of the mock server.
func (s *Server) URL() string {
	return s.HTTPServer.URL
}

// Close shuts down the mock server.
func (s *Server) Close() {
	s.HTTPServer.Close()
}

// AddTags adds tags for a repository.
func (s *Server) AddTags(owner, repo string, tags []string) {
	repoPath := owner + "/" + repo

	s.tags[repoPath] = make([]TagResponse, 0, len(tags))
	for _, tag := range tags {
		s.tags[repoPath] = append(s.tags[repoPath], TagResponse{Ref: "refs/tags/" + tag})
	}
}

// AddReleases adds releases for a repository.
func (s *Server) AddReleases(owner, repo string, releases []string) {
	repoPath := owner + "/" + repo

	s.releases[repoPath] = make([]ReleaseResponse, 0, len(releases))
	for _, release := range releases {
		s.releases[repoPath] = append(s.releases[repoPath], ReleaseResponse{TagName: release})
	}
}
