package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Tencent/Xelora/internal/handler"
	"github.com/Tencent/Xelora/internal/logger"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/mod/semver"
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

const internalReleaseManifestSchemaVersion = 1

var (
	internalReleaseFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.zip$`)
	internalReleaseChecksumPattern = regexp.MustCompile(`^[A-Fa-f0-9]{64}$`)
)

type internalReleaseAsset struct {
	Version     string `json:"version"`
	Build       string `json:"build,omitempty"`
	File        string `json:"file"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"sizeBytes"`
	PublishedAt string `json:"publishedAt,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type internalReleaseManifest struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Product       string                 `json:"product"`
	Channel       string                 `json:"channel"`
	Latest        internalReleaseAsset   `json:"latest"`
	Releases      []internalReleaseAsset `json:"releases,omitempty"`
}

func desktopUpdateFeedURL() string {
	if feedURL := strings.TrimSpace(os.Getenv("XELORA_UPDATE_FEED_URL")); feedURL != "" {
		return feedURL
	}
	if desktopBrandProfileIsInternal() {
		if feedURL := strings.TrimSpace(packagedDesktopUpdateFeedURL); feedURL != "" {
			return feedURL
		}
		// The linker-injected feed is required for production internal builds
		// because the frontend can be on a different port than the API server.
		// This fallback keeps local developer packages usable when both paths
		// are served by the same origin.
		return strings.TrimRight(defaultEnterpriseServerURL(), "/") + "/downloads/channels/stable.json"
	}
	return "https://api.github.com/repos/Tencent/Xelora/releases/latest"
}

func checkUpdate(ctx context.Context, currentVersion string, showUpToDate bool, autoDownload bool) {
	go func() {
		if currentVersion == "unknown" || currentVersion == "" {
			if showUpToDate {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.InfoDialog,
					Title:         "Check for Updates",
					Message:       "Unable to determine the current version. Cannot check for updates.",
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}

		if !strings.HasPrefix(currentVersion, "v") {
			currentVersion = "v" + currentVersion
		}

		client := &http.Client{Timeout: 10 * time.Second}
		if desktopBrandProfileIsInternal() {
			checkInternalUpdate(ctx, client, currentVersion, showUpToDate, autoDownload)
			return
		}

		// Public builds retain the upstream GitHub feed and its release format.
		feedURL := desktopUpdateFeedURL()
		if feedURL == "" {
			if showUpToDate {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.InfoDialog,
					Title:         "Check for Updates",
					Message:       fmt.Sprintf("No update feed is configured for %s.", desktopProductName()),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}
		req, err := http.NewRequest("GET", feedURL, nil)
		if err != nil {
			logger.Warnf(context.Background(), "Check update failed: %v", err)
			return
		}

		// Add User-Agent header which is required/recommended by GitHub API
		req.Header.Set("User-Agent", strings.ReplaceAll(desktopProductName(), " ", "-")+"-Desktop-App")

		// Add Authorization header if GITHUB_TOKEN is present to increase rate limit
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			logger.Warnf(context.Background(), "Check update failed: %v", err)
			if showUpToDate {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Check Update Failed",
					Message:       fmt.Sprintf("Failed to connect to GitHub to check for updates:\n%v", err),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Warnf(context.Background(), "GitHub API returned status code: %d", resp.StatusCode)
			if showUpToDate {
				msg := fmt.Sprintf("GitHub API returned an unexpected status code: %d", resp.StatusCode)
				if resp.StatusCode == 403 {
					msg = "GitHub API rate limit exceeded. Please try again later."
				}
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Check Update Failed",
					Message:       msg,
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}

		var release githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			logger.Warnf(context.Background(), "Failed to parse release info: %v", err)
			return
		}

		latestVersion := release.TagName
		if !strings.HasPrefix(latestVersion, "v") {
			latestVersion = "v" + latestVersion
		}

		if semver.IsValid(latestVersion) && semver.IsValid(currentVersion) {
			if semver.Compare(latestVersion, currentVersion) > 0 {
				assetURL, assetName := findBestAsset(release.Assets, runtime.GOOS, runtime.GOARCH)

				if autoDownload && assetURL != "" {
					// Silent download in background
					downloadAndInstall(ctx, assetURL, assetName, currentVersion, latestVersion, true)
					return
				}

				msg := fmt.Sprintf("A new version of %s is available!\n\nCurrent version: %s\nLatest version: %s\n\nWould you like to download it now?", desktopProductName(), currentVersion, latestVersion)
				choice, _ := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.InfoDialog,
					Title:         "Update Available",
					Message:       msg,
					Buttons:       []string{"Download", "Cancel"},
					DefaultButton: "Download",
				})
				if choice == "Download" {
					if assetURL != "" {
						downloadAndInstall(ctx, assetURL, assetName, currentVersion, latestVersion, false)
					} else {
						// Fallback to opening the release page if no specific asset is found
						wailsruntime.BrowserOpenURL(ctx, release.HTMLURL)
					}
				}
				return
			}
		}

		if showUpToDate {
			wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type:          wailsruntime.InfoDialog,
				Title:         "Up to Date",
				Message:       fmt.Sprintf("You are using the latest version of %s.\n\nCurrent version: %s", desktopProductName(), currentVersion),
				Buttons:       []string{"OK"},
				DefaultButton: "OK",
			})
		}
	}()
}

func checkInternalUpdate(ctx context.Context, client *http.Client, currentVersion string, showUpToDate bool, autoDownload bool) {
	feedURL := desktopUpdateFeedURL()
	if feedURL == "" {
		if showUpToDate {
			wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type:          wailsruntime.ErrorDialog,
				Title:         "Check Update Failed",
				Message:       "No Knavis release channel is configured.",
				Buttons:       []string{"OK"},
				DefaultButton: "OK",
			})
		}
		return
	}

	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		showInternalUpdateError(ctx, err, showUpToDate)
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", strings.ReplaceAll(desktopProductName(), " ", "-")+"-Desktop-App")

	resp, err := client.Do(req)
	if err != nil {
		showInternalUpdateError(ctx, fmt.Errorf("connect to Knavis release channel: %w", err), showUpToDate)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		showInternalUpdateError(ctx, fmt.Errorf("release channel returned HTTP %d", resp.StatusCode), showUpToDate)
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		showInternalUpdateError(ctx, fmt.Errorf("read release channel: %w", err), showUpToDate)
		return
	}
	manifest, err := parseInternalReleaseManifest(body)
	if err != nil {
		showInternalUpdateError(ctx, fmt.Errorf("invalid Knavis release channel: %w", err), showUpToDate)
		return
	}

	currentSemver, err := normalizeReleaseVersion(currentVersion)
	if err != nil {
		showInternalUpdateError(ctx, fmt.Errorf("invalid current client version %q: %w", currentVersion, err), showUpToDate)
		return
	}
	latestSemver, err := normalizeReleaseVersion(manifest.Latest.Version)
	if err != nil {
		showInternalUpdateError(ctx, fmt.Errorf("invalid latest client version: %w", err), showUpToDate)
		return
	}

	if semver.Compare(latestSemver, currentSemver) > 0 {
		downloadURL, err := internalReleaseDownloadURL(feedURL, manifest.Latest.File)
		if err != nil {
			showInternalUpdateError(ctx, err, showUpToDate)
			return
		}
		if autoDownload {
			downloadAndVerifyInternalRelease(ctx, downloadURL, manifest.Latest, true)
			return
		}

		message := fmt.Sprintf(
			"A verified %s update is available.\n\nCurrent version: %s\nLatest version: %s",
			desktopProductName(), currentVersion, latestSemver,
		)
		if notes := strings.TrimSpace(manifest.Latest.Notes); notes != "" {
			message += "\n\nRelease notes:\n" + notes
		}
		choice, _ := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type:          wailsruntime.InfoDialog,
			Title:         "Update Available",
			Message:       message,
			Buttons:       []string{"Download", "Cancel"},
			DefaultButton: "Download",
		})
		if choice == "Download" {
			downloadAndVerifyInternalRelease(ctx, downloadURL, manifest.Latest, false)
		}
		return
	}

	if showUpToDate {
		wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type:          wailsruntime.InfoDialog,
			Title:         "Up to Date",
			Message:       fmt.Sprintf("You are using the latest version of %s.\n\nCurrent version: %s", desktopProductName(), currentVersion),
			Buttons:       []string{"OK"},
			DefaultButton: "OK",
		})
	}
}

func normalizeReleaseVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	if !semver.IsValid(value) {
		return "", fmt.Errorf("invalid semantic version %q", value)
	}
	return value, nil
}

func parseInternalReleaseManifest(raw []byte) (internalReleaseManifest, error) {
	var manifest internalReleaseManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return internalReleaseManifest{}, err
	}
	if manifest.SchemaVersion != internalReleaseManifestSchemaVersion {
		return internalReleaseManifest{}, fmt.Errorf("unsupported schemaVersion %d", manifest.SchemaVersion)
	}
	if !strings.EqualFold(strings.TrimSpace(manifest.Product), "Knavis Personal") {
		return internalReleaseManifest{}, fmt.Errorf("unexpected product %q", manifest.Product)
	}
	manifest.Channel = strings.ToLower(strings.TrimSpace(manifest.Channel))
	if manifest.Channel != "stable" && manifest.Channel != "beta" {
		return internalReleaseManifest{}, fmt.Errorf("unsupported channel %q", manifest.Channel)
	}
	if err := validateInternalReleaseAsset(manifest.Latest, manifest.Channel); err != nil {
		return internalReleaseManifest{}, fmt.Errorf("latest release: %w", err)
	}
	return manifest, nil
}

func validateInternalReleaseAsset(asset internalReleaseAsset, channel string) error {
	version, err := normalizeReleaseVersion(asset.Version)
	if err != nil {
		return err
	}
	if channel == "stable" && strings.Contains(version, "-") {
		return fmt.Errorf("stable channel cannot publish prerelease %q", asset.Version)
	}
	if !internalReleaseFilenamePattern.MatchString(asset.File) {
		return fmt.Errorf("unsafe ZIP filename %q", asset.File)
	}
	if !internalReleaseChecksumPattern.MatchString(asset.SHA256) {
		return fmt.Errorf("invalid SHA-256 checksum")
	}
	if asset.SizeBytes <= 0 {
		return fmt.Errorf("sizeBytes must be positive")
	}
	return nil
}

func internalReleaseDownloadURL(feedURL, filename string) (string, error) {
	if !internalReleaseFilenamePattern.MatchString(filename) {
		return "", fmt.Errorf("unsafe release filename %q", filename)
	}
	parsed, err := url.Parse(strings.TrimSpace(feedURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid release feed URL %q", feedURL)
	}
	feedPath := path.Clean(parsed.Path)
	if path.Base(path.Dir(feedPath)) != "channels" || path.Base(path.Dir(path.Dir(feedPath))) != "downloads" {
		return "", fmt.Errorf("release feed must be under /downloads/channels")
	}
	parsed.Path = path.Join(path.Dir(path.Dir(feedPath)), filename)
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func showInternalUpdateError(ctx context.Context, err error, showDialog bool) {
	logger.Warnf(context.Background(), "Knavis update check failed: %v", err)
	if !showDialog {
		return
	}
	wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.ErrorDialog,
		Title:         "Check Update Failed",
		Message:       err.Error(),
		Buttons:       []string{"OK"},
		DefaultButton: "OK",
	})
}

func internalReleaseDownloadDir() string {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		dir := filepath.Join(home, "Downloads", "Knavis Updates")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			return dir
		}
	}
	return os.TempDir()
}

func fileMatchesInternalRelease(filePath string, asset internalReleaseAsset) (bool, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Size() != asset.SizeBytes {
		return false, nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, err
	}
	return strings.EqualFold(hex.EncodeToString(hasher.Sum(nil)), asset.SHA256), nil
}

func downloadAndVerifyInternalRelease(ctx context.Context, downloadURL string, asset internalReleaseAsset, silent bool) {
	go func() {
		downloadDir := internalReleaseDownloadDir()
		savePath := filepath.Join(downloadDir, asset.File)
		if matched, err := fileMatchesInternalRelease(savePath, asset); err != nil {
			showInternalUpdateError(ctx, fmt.Errorf("validate existing update: %w", err), !silent)
			return
		} else if matched {
			logger.Infof(context.Background(), "Verified Knavis update already exists at %s", savePath)
			if !silent {
				openInternalReleaseLocation(ctx, savePath)
			}
			return
		}
		if _, err := os.Stat(savePath); err == nil {
			showInternalUpdateError(ctx, fmt.Errorf("refusing to overwrite an existing update with a different checksum: %s", savePath), !silent)
			return
		} else if !os.IsNotExist(err) {
			showInternalUpdateError(ctx, fmt.Errorf("inspect update destination: %w", err), !silent)
			return
		}

		temporaryFile, err := os.CreateTemp(downloadDir, ".knavis-update-*.part")
		if err != nil {
			showInternalUpdateError(ctx, fmt.Errorf("create update file: %w", err), !silent)
			return
		}
		temporaryPath := temporaryFile.Name()
		defer os.Remove(temporaryPath)

		client := &http.Client{Timeout: 30 * time.Minute}
		response, err := client.Get(downloadURL)
		if err != nil {
			showInternalUpdateError(ctx, fmt.Errorf("download update: %w", err), !silent)
			_ = temporaryFile.Close()
			return
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			_ = temporaryFile.Close()
			showInternalUpdateError(ctx, fmt.Errorf("download server returned HTTP %d", response.StatusCode), !silent)
			return
		}

		hasher := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(temporaryFile, hasher), response.Body)
		response.Body.Close()
		closeErr := temporaryFile.Close()
		if copyErr != nil {
			showInternalUpdateError(ctx, fmt.Errorf("write update: %w", copyErr), !silent)
			return
		}
		if closeErr != nil {
			showInternalUpdateError(ctx, fmt.Errorf("close update file: %w", closeErr), !silent)
			return
		}
		if written != asset.SizeBytes {
			showInternalUpdateError(ctx, fmt.Errorf("update size mismatch: got %d bytes, expected %d", written, asset.SizeBytes), !silent)
			return
		}
		actualChecksum := hex.EncodeToString(hasher.Sum(nil))
		if !strings.EqualFold(actualChecksum, asset.SHA256) {
			showInternalUpdateError(ctx, fmt.Errorf("update SHA-256 mismatch: got %s", actualChecksum), !silent)
			return
		}
		if err := os.Rename(temporaryPath, savePath); err != nil {
			showInternalUpdateError(ctx, fmt.Errorf("finalize verified update: %w", err), !silent)
			return
		}
		temporaryPath = ""
		logger.Infof(context.Background(), "Downloaded and verified Knavis update %s at %s", asset.Version, savePath)
		if silent {
			return
		}
		choice, _ := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type:          wailsruntime.InfoDialog,
			Title:         "Update Downloaded",
			Message:       fmt.Sprintf("%s %s was downloaded and verified. Close the current client, extract the ZIP, and start the new client from the extracted folder.", desktopProductName(), asset.Version),
			Buttons:       []string{"Open Folder", "Later"},
			DefaultButton: "Open Folder",
		})
		if choice == "Open Folder" {
			openInternalReleaseLocation(ctx, savePath)
		}
	}()
}

func openInternalReleaseLocation(_ context.Context, filePath string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("explorer.exe", "/select,", filePath).Start()
	case "darwin":
		_ = exec.Command("open", "-R", filePath).Start()
	default:
		_ = exec.Command("xdg-open", filepath.Dir(filePath)).Start()
	}
}

func findBestAsset(assets []githubAsset, goos, goarch string) (string, string) {
	osKeyword := ""
	switch goos {
	case "darwin":
		osKeyword = "mac"
	case "windows":
		osKeyword = "win"
	case "linux":
		osKeyword = "linux"
	}

	archKeyword := ""
	switch goarch {
	case "amd64":
		archKeyword = "amd64"
	case "arm64":
		archKeyword = "arm64"
	}

	// 1. Try to match both OS and Arch
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, osKeyword) && (strings.Contains(name, archKeyword) || strings.Contains(name, "universal") || strings.Contains(name, "aarch64")) {
			return asset.BrowserDownloadURL, asset.Name
		}
	}

	// 2. Try to match OS only (e.g. universal binaries without arch in name)
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, osKeyword) {
			return asset.BrowserDownloadURL, asset.Name
		}
	}

	// 3. MacOS specific fallback (e.g. .dmg)
	if goos == "darwin" {
		for _, asset := range assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".dmg") {
				return asset.BrowserDownloadURL, asset.Name
			}
		}
	}

	// 4. Windows specific fallback (e.g. .exe)
	if goos == "windows" {
		for _, asset := range assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
				return asset.BrowserDownloadURL, asset.Name
			}
		}
	}

	return "", ""
}

func downloadAndInstall(ctx context.Context, url string, filename string, currentVersion string, latestVersion string, silent bool) {
	tempDir := os.TempDir()
	savePath := filepath.Join(tempDir, filename)

	go func() {
		logger.Infof(context.Background(), "Starting download from %s to %s", url, savePath)
		resp, err := http.Get(url)
		if err != nil {
			logger.Warnf(context.Background(), "Download failed: %v", err)
			if !silent {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Download Failed",
					Message:       fmt.Sprintf("Failed to download the update:\n%v", err),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Warnf(context.Background(), "Download failed, server returned status: %d", resp.StatusCode)
			if !silent {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Download Failed",
					Message:       fmt.Sprintf("Server returned status: %d", resp.StatusCode),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}

		out, err := os.Create(savePath)
		if err != nil {
			logger.Warnf(context.Background(), "Failed to create file for download: %v", err)
			if !silent {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Save Failed",
					Message:       fmt.Sprintf("Failed to save the update file:\n%v", err),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			logger.Warnf(context.Background(), "Error occurred during download copying: %v", err)
			if !silent {
				wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
					Type:          wailsruntime.ErrorDialog,
					Title:         "Download Error",
					Message:       fmt.Sprintf("An error occurred while downloading:\n%v", err),
					Buttons:       []string{"OK"},
					DefaultButton: "OK",
				})
			}
			return
		}

		logger.Infof(context.Background(), "Download completed successfully: %s", savePath)

		// Prompt user to restart
		choice, _ := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type:          wailsruntime.InfoDialog,
			Title:         "Update Ready",
			Message:       fmt.Sprintf("%s %s has been downloaded successfully.\n\nWould you like to restart and install the new version now?", desktopProductName(), latestVersion),
			Buttons:       []string{"Restart Now", "Later"},
			DefaultButton: "Restart Now",
		})

		if choice == "Restart Now" {
			applyUpdateAndRestart(ctx, savePath)
		}
	}()
}

// desktopAboutVersion 优先使用构建脚本注入的 handler.Version，否则尝试读取仓库根目录 VERSION（本地 wails dev 等未带 ldflags 时）。
func desktopAboutVersion() string {
	if v := strings.TrimSpace(handler.Version); v != "" && v != "unknown" {
		return v
	}
	for _, p := range []string{
		"VERSION",
		filepath.Join("..", "..", "VERSION"),
		filepath.Join("..", "..", "..", "VERSION"),
	} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	return "unknown"
}

// applyUpdateAndRestart applies the downloaded update and restarts the application
func applyUpdateAndRestart(ctx context.Context, savePath string) {
	if runtime.GOOS == "windows" {
		scriptPath := filepath.Join(os.TempDir(), strings.ToLower(desktopBrandName())+"_update.bat")
		execPath, err := os.Executable()
		if err != nil {
			logger.Warnf(context.Background(), "Failed to get executable path: %v", err)
			return
		}
		scriptContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak
start /wait "" "%s" /S
start "" "%s"
del "%%~f0"
`, savePath, execPath)
		os.WriteFile(scriptPath, []byte(scriptContent), 0755)

		cmd := exec.Command("cmd.exe", "/C", "start", "/b", scriptPath)
		cmd.Start()
		wailsruntime.Quit(ctx)

	} else if runtime.GOOS == "darwin" {
		if strings.HasSuffix(strings.ToLower(savePath), ".dmg") {
			go func() {
				execPath, err := os.Executable()
				if err != nil {
					logger.Warnf(context.Background(), "Failed to get executable path: %v", err)
					exec.Command("open", savePath).Start()
					wailsruntime.Quit(ctx)
					return
				}

				appBundlePath := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
				if !strings.HasSuffix(appBundlePath, ".app") {
					exec.Command("open", savePath).Start()
					wailsruntime.Quit(ctx)
					return
				}

				mountPoint := filepath.Join(os.TempDir(), "XeloraUpdateMount")
				os.MkdirAll(mountPoint, 0755)

				cmdMount := exec.Command("hdiutil", "attach", savePath, "-mountpoint", mountPoint, "-nobrowse", "-quiet")
				if err := cmdMount.Run(); err != nil {
					logger.Warnf(context.Background(), "Failed to mount dmg: %v", err)
					exec.Command("open", savePath).Start()
					wailsruntime.Quit(ctx)
					return
				}

				entries, err := os.ReadDir(mountPoint)
				if err != nil {
					exec.Command("hdiutil", "detach", mountPoint, "-force").Run()
					exec.Command("open", savePath).Start()
					wailsruntime.Quit(ctx)
					return
				}

				var newAppPath string
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".app") {
						newAppPath = filepath.Join(mountPoint, entry.Name())
						break
					}
				}

				if newAppPath == "" {
					exec.Command("hdiutil", "detach", mountPoint, "-force").Run()
					exec.Command("open", savePath).Start()
					wailsruntime.Quit(ctx)
					return
				}

				appDir := filepath.Dir(appBundlePath)
				scriptPath := filepath.Join(os.TempDir(), strings.ToLower(desktopBrandName())+"_update.sh")
				scriptContent := fmt.Sprintf(`#!/bin/bash
sleep 2
if ! (rm -rf "%s" && cp -a "%s" "%s"); then
    osascript -e "do shell script \"rm -rf \\\"%s\\\" && cp -a \\\"%s\\\" \\\"%s\\\"\" with administrator privileges"
fi
hdiutil detach "%s" -force
open "%s"
rm "$0"
`, appBundlePath, newAppPath, appDir, appBundlePath, newAppPath, appDir, mountPoint, appBundlePath)

				os.WriteFile(scriptPath, []byte(scriptContent), 0755)

				exec.Command("bash", scriptPath).Start()
				wailsruntime.Quit(ctx)
			}()
		} else {
			exec.Command("open", savePath).Start()
			wailsruntime.Quit(ctx)
		}
	} else {
		exec.Command("xdg-open", savePath).Start()
		wailsruntime.Quit(ctx)
	}
}
