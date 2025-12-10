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

package asdf_plugin_go

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
)

// errGoNoVersionsFound is returned when no Go versions are discovered.
var errGoNoVersionsFound = errors.New("no versions found")

// ListAll returns all available Go versions using the GitHub API.
func (p *Plugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := p.githubClient.GetTags(ctx, goGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("listing go versions: %w", err)
	}

	versions := parseGoTags(tags)

	versions = filterOldVersions(versions)

	// Prefer stable versions in list-all output when possible, but keep
	// prereleases when no stable versions exist.
	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		versions = stable
	}

	sortGoVersions(versions)

	return versions, nil
}

// LatestStable returns the latest stable Go version.
func (p *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := p.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errGoNoVersionsFound
	}

	if query != "" {
		filtered := asdf.FilterVersions(versions, func(v string) bool {
			return strings.HasPrefix(v, query)
		})
		if len(filtered) > 0 {
			versions = filtered
		}
	}

	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) == 0 {
		return versions[len(versions)-1], nil
	}

	return stable[len(stable)-1], nil
}

// parseGoTags extracts version numbers from GitHub tag names.
func parseGoTags(tags []string) []string {
	var versions []string
	for _, tag := range tags {
		if after, ok := strings.CutPrefix(tag, "go"); ok {
			versions = append(versions, after)
		}
	}

	return versions
}

// parseGoVersions extracts version numbers from git ls-remote output.
func parseGoVersions(output string) []string {
	versions := make([]string, 0, 10)

	seen := make(map[string]bool)

	re := regexp.MustCompile(`refs/tags/go([^\s^{}]+)`)

	for line := range strings.SplitSeq(output, "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		version := matches[1]
		if seen[version] {
			continue
		}

		seen[version] = true
		versions = append(versions, version)
	}

	return versions
}

// filterOldVersions removes old/unsupported Go versions.
// Filters out: 1, 1.0, 1.0.x, 1.1, 1.1rc*, 1.1.x, 1.2, 1.2rc*, 1.2.1, 1.8.5rc5.
func filterOldVersions(versions []string) []string {
	excludePatterns := []string{
		`^1$`,
		`^1\.0$`,
		`^1\.0\.[0-9]+$`,
		`^1\.1$`,
		`^1\.1rc[0-9]+$`,
		`^1\.1\.[0-9]+$`,
		`^1\.2$`,
		`^1\.2rc[0-9]+$`,
		`^1\.2\.1$`,
		`^1\.8\.5rc5$`,
	}

	compiled := make([]*regexp.Regexp, 0, len(excludePatterns))
	for _, pattern := range excludePatterns {
		compiled = append(compiled, regexp.MustCompile(pattern))
	}

	return asdf.FilterVersions(versions, func(v string) bool {
		for _, re := range compiled {
			if re.MatchString(v) {
				return false
			}
		}

		return true
	})
}

// sortGoVersions sorts versions in semver order.
func sortGoVersions(versions []string) {
	asdf.SortVersions(versions)
}

// compareGoVersions compares two Go version strings.
func compareGoVersions(a, b string) int {
	return asdf.CompareVersions(a, b)
}
