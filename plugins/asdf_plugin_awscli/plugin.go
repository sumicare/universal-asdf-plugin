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

package asdf_plugin_awscli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/sumicare/universal-asdf-plugin/plugins/asdf"
	"github.com/sumicare/universal-asdf-plugin/plugins/github"
)

var (
	// errAWSNoVersionsFound is returned when no AWS CLI versions are discovered.
	errAWSNoVersionsFound = errors.New("no versions found")
	// errAWSNoVersionsMatching is returned when no versions match a LatestStable query.
	errAWSNoVersionsMatching = errors.New("no versions matching query")
	// errAWSUnsupportedPlatform is returned when the current OS/arch pair has no installer.
	errAWSUnsupportedPlatform = errors.New("unsupported platform")
	// errAWSDownloadFailed indicates a non-success HTTP response when downloading AWS CLI.
	errAWSDownloadFailed = errors.New("download failed")
	// errAWSUnsupportedInstallOS is returned when Install is invoked on an unsupported OS.
	errAWSUnsupportedInstallOS = errors.New("unsupported install platform")

	// awscliDownloadBaseURL is the base URL used to construct AWS CLI download URLs.
	// It is a variable (not a constant) to allow tests to override it with a
	// local httptest.Server, ensuring Download can run fully offline.
	awscliDownloadBaseURL = "https://awscli.amazonaws.com" //nolint:gochecknoglobals // configurable in tests

	// awscliHTTPClient is the HTTP client used for downloads. It is a variable so
	// tests can replace it with a mock client to avoid real network usage while
	// still exercising the non-cached Download path.
	awscliHTTPClient = http.DefaultClient //nolint:gochecknoglobals // configurable in tests

	// execCommandContextFn wraps exec.CommandContext to allow tests to stub
	// external command execution (e.g., pkgutil, cp, installer scripts) without
	// invoking real system binaries.
	execCommandContextFn = exec.CommandContext //nolint:gochecknoglobals // configurable in tests

	// extractZipFn wraps asdf.ExtractZip so tests can force an extraction error
	// without relying on actual corrupt archives.
	extractZipFn = asdf.ExtractZip //nolint:gochecknoglobals // configurable in tests
)

// awscliGitRepoURL is the upstream Git repository for awscli.
const awscliGitRepoURL = "https://github.com/aws/aws-cli"

type (
	// awscliRuntime captures the minimal OS/arch runtime information needed
	// to determine the correct AWS CLI distribution for the current platform.
	awscliRuntime struct {
		goos string
		arch string
	}

	// Plugin implements the asdf.Plugin interface for AWS CLI.
	Plugin struct {
		githubClient *github.Client
		runtime      awscliRuntime
	}
)

// defaultAWSCLIRuntime returns an awscliRuntime based on the current Go runtime.
func defaultAWSCLIRuntime() awscliRuntime {
	return awscliRuntime{
		goos: runtime.GOOS,
		arch: runtime.GOARCH,
	}
}

// New creates a new AWS CLI plugin instance.
func New() *Plugin {
	return &Plugin{
		githubClient: github.NewClient(),
		runtime:      defaultAWSCLIRuntime(),
	}
}

// NewWithClient creates a new AWS CLI plugin with custom client (for testing).
func NewWithClient(client *github.Client) *Plugin {
	return &Plugin{
		githubClient: client,
		runtime:      defaultAWSCLIRuntime(),
	}
}

// newPluginWithRuntime constructs a Plugin with the provided runtime, used in tests.
func newPluginWithRuntime(client *github.Client, rt awscliRuntime) *Plugin {
	return &Plugin{
		githubClient: client,
		runtime:      rt,
	}
}

// Name returns the plugin name.
func (*Plugin) Name() string {
	return "awscli"
}

// ListBinPaths returns the binary paths for AWS CLI installations.
func (*Plugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for AWS CLI execution.
func (*Plugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for AWS CLI.
func (*Plugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy AWS CLI version file.
func (*Plugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes an AWS CLI installation.
func (*Plugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the AWS CLI plugin.
func (*Plugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `AWS CLI - The AWS Command Line Interface.
This plugin downloads pre-built AWS CLI v2 binaries.`,
		Deps: `Linux: glibc, groff, less
macOS: Rosetta 2 (for Apple Silicon)`,
		Config: `Environment variables:
  AWS_CONFIG_FILE - Override AWS config file location
  AWS_SHARED_CREDENTIALS_FILE - Override credentials file location`,
		Links: `Homepage: https://aws.amazon.com/cli/
Documentation: https://docs.aws.amazon.com/cli/
Source: https://github.com/aws/aws-cli`,
	}
}

// ListAll lists all available AWS CLI versions.
func (plugin *Plugin) ListAll(ctx context.Context) ([]string, error) {
	tags, err := plugin.githubClient.GetTags(ctx, awscliGitRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	versionRegex := regexp.MustCompile(`^2\.\d+\.\d+$`)

	versions := make([]string, 0, len(tags))
	for _, tag := range tags {
		if versionRegex.MatchString(tag) {
			versions = append(versions, tag)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return asdf.CompareVersions(versions[i], versions[j]) < 0
	})

	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) > 0 {
		return stable, nil
	}

	return versions, nil
}

// LatestStable returns the latest stable AWS CLI version.
func (plugin *Plugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", errAWSNoVersionsFound
	}

	if query != "" {
		var filtered []string
		for _, v := range versions {
			if strings.HasPrefix(v, query) {
				filtered = append(filtered, v)
			}
		}

		versions = filtered
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("%w: %s", errAWSNoVersionsMatching, query)
	}

	stable := asdf.FilterVersions(versions, func(v string) bool {
		return !asdf.IsPrereleaseVersion(v)
	})

	if len(stable) == 0 {
		return versions[len(versions)-1], nil
	}

	return stable[len(stable)-1], nil
}

// getDownloadURL returns the download URL for the specified version.
func (plugin *Plugin) getDownloadURL(version string) (string, error) {
	goos := plugin.runtime.goos
	arch := plugin.runtime.arch

	switch goos {
	case "linux":
		switch arch {
		case "amd64":
			return fmt.Sprintf("%s/awscli-exe-linux-x86_64-%s.zip", awscliDownloadBaseURL, version), nil
		case "arm64":
			return fmt.Sprintf("%s/awscli-exe-linux-aarch64-%s.zip", awscliDownloadBaseURL, version), nil
		}

	case "darwin":
		return fmt.Sprintf("%s/AWSCLIV2-%s.pkg", awscliDownloadBaseURL, version), nil
	}

	return "", fmt.Errorf("%w: %s/%s", errAWSUnsupportedPlatform, goos, arch)
}

// Download downloads the specified AWS CLI version.
func (plugin *Plugin) Download(ctx context.Context, version, downloadPath string) error {
	url, err := plugin.getDownloadURL(version)
	if err != nil {
		return err
	}

	filename := filepath.Base(url)

	filePath := filepath.Join(downloadPath, filename)
	if info, err := os.Stat(filePath); err == nil && info.Size() > 1024 {
		asdf.Msgf("Using cached download for awscli %s", version)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := awscliHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading awscli: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status: %d", errAWSDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, asdf.CommonFilePermission)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if strings.HasSuffix(filename, ".zip") {
		if err := plugin.extractZip(filePath, downloadPath); err != nil {
			return fmt.Errorf("extracting zip: %w", err)
		}
	}

	return nil
}

// extractZip extracts a zip file to the destination directory.
func (*Plugin) extractZip(zipPath, destPath string) error {
	return extractZipFn(zipPath, destPath)
}

// Install installs AWS CLI from the downloaded files.
func (plugin *Plugin) Install(ctx context.Context, version, downloadPath, installPath string) error {
	goos := plugin.runtime.goos

	switch goos {
	case "linux":
		return plugin.installLinux(ctx, downloadPath, installPath)
	case "darwin":
		return plugin.installDarwin(ctx, version, downloadPath, installPath)
	default:
		return fmt.Errorf("%w: %s", errAWSUnsupportedInstallOS, goos)
	}
}

// installLinux installs AWS CLI on Linux.
func (*Plugin) installLinux(ctx context.Context, downloadPath, installPath string) error {
	installerPath := filepath.Join(downloadPath, "aws", "install")

	if err := os.Chmod(installerPath, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("making installer executable: %w", err)
	}

	binDir := filepath.Join(installPath, "bin")
	libDir := filepath.Join(installPath, "aws-cli")

	cmd := execCommandContextFn(ctx, installerPath, "-i", libDir, "-b", binDir)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running installer: %w", err)
	}

	return nil
}

// installDarwin installs AWS CLI on macOS.
func (*Plugin) installDarwin(ctx context.Context, version, downloadPath, installPath string) error {
	pkgPath := filepath.Join(downloadPath, fmt.Sprintf("AWSCLIV2-%s.pkg", version))

	binDir := filepath.Join(installPath, "bin")
	if err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission); err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	extractDir := filepath.Join(downloadPath, "extracted")

	cmd := execCommandContextFn(ctx, "pkgutil", "--expand-full", pkgPath, extractDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("extracting pkg: %w", err)
	}

	awsCliSrc := filepath.Join(extractDir, "aws-cli.pkg", "Payload", "aws-cli")
	awsCliDst := filepath.Join(installPath, "aws-cli")

	cmd = execCommandContextFn(ctx, "cp", "-r", awsCliSrc, awsCliDst)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copying aws-cli: %w", err)
	}

	awsBin := filepath.Join(awsCliDst, "aws")
	awsCompleter := filepath.Join(awsCliDst, "aws_completer")

	if err := os.Symlink(awsBin, filepath.Join(binDir, "aws")); err != nil {
		return fmt.Errorf("creating aws symlink: %w", err)
	}

	if err := os.Symlink(awsCompleter, filepath.Join(binDir, "aws_completer")); err != nil {
		return fmt.Errorf("creating aws_completer symlink: %w", err)
	}

	return nil
}
