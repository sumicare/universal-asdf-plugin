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
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	It("serves tags and releases based on registered tags", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		server.RegisterTag("v1.0.0")
		server.RegisterTag("v1.1.0")

		baseURL, err := url.Parse(server.URL())
		Expect(err).NotTo(HaveOccurred())

		tagsURL := baseURL.ResolveReference(&url.URL{Path: "/repos/owner/repo/git/refs/tags"})
		resp, err := http.Get(tagsURL.String())
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var tagsResp []struct {
			Ref string `json:"ref"`
		}
		decoder := json.NewDecoder(resp.Body)
		Expect(decoder.Decode(&tagsResp)).To(Succeed())
		Expect(tagsResp).To(HaveLen(2))
		Expect(tagsResp[0].Ref).To(Equal("refs/tags/v1.0.0"))
		Expect(tagsResp[1].Ref).To(Equal("refs/tags/v1.1.0"))

		resp, err = http.Get(baseURL.ResolveReference(&url.URL{Path: "/repos/owner/repo/releases"}).String())
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var relResp []struct {
			TagName string `json:"tag_name"`
		}
		decoder = json.NewDecoder(resp.Body)
		Expect(decoder.Decode(&relResp)).To(Succeed())
		Expect(relResp).To(HaveLen(2))
		Expect(relResp[0].TagName).To(Equal("v1.0.0"))
		Expect(relResp[1].TagName).To(Equal("v1.1.0"))
	})

	It("clears tags with ClearTags", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		server.RegisterTag("v1.0.0")
		server.ClearTags()

		resp, err := http.Get(server.URL() + "/repos/owner/repo/git/refs/tags")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var tagsResp []struct {
			Ref string `json:"ref"`
		}
		Expect(json.NewDecoder(resp.Body).Decode(&tagsResp)).To(Succeed())
		Expect(tagsResp).To(BeEmpty())
	})

	It("serves registered downloads and returns 404 for unknown paths", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		const path = "/owner/repo/releases/download/v1.0.0/tool"
		server.RegisterText(path, "ok")

		resp, err := http.Get(server.URL() + path)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("ok"))

		resp, err = http.Get(server.URL() + "/unknown")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("exposes an HTTP client via Client", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		client := server.Client()
		Expect(client).NotTo(BeNil())

		resp, err := client.Get(server.URL() + "/repos/owner/repo/git/refs/tags")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("serves basic binary downloads registered with RegisterDownload", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		const path = "/owner/repo/releases/download/v1.0.0/binary"
		server.RegisterDownload(path)

		resp, err := http.Get(server.URL() + path)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(Equal("application/octet-stream"))
	})

	It("serves zip, tar.gz, tar.xz, and gz downloads", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		server.RegisterZipDownload("/zip", map[string]string{"bin": "zip"})
		server.RegisterTarGzDownload("/targz", map[string]string{"bin": "targz"})
		server.RegisterTarXzDownload("/tarxz", map[string]string{"bin": "tarxz"})
		server.RegisterGzDownload("/gz", "gz")

		paths := map[string]string{
			"/zip":   "application/zip",
			"/targz": "application/gzip",
			"/tarxz": "application/x-xz",
			"/gz":    "application/gzip",
		}

		for p, contentType := range paths {
			resp, err := http.Get(server.URL() + p)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(Equal(contentType))
		}
	})

	It("serves file, HTML, and JSON responses", func() {
		server := NewServer("owner", "repo")
		defer server.Close()

		server.RegisterFile("/file", []byte("data"))
		server.RegisterHTML("/html", "<h1>ok</h1>")
		server.RegisterJSON("/json", map[string]string{"k": "v"})

		resp, err := http.Get(server.URL() + "/file")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(Equal("application/octet-stream"))
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("data"))

		resp, err = http.Get(server.URL() + "/html")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(Equal("text/html"))
		body, err = io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("<h1>ok</h1>"))

		resp, err = http.Get(server.URL() + "/json")
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))

		var decoded map[string]string
		Expect(json.NewDecoder(resp.Body).Decode(&decoded)).To(Succeed())
		Expect(decoded).To(HaveKeyWithValue("k", "v"))
	})
})
