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
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
)

// TestParseGitTagsOutputGoldie tests git tags output parsing with golden files.
func TestParseGitTagsOutputGoldie(t *testing.T) {
	output := `abc123	refs/tags/go1.20.0
def456	refs/tags/go1.21.0
ghi789	refs/tags/go1.21.0^{}`

	tags := ParseGitTagsOutput(output)

	goldieRecorder := goldie.New(t)
	goldieRecorder.Assert(t, "git_tags_output", []byte(strings.Join(tags, "\n")))
}
