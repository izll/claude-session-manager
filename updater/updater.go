package updater

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	RepoOwner    = "izll"
	RepoName     = "agent-session-manager"
	BinaryName   = "asmgr"
	CheckTimeout = 5 * time.Second
)

type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
}

// CheckForUpdate checks if a newer version is available
// Returns the new version string if available, empty string if up to date
func CheckForUpdate(currentVersion string) string {
	client := &http.Client{Timeout: CheckTimeout}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)

	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	currentVer := strings.TrimPrefix(currentVersion, "v")
	latestVer := strings.TrimPrefix(release.TagName, "v")

	if latestVer != currentVer && latestVer > currentVer {
		return release.TagName
	}

	return ""
}

// DownloadAndInstall downloads and installs the specified version
func DownloadAndInstall(version string) error {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	verNum := strings.TrimPrefix(version, "v")
	filename := fmt.Sprintf("%s_%s_%s_%s.tar.gz", BinaryName, verNum, osName, arch)
	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		RepoOwner, RepoName, version, filename)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("cannot resolve symlinks: %w", err)
	}

	// Extract binary from tarball
	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var binaryData []byte
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		if header.Name == BinaryName {
			binaryData, err = io.ReadAll(tarReader)
			if err != nil {
				return fmt.Errorf("failed to read binary: %w", err)
			}
			break
		}
	}

	if binaryData == nil {
		return fmt.Errorf("binary not found in archive")
	}

	// Write new binary
	tmpPath := execPath + ".new"
	if err := os.WriteFile(tmpPath, binaryData, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// Replace old binary
	oldPath := execPath + ".old"
	os.Remove(oldPath)

	if err := os.Rename(execPath, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup old binary: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Rename(oldPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	os.Remove(oldPath)

	return nil
}
