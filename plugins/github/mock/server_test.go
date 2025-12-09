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
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestMockServerSuite runs the GitHub mock server Ginkgo test suite.
func TestMockServerSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GitHub Mock Server Suite")
}

var _ = Describe("GitHub Mock Server", func() {
	var server *Server

	BeforeEach(func() {
		server = NewServer()
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Describe("NewServer", func() {
		It("creates a server with valid URL", func() {
			Expect(server).NotTo(BeNil())
			Expect(server.URL()).NotTo(BeEmpty())
			Expect(server.HTTPServer).NotTo(BeNil())
		})
	})

	testRepoDataEndpoint := func(setup func(*Server), endpoint string, expectedLen int) {
		setup(server)

		resp, err := http.Get(server.URL() + endpoint)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		var items []json.RawMessage
		err = json.Unmarshal(body, &items)
		Expect(err).NotTo(HaveOccurred())
		Expect(items).To(HaveLen(expectedLen))
	}

	Describe("AddTags", func() {
		It("adds tags for a repository", func() {
			testRepoDataEndpoint(
				func(s *Server) { s.AddTags("golang", "go", []string{"go1.20.0", "go1.21.0"}) },
				"/repos/golang/go/git/refs/tags", 2)
		})
	})

	Describe("AddReleases", func() {
		It("adds releases for a repository", func() {
			testRepoDataEndpoint(
				func(s *Server) { s.AddReleases("kubernetes", "kubernetes", []string{"v1.28.0", "v1.29.0"}) },
				"/repos/kubernetes/kubernetes/releases", 2)
		})
	})

	Describe("HTTP endpoints", func() {
		It("returns 404 for unknown paths", func() {
			resp, err := http.Get(server.URL() + "/unknown/path")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for repos without tags", func() {
			resp, err := http.Get(server.URL() + "/repos/unknown/repo/git/refs/tags")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for repos without releases", func() {
			resp, err := http.Get(server.URL() + "/repos/unknown/repo/releases")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("extractRepoPath", func() {
		It("extracts owner/repo from path", func() {
			Expect(extractRepoPath("/repos/golang/go/git/refs/tags", "/git/refs/tags")).To(Equal("golang/go"))
			Expect(extractRepoPath("/repos/k8s/k8s/releases", "/releases")).To(Equal("k8s/k8s"))
			Expect(extractRepoPath("/invalid", "")).To(Equal("/invalid"))
		})
	})
})
