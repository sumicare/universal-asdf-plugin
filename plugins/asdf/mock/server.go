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

// Package mock provides a generic HTTP server used in tests for simulating
// download endpoints and release metadata.
package mock

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/ulikunitz/xz"
)

type (
	// downloadHandler handles a single download response for a given path.
	downloadHandler func(responseWriter http.ResponseWriter)

	// Server is a generic mock HTTP server for GitHub releases.
	Server struct {
		server    *httptest.Server
		downloads map[string]downloadHandler
		RepoOwner string
		RepoName  string
		tags      []string
		mu        sync.RWMutex
	}
)

// NewServer creates a new mock server.
func NewServer(repoOwner, repoName string) *Server {
	srv := &Server{
		downloads: make(map[string]downloadHandler),
		tags:      make([]string, 0, 10),
		RepoOwner: repoOwner,
		RepoName:  repoName,
	}

	mux := http.NewServeMux()

	tagsPath := fmt.Sprintf("/repos/%s/%s/git/refs/tags", repoOwner, repoName)
	mux.HandleFunc(tagsPath, srv.handleTags)

	releasesPath := fmt.Sprintf("/repos/%s/%s/releases", repoOwner, repoName)
	mux.HandleFunc(releasesPath, srv.handleReleases)

	mux.HandleFunc("/", srv.handleDownload)

	srv.server = httptest.NewServer(mux)

	return srv
}

// URL returns the mock server URL.
func (server *Server) URL() string {
	return server.server.URL
}

// Close shuts down the mock server.
func (server *Server) Close() {
	server.server.Close()
}

// Client returns the HTTP client for the mock server.
func (server *Server) Client() *http.Client {
	return server.server.Client()
}

// RegisterDownload registers a valid download path with default mock binary content.
func (server *Server) RegisterDownload(path string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/octet-stream")

		_, _ = writer.Write([]byte("#!/bin/sh\necho 'mock binary'\n")) //nolint:errcheck // we'll ignore mocked errors
	}
}

// RegisterZipDownload registers a zip download with the given file contents.
func (server *Server) RegisterZipDownload(path string, files map[string]string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/zip")

		zw := zip.NewWriter(writer)
		defer zw.Close()

		for name, content := range files {
			fw, _ := zw.Create(name) //nolint:errcheck // we'll ignore mocked errors

			_, _ = fw.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
		}
	}
}

// RegisterTarGzDownload registers a tar.gz download with the given file contents.
func (server *Server) RegisterTarGzDownload(path string, files map[string]string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/gzip")

		gw := gzip.NewWriter(writer)
		defer gw.Close()

		tw := tar.NewWriter(gw)
		defer tw.Close()

		for name, content := range files {
			hdr := &tar.Header{
				Name: name,
				Mode: int64(CommonDirectoryPermission),
				Size: int64(len(content)),
			}

			_ = tw.WriteHeader(hdr)          //nolint:errcheck // we'll ignore mocked errors
			_, _ = tw.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
		}
	}
}

// RegisterTarXzDownload registers a tar.xz download with the given file contents.
func (server *Server) RegisterTarXzDownload(path string, files map[string]string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/x-xz")

		xw, err := xz.NewWriter(writer)
		if err != nil {
			http.Error(writer, "failed to create xz writer", http.StatusInternalServerError)
			return
		}
		defer xw.Close()

		tw := tar.NewWriter(xw)
		defer tw.Close()

		for name, content := range files {
			hdr := &tar.Header{
				Name: name,
				Mode: int64(CommonDirectoryPermission),
				Size: int64(len(content)),
			}

			_ = tw.WriteHeader(hdr)          //nolint:errcheck // we'll ignore mocked errors
			_, _ = tw.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
		}
	}
}

// RegisterGzDownload registers a gz download with the given content.
func (server *Server) RegisterGzDownload(path, content string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/gzip")

		gw := gzip.NewWriter(writer)
		defer gw.Close()

		_, _ = gw.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
	}
}

// RegisterFile registers a file download with the given content.
func (server *Server) RegisterFile(path string, content []byte) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/octet-stream")

		_, _ = writer.Write(content) //nolint:errcheck // we'll ignore mocked errors
	}
}

// RegisterText registers a text response.
func (server *Server) RegisterText(path, content string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "text/plain")

		_, _ = writer.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
	}
}

// RegisterHTML registers an HTML response.
func (server *Server) RegisterHTML(path, content string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "text/html")

		_, _ = writer.Write([]byte(content)) //nolint:errcheck // we'll ignore mocked errors
	}
}

// RegisterJSON registers a JSON response.
func (server *Server) RegisterJSON(path string, data any) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.downloads[path] = func(writer http.ResponseWriter) {
		writer.Header().Set("Content-Type", "application/json")

		_ = json.NewEncoder(writer).Encode(data) //nolint:errcheck,errchkjson // we'll ignore mocked errors
	}
}

// RegisterTag registers a git tag.
func (server *Server) RegisterTag(tag string) {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.tags = append(server.tags, tag)
}

// ClearTags removes all registered git tags from the mock server.
// This is primarily used in tests that need to simulate an empty tag set
// or to replace existing tags with a new set.
func (server *Server) ClearTags() {
	server.mu.Lock()
	defer server.mu.Unlock()

	server.tags = server.tags[:0]
}

// handleTags returns the registered tags as a JSON response.
func (server *Server) handleTags(writer http.ResponseWriter, _ *http.Request) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	type tagResponse struct {
		Ref string `json:"ref"`
	}

	tags := make([]tagResponse, 0, len(server.tags))
	for i := range server.tags {
		tags = append(tags, tagResponse{Ref: "refs/tags/" + server.tags[i]})
	}

	writer.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(writer).Encode(tags) //nolint:errcheck,errchkjson // we'll ignore mocked errors
}

// handleReleases returns the registered releases as a JSON response.
func (server *Server) handleReleases(writer http.ResponseWriter, _ *http.Request) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	type releaseResponse struct {
		TagName string `json:"tag_name"`
	}

	releases := make([]releaseResponse, 0, len(server.tags))
	for i := range server.tags {
		releases = append(releases, releaseResponse{TagName: server.tags[i]})
	}

	writer.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(writer).Encode(releases) //nolint:errcheck,errchkjson // we'll ignore mocked errors
}

// handleDownload dispatches a download request to a registered downloadHandler
// or returns 404 if no handler is registered for the requested path.
func (server *Server) handleDownload(responseWriter http.ResponseWriter, req *http.Request) {
	server.mu.RLock()

	handler, ok := server.downloads[req.URL.Path]
	server.mu.RUnlock()

	if !ok {
		fmt.Printf("Mock 404: Path %s not found\n", req.URL.Path)
		http.NotFound(responseWriter, req)

		return
	}

	handler(responseWriter)
}
