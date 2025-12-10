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

package asdf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup     func(*testing.T) (src, dst string)
		validate  func(*testing.T, string)
		name      string
		errSubstr string
		wantErr   bool
	}{
		{
			name: "copies directory recursively",
			setup: func(t *testing.T) (string, string) {
				t.Helper()

				src := t.TempDir()
				dst := t.TempDir()

				// Create source structure
				// src/
				//   file1.txt
				//   subdir/
				//     file2.txt
				//   symlink -> file1.txt

				require.NoError(t, os.WriteFile(filepath.Join(src, "file1.txt"), []byte("content1"), 0o600))
				require.NoError(t, os.Mkdir(filepath.Join(src, "subdir"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(src, "subdir", "file2.txt"), []byte("content2"), 0o600))
				require.NoError(t, os.Symlink("file1.txt", filepath.Join(src, "symlink")))

				return src, filepath.Join(dst, "copied")
			},
			validate: func(t *testing.T, dstPath string) {
				t.Helper()

				content1, err := os.ReadFile(filepath.Join(dstPath, "file1.txt"))
				require.NoError(t, err)
				require.Equal(t, "content1", string(content1))

				content2, err := os.ReadFile(filepath.Join(dstPath, "subdir", "file2.txt"))
				require.NoError(t, err)
				require.Equal(t, "content2", string(content2))

				linkTarget, err := os.Readlink(filepath.Join(dstPath, "symlink"))
				require.NoError(t, err)
				require.Equal(t, "file1.txt", linkTarget)
			},
		},
		{
			name: "returns error for non-existent source",
			setup: func(t *testing.T) (string, string) {
				t.Helper()

				return "/non-existent-path", t.TempDir()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:gocritic // let's waste some memory
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src, dst := tt.setup(t)

			err := CopyDir(src, dst)
			if tt.wantErr {
				require.Error(t, err)

				if tt.errSubstr != "" {
					require.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)

				if tt.validate != nil {
					tt.validate(t, dst)
				}
			}
		})
	}
}
