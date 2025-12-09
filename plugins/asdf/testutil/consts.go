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

package testutil

import "os"

const (
	// CommonFilePermission is the default file permission used when creating files.
	CommonFilePermission os.FileMode = 0o600
	// CommonDirectoryPermission is the default permission used when creating directories.
	CommonDirectoryPermission os.FileMode = 0o755
	// CommonExecutablePermission is the default permission used when creating directories.
	CommonExecutablePermission os.FileMode = 0o755
	// ExecutablePermissionMask is the mask used to set executable permissions.
	ExecutablePermissionMask os.FileMode = 0o111
)
