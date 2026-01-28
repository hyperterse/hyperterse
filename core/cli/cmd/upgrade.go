package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	upgradePrerelease bool
	upgradeMajor      string
)

var upgradeCmd = &cobra.Command{
	Use:          "upgrade",
	Short:        "Upgrade Hyperterse to the latest version",
	RunE:         runUpgrade,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradePrerelease, "prerelease", false, "Include pre-releases when finding the latest version")
	upgradeCmd.Flags().StringVar(&upgradeMajor, "major", "", "Upgrade to a specific major version (e.g., '2') or use 'next' to upgrade to the next major version")
}

type Release struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	// Get current version
	currentVersion, err := getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current version: %s\n", currentVersion)

	// Auto-detect if current version is a prerelease and include prereleases automatically
	// unless explicitly disabled by user
	if !upgradePrerelease && isPrereleaseVersion(currentVersion) {
		upgradePrerelease = true
		fmt.Printf("Current version is a prerelease, including prereleases in search\n")
	}

	// Determine target major version
	var targetMajorVersion int
	if upgradeMajor != "" {
		if upgradeMajor == "next" {
			// Bump to next major version
			currentMajor, err := parseMajorVersion(currentVersion)
			if err != nil {
				return fmt.Errorf("failed to parse current version: %w", err)
			}
			targetMajorVersion = currentMajor + 1
			fmt.Printf("Upgrading to next major version: %d\n", targetMajorVersion)
		} else {
			// Parse the provided major version
			if _, err := fmt.Sscanf(upgradeMajor, "%d", &targetMajorVersion); err != nil {
				return fmt.Errorf("invalid major version: %s (must be a number or 'next')", upgradeMajor)
			}
			fmt.Printf("Upgrading to major version: %d\n", targetMajorVersion)
		}
	} else {
		// Use current major version
		var err error
		targetMajorVersion, err = parseMajorVersion(currentVersion)
		if err != nil {
			return fmt.Errorf("failed to parse version: %w", err)
		}
		fmt.Printf("Staying on major version: %d\n", targetMajorVersion)
	}

	// Fetch all releases
	releases, err := fetchReleases()
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}

	// Filter to target major version and find latest
	latestVersion, err := findLatestInMajorVersion(releases, targetMajorVersion, upgradePrerelease)
	if err != nil {
		return fmt.Errorf("failed to find latest version: %w", err)
	}

	if latestVersion == "" {
		// No releases found for the target major version
		return fmt.Errorf("no releases found for major version %d", targetMajorVersion)
	}

	// Check if already on latest
	if latestVersion == currentVersion {
		fmt.Printf("Already on latest version: %s\n", currentVersion)
		return nil
	}

	fmt.Printf("Latest version in major version %d: %s\n", targetMajorVersion, latestVersion)

	// Download and install
	if err := downloadAndInstall(latestVersion); err != nil {
		return fmt.Errorf("failed to download and install: %w", err)
	}

	fmt.Printf("Successfully upgraded to %s\n", latestVersion)
	return nil
}

func getCurrentVersion() (string, error) {
	// First, try to use the version from GetVersion()
	currentVersion := GetVersion()
	if currentVersion != "dev" {
		version := strings.TrimPrefix(currentVersion, "v")
		return version, nil
	}

	// Try to get version from the binary itself using --version flag
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(execPath, "--version")
	output, err := cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		// Remove 'v' prefix if present
		version = strings.TrimPrefix(version, "v")
		if version != "" && version != "dev" {
			return version, nil
		}
	}

	// Fallback: try to get from git if we're in a git repo
	if gitVersion, err := getVersionFromGit(); err == nil {
		return gitVersion, nil
	}

	return "", fmt.Errorf("could not determine current version")
}

func getVersionFromGit() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	// Remove -dirty suffix if present
	version = strings.TrimSuffix(version, "-dirty")
	// Remove commit hash suffix (e.g., v1.0.0-5-gabc1234 -> v1.0.0)
	if idx := strings.LastIndex(version, "-"); idx > 0 {
		parts := strings.Split(version, "-")
		if len(parts) >= 2 {
			// Check if it's a version tag format
			if strings.Count(parts[0], ".") == 2 {
				version = parts[0]
			}
		}
	}
	return version, nil
}

func isPrereleaseVersion(version string) bool {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	// Check if version contains a prerelease suffix (e.g., 1.0.0-alpha.1, 1.0.0-beta.1)
	return strings.Contains(version, "-")
}

func parseMajorVersion(version string) (int, error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Handle prerelease versions (e.g., 1.0.0-alpha.1)
	parts := strings.Split(version, "-")
	version = parts[0]

	// Split by dots
	parts = strings.Split(version, ".")
	if len(parts) < 1 {
		return 0, fmt.Errorf("invalid version format: %s", version)
	}

	var major int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	return major, nil
}

func fetchReleases() ([]Release, error) {
	url := "https://api.github.com/repos/hyperterse/hyperterse/releases"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch releases: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func versionExistsInReleases(releases []Release, version string) bool {
	version = strings.TrimPrefix(version, "v")
	for _, release := range releases {
		releaseVersion := strings.TrimPrefix(release.TagName, "v")
		if releaseVersion == version {
			return true
		}
	}
	return false
}

// comparePrereleaseSuffixes compares two prerelease suffixes according to SemVer rules
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func comparePrereleaseSuffixes(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return -1 // Empty is older than any prerelease
	}
	if b == "" {
		return 1 // Any prerelease is newer than empty
	}

	// Split by dots to get individual identifiers
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Compare each identifier
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aPart, bPart string
		if i < len(aParts) {
			aPart = aParts[i]
		}
		if i < len(bParts) {
			bPart = bParts[i]
		}

		// If one is missing, the longer one is newer (if all previous match)
		if aPart == "" {
			return -1
		}
		if bPart == "" {
			return 1
		}

		// Try to parse as numbers
		var aVal, bVal int
		_, aErr := fmt.Sscanf(aPart, "%d", &aVal)
		_, bErr := fmt.Sscanf(bPart, "%d", &bVal)

		// Both are numeric - compare numerically
		if aErr == nil && bErr == nil {
			if aVal < bVal {
				return -1
			}
			if aVal > bVal {
				return 1
			}
			continue
		}

		// Numeric identifiers come before string identifiers
		if aErr == nil {
			return -1
		}
		if bErr == nil {
			return 1
		}

		// Both are strings - compare lexicographically
		if aPart < bPart {
			return -1
		}
		if aPart > bPart {
			return 1
		}
	}

	return 0
}

func findLatestInMajorVersion(releases []Release, majorVersion int, includePrerelease bool) (string, error) {
	var latestVersion string
	var latestMinor, latestPatch int = -1, -1
	var latestPrerelease string = ""

	for _, release := range releases {
		// Skip prereleases if not including them
		if !includePrerelease && release.Prerelease {
			continue
		}

		// Parse version
		version := strings.TrimPrefix(release.TagName, "v")
		major, err := parseMajorVersion(version)
		if err != nil {
			continue
		}

		// Only consider same major version
		if major != majorVersion {
			continue
		}

		// Debug: log that we found a matching release
		// fmt.Printf("DEBUG: Found release %s (prerelease: %v, major: %d)\n", version, release.Prerelease, major)

		// Parse full version
		// First, separate base version from prerelease suffix (e.g., "1.0.0-alpha.0" -> "1.0.0" and "alpha.0")
		var baseVersion string
		var prereleaseSuffix string
		if strings.Contains(version, "-") {
			versionParts := strings.SplitN(version, "-", 2)
			baseVersion = versionParts[0]
			prereleaseSuffix = versionParts[1]
		} else {
			baseVersion = version
		}

		// Parse base version (e.g., "1.0.0")
		parts := strings.Split(baseVersion, ".")
		if len(parts) < 3 {
			continue
		}

		var minor, patch int
		if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
			continue
		}

		if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
			continue
		}

		// Compare versions
		// Stable releases are always preferred over prereleases (when includePrerelease is true)
		isCurrentPrerelease := release.Prerelease
		isLatestPrerelease := latestPrerelease != ""

		// If current is stable and latest is prerelease, current wins
		if !isCurrentPrerelease && isLatestPrerelease {
			latestMinor = minor
			latestPatch = patch
			latestPrerelease = ""
			latestVersion = version
			continue
		}

		// If current is prerelease and latest is stable, skip (stable always wins)
		// But only if we already have a stable version (latestMinor != -1)
		if isCurrentPrerelease && !isLatestPrerelease && latestMinor != -1 {
			continue
		}

		// Both are same type (both stable or both prerelease)
		// Compare by version numbers first
		// If latestMinor is -1, this is the first version, so always select it
		versionIsNewer := latestMinor == -1 || minor > latestMinor ||
			(minor == latestMinor && patch > latestPatch)

		// If version numbers are equal and both are prereleases, compare prerelease suffixes
		prereleaseSuffixIsNewer := false
		if isCurrentPrerelease {
			if latestMinor == -1 {
				// First version ever - select it
				prereleaseSuffixIsNewer = true
			} else if minor == latestMinor && patch == latestPatch {
				// Same version numbers, compare prerelease suffixes
				if latestPrerelease == "" {
					// We have a stable version already, prerelease can't be newer
					prereleaseSuffixIsNewer = false
				} else {
					// Compare prerelease suffixes using SemVer comparison rules
					// Returns 1 if prereleaseSuffix > latestPrerelease
					prereleaseSuffixIsNewer = comparePrereleaseSuffixes(prereleaseSuffix, latestPrerelease) > 0
				}
			}
		}

		shouldUpdate := versionIsNewer || prereleaseSuffixIsNewer
		if shouldUpdate {
			latestMinor = minor
			latestPatch = patch
			if isCurrentPrerelease {
				latestPrerelease = prereleaseSuffix
			} else {
				latestPrerelease = ""
			}
			latestVersion = version
		}
	}

	return latestVersion, nil
}

func downloadAndInstall(version string) error {
	// Detect OS and architecture
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Normalize architecture names
	if goarch == "aarch64" {
		goarch = "arm64"
	}

	// Build binary name
	binaryName := fmt.Sprintf("hyperterse-%s-%s", goos, goarch)
	if goos == "windows" {
		binaryName += ".exe"
	}

	// Build download URL
	versionTag := version
	if !strings.HasPrefix(versionTag, "v") {
		versionTag = "v" + versionTag
	}
	url := fmt.Sprintf("https://github.com/hyperterse/hyperterse/releases/download/%s/%s", versionTag, binaryName)

	fmt.Printf("Downloading from: %s\n", url)

	// Download the binary
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(execPath)
	if err == nil {
		execPath = realPath
	}

	// Create temporary file for download
	tmpFile := execPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	// Copy downloaded content to temp file
	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Make executable (Unix-like systems only)
	if goos != "windows" {
		if err := os.Chmod(tmpFile, 0755); err != nil {
			os.Remove(tmpFile)
			return err
		}
	}

	// Replace the binary
	// On Windows, we need to remove the old file first
	if goos == "windows" {
		// Try to remove the old file (may fail if in use, but that's okay for now)
		os.Remove(execPath)
	}

	if err := os.Rename(tmpFile, execPath); err != nil {
		// If rename fails on Windows, try copy and remove
		if goos == "windows" {
			// Read the temp file
			data, readErr := os.ReadFile(tmpFile)
			if readErr != nil {
				os.Remove(tmpFile)
				return fmt.Errorf("failed to read downloaded file: %w", readErr)
			}
			// Write to destination
			if writeErr := os.WriteFile(execPath, data, 0755); writeErr != nil {
				os.Remove(tmpFile)
				return fmt.Errorf("failed to write binary: %w", writeErr)
			}
			os.Remove(tmpFile)
		} else {
			os.Remove(tmpFile)
			return err
		}
	}

	return nil
}
