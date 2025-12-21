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

package plugins

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
)

const (
	// awscliGitRepoURL is the GitHub repository URL for AWS CLI.
	awscliGitRepoURL = "https://github.com/aws/aws-cli"
	// awscliDownloadBaseURL is the base URL for downloading AWS CLI packages.
	awscliDownloadBaseURL = "https://awscli.amazonaws.com"
)

type (
	// AwscliPlugin implements the asdf.Plugin interface for AWS CLI.
	AwscliPlugin struct {
		githubClient *github.Client
	}
)

// NewAwscliPlugin creates a new AWS CLI plugin instance.
func NewAwscliPlugin() asdf.Plugin {
	return &AwscliPlugin{
		githubClient: github.NewClient(),
	}
}

// Name returns the plugin name.
func (*AwscliPlugin) Name() string {
	return "awscli"
}

// ListBinPaths returns the binary paths for AWS CLI installations.
func (*AwscliPlugin) ListBinPaths() string {
	return "bin"
}

// ExecEnv returns environment variables for AWS CLI execution.
func (*AwscliPlugin) ExecEnv(_ string) map[string]string {
	return nil
}

// ListLegacyFilenames returns legacy version filenames for AWS CLI.
func (*AwscliPlugin) ListLegacyFilenames() []string {
	return nil
}

// ParseLegacyFile parses a legacy AWS CLI version file.
func (*AwscliPlugin) ParseLegacyFile(path string) (string, error) {
	return asdf.ReadLegacyVersionFile(path)
}

// Uninstall removes an AWS CLI installation.
func (*AwscliPlugin) Uninstall(_ context.Context, installPath string) error {
	return os.RemoveAll(installPath)
}

// Help returns help information for the AWS CLI plugin.
func (*AwscliPlugin) Help() asdf.PluginHelp {
	return asdf.PluginHelp{
		Overview: `AWS CLI - The AWS Command Line Interface.
This plugin downloads pre-built AWS CLI v2 binaries.`,
		Deps: `Linux: glibc, groff, less
macOS: Rosetta 2 (for Apple Silicon)`,
		Config: `Environment variables:
  AWS_CONFIG_FILE - Override AWS awscliConfig file location
  AWS_SHARED_CREDENTIALS_FILE - Override credentials file location`,
		Links: `Homepage: https://aws.amazon.com/cli/
Documentation: https://docs.aws.amazon.com/cli/
Source: https://github.com/aws/aws-cli`,
	}
}

// ListAll lists all available AWS CLI versions.
func (plugin *AwscliPlugin) ListAll(ctx context.Context) ([]string, error) {
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
func (plugin *AwscliPlugin) LatestStable(ctx context.Context, query string) (string, error) {
	versions, err := plugin.ListAll(ctx)
	if err != nil {
		return "", err
	}

	return asdf.LatestStableWithQuery(
		ctx,
		query,
		versions,
		errAWSNoVersionsFound,
		errAWSNoVersionsMatching,
	)
}

// getDownloadURL returns the download URL for the specified version.
func (*AwscliPlugin) getDownloadURL(version string) (string, error) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return fmt.Sprintf(
				"%s/awscli-exe-linux-x86_64-%s.zip",
				awscliDownloadBaseURL,
				version,
			), nil
		case "arm64":
			return fmt.Sprintf(
				"%s/awscli-exe-linux-aarch64-%s.zip",
				awscliDownloadBaseURL,
				version,
			), nil
		}

	case "darwin":
		return fmt.Sprintf("%s/AWSCLIV2-%s.pkg", awscliDownloadBaseURL, version), nil
	}

	return "", fmt.Errorf("%w: %s/%s", errAWSUnsupportedPlatform, runtime.GOOS, runtime.GOARCH)
}

// Download downloads the specified AWS CLI version.
func (plugin *AwscliPlugin) Download(ctx context.Context, version, downloadPath string) error {
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading awscli: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w with status: %d", errAWSDownloadFailed, resp.StatusCode)
	}

	outFile, err := os.OpenFile(
		filePath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		asdf.CommonFilePermission,
	)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	if strings.HasSuffix(filename, ".zip") {
		err := asdf.ExtractZip(filePath, downloadPath)
		if err != nil {
			return fmt.Errorf("extracting zip: %w", err)
		}
	}

	return nil
}

// Install installs AWS CLI from the downloaded files.
func (plugin *AwscliPlugin) Install(
	ctx context.Context,
	version, downloadPath, installPath string,
) error {
	switch runtime.GOOS {
	case "linux":
		return plugin.installLinux(ctx, downloadPath, installPath)
	case "darwin":
		return plugin.installDarwin(ctx, version, downloadPath, installPath)
	default:
		return fmt.Errorf("%w: %s", errAWSUnsupportedInstallOS, runtime.GOOS)
	}
}

// installLinux installs AWS CLI on Linux.
func (*AwscliPlugin) installLinux(ctx context.Context, downloadPath, installPath string) error {
	installerPath := filepath.Join(downloadPath, "aws", "install")

	err := os.Chmod(installerPath, asdf.CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("making installer executable: %w", err)
	}

	binDir := filepath.Join(installPath, "bin")
	libDir := filepath.Join(installPath, "aws-cli")

	cmd := exec.CommandContext(ctx, installerPath, "-i", libDir, "-b", binDir)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("running installer: %w", err)
	}

	return nil
}

// installDarwin installs AWS CLI on macOS.
func (*AwscliPlugin) installDarwin(
	ctx context.Context,
	version, downloadPath, installPath string,
) error {
	pkgPath := filepath.Join(downloadPath, fmt.Sprintf("AWSCLIV2-%s.pkg", version))

	binDir := filepath.Join(installPath, "bin")

	err := os.MkdirAll(binDir, asdf.CommonDirectoryPermission)
	if err != nil {
		return fmt.Errorf("creating bin directory: %w", err)
	}

	extractDir := filepath.Join(downloadPath, "extracted")

	cmd := exec.CommandContext(ctx, "pkgutil", "--expand-full", pkgPath, extractDir)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("extracting pkg: %w", err)
	}

	awsCliSrc := filepath.Join(extractDir, "aws-cli.pkg", "Payload", "aws-cli")
	awsCliDst := filepath.Join(installPath, "aws-cli")

	cmd = exec.CommandContext(ctx, "cp", "-r", awsCliSrc, awsCliDst)

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("copying aws-cli: %w", err)
	}

	awsBin := filepath.Join(awsCliDst, "aws")
	awsCompleter := filepath.Join(awsCliDst, "aws_completer")

	err = os.Symlink(awsBin, filepath.Join(binDir, "aws"))
	if err != nil {
		return fmt.Errorf("creating aws symlink: %w", err)
	}

	err = os.Symlink(awsCompleter, filepath.Join(binDir, "aws_completer"))
	if err != nil {
		return fmt.Errorf("creating aws_completer symlink: %w", err)
	}

	return nil
}
