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
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sumicare/universal-asdf-plugin/plugins/github/mock"
)

// isOnline reports whether integration tests should run against the real
// GitHub API. When ONLINE is not set, tests are executed purely against
// the mock server implementation.
func isOnline() bool {
	v := os.Getenv("ONLINE")
	return v == "1" || strings.EqualFold(v, "true")
}

var _ = Describe("GitHub Client", func() {
	Describe("NewClient", func() {
		It("creates a client with defaults", func() {
			client := NewClient()
			Expect(client).NotTo(BeNil())
		})

		It("uses GITHUB_TOKEN from environment", func() {
			original := os.Getenv("GITHUB_TOKEN")
			defer os.Setenv("GITHUB_TOKEN", original)

			os.Setenv("GITHUB_TOKEN", "test-token")
			client := NewClient()
			Expect(client.GetToken()).To(Equal("test-token"))
		})

		It("uses GITHUB_API_TOKEN as fallback", func() {
			originalToken := os.Getenv("GITHUB_TOKEN")
			originalAPIToken := os.Getenv("GITHUB_API_TOKEN")
			defer func() {
				os.Setenv("GITHUB_TOKEN", originalToken)
				os.Setenv("GITHUB_API_TOKEN", originalAPIToken)
			}()

			os.Unsetenv("GITHUB_TOKEN")
			os.Setenv("GITHUB_API_TOKEN", "api-token")
			client := NewClient()
			Expect(client.GetToken()).To(Equal("api-token"))
		})
	})

	Describe("NewClientWithToken", func() {
		It("creates client with specified token", func() {
			client := NewClientWithToken("my-token")
			Expect(client.GetToken()).To(Equal("my-token"))
		})
	})

	Describe("NewClientWithHTTP", func() {
		It("creates client with custom HTTP client", func() {
			httpClient := &http.Client{Timeout: 5 * time.Second}
			client := NewClientWithHTTP(httpClient, "https://custom.api.com")
			Expect(client).NotTo(BeNil())
		})
	})

	Describe("SetToken", func() {
		It("sets the authentication token", func() {
			client := NewClient()
			client.SetToken("new-token")
			Expect(client.GetToken()).To(Equal("new-token"))
		})
	})

	DescribeTable("GetOwnerRepo",
		func(url, expectedOwner, expectedRepo string, expectError bool) {
			owner, repo, err := GetOwnerRepo(url)
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(owner).To(Equal(expectedOwner))
				Expect(repo).To(Equal(expectedRepo))
			}
		},
		Entry("HTTPS URL", "https://github.com/golang/go", "golang", "go", false),
		Entry("HTTPS URL with .git", "https://github.com/golang/go.git", "golang", "go", false),
		Entry("SSH URL", "git@github.com:golang/go.git", "golang", "go", false),
		Entry("invalid URL", "invalid", "", "", true),
		Entry("too many parts", "https://github.com/a/b/c", "", "", true),
	)

	Describe("GetTags with mock server", func() {
		var server *mock.Server

		BeforeEach(func() {
			if isOnline() {
				Skip("skipping mock test in integration mode")
			}
			server = mock.NewServer()
		})

		AfterEach(func() {
			if server != nil {
				server.Close()
			}
		})

		It("fetches tags from mock server", func() {
			server.AddTags("golang", "go", []string{"go1.20.0", "go1.21.0", "go1.22.0"})

			client := NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
			tags, err := client.GetTags(context.Background(), "https://github.com/golang/go")

			Expect(err).NotTo(HaveOccurred())
			Expect(tags).To(HaveLen(3))
			Expect(tags).To(ContainElements("go1.20.0", "go1.21.0", "go1.22.0"))
		})

		It("returns error for invalid URL", func() {
			client := NewClient()
			_, err := client.GetTags(context.Background(), "invalid")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetReleases with mock server", func() {
		var server *mock.Server

		BeforeEach(func() {
			if isOnline() {
				Skip("skipping mock test in integration mode")
			}
			server = mock.NewServer()
		})

		AfterEach(func() {
			if server != nil {
				server.Close()
			}
		})

		It("fetches releases from mock server", func() {
			server.AddReleases("kubernetes", "kubernetes", []string{"v1.28.0", "v1.29.0"})

			client := NewClientWithHTTP(server.HTTPServer.Client(), server.URL())
			releases, err := client.GetReleases(context.Background(), "https://github.com/kubernetes/kubernetes")

			Expect(err).NotTo(HaveOccurred())
			Expect(releases).To(HaveLen(2))
		})
	})

	Describe("ParseGitTagsOutput", func() {
		It("parses git ls-remote output", func() {
			output := "abc123\trefs/tags/go1.20.0\ndef456\trefs/tags/go1.21.0\nghi789\trefs/tags/go1.21.0^{}"
			tags := ParseGitTagsOutput(output)

			Expect(tags).To(HaveLen(2))
			Expect(tags).To(ContainElements("go1.20.0", "go1.21.0"))
		})
	})

	When("running online tests", func() {
		BeforeEach(func() {
			if !isOnline() {
				Skip("skipping online test (set ONLINE=1 to run)")
			}
		})

		It("fetches real tags from GitHub", func() {
			client := NewClient()
			tags, err := client.GetTags(context.Background(), "https://github.com/golang/go")

			Expect(err).NotTo(HaveOccurred())
			Expect(tags).NotTo(BeEmpty())

			found := false
			for _, tag := range tags {
				if strings.Contains(tag, "go1.21") {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected to find go1.21.x tag")
		})

		It("fetches real releases from GitHub", func() {
			client := NewClient()
			releases, err := client.GetReleases(context.Background(), "https://github.com/kubernetes/kubernetes")

			Expect(err).NotTo(HaveOccurred())
			Expect(releases).NotTo(BeEmpty())
		})
	})
})
