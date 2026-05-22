// Package selfupdate replaces the running binary with the latest GitHub
// release matching the host OS and architecture.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
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

// Release is the subset of the GitHub releases API we use.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// Options configures an update run.
type Options struct {
	Repo       string // <owner>/<name>
	Current    string // current version (the "v" prefix is normalised)
	GOOS       string // defaults to runtime.GOOS
	GOARCH     string // defaults to runtime.GOARCH
	HTTPClient *http.Client
}

// Result describes what an update run did or would do.
type Result struct {
	Current  string
	Latest   string
	Asset    string
	UpToDate bool
	Replaced bool
	Path     string // final path of the swapped binary
}

// FetchLatest queries the GitHub releases API for the latest release on Repo.
func FetchLatest(opts Options) (*Release, error) {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", opts.Repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// PickAsset returns the archive matching the given goos/goarch, scanning
// the asset names produced by GoReleaser's default template.
func PickAsset(rel *Release, goos, goarch string) (*Asset, error) {
	suffix := fmt.Sprintf("_%s_%s", goos, goarch)
	for i := range rel.Assets {
		a := &rel.Assets[i]
		name := strings.ToLower(a.Name)
		if !strings.Contains(name, strings.ToLower(suffix)) {
			continue
		}
		if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no asset for %s/%s in release %s", goos, goarch, rel.TagName)
}

// FetchChecksum downloads the checksums.txt asset and returns the expected
// sha256 for assetName, or "" if no checksums asset exists.
func FetchChecksum(rel *Release, assetName string, client *http.Client) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	for _, a := range rel.Assets {
		if a.Name != "checksums.txt" {
			continue
		}
		resp, err := client.Get(a.DownloadURL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GET checksums.txt: %s", resp.Status)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return parseChecksum(string(data), assetName), nil
	}
	return "", nil
}

func parseChecksum(body, assetName string) string {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[1] == assetName {
			return strings.ToLower(fields[0])
		}
	}
	return ""
}

// Download retrieves the asset bytes via client.
func Download(a *Asset, client *http.Client) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	resp, err := client.Get(a.DownloadURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", a.DownloadURL, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// ExtractBinary pulls the claude-3p-helper executable out of an archive.
func ExtractBinary(data []byte, assetName string) ([]byte, error) {
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractTarGz(data, "claude-3p-helper")
	case strings.HasSuffix(assetName, ".zip"):
		return extractZip(data, "claude-3p-helper.exe")
	default:
		return nil, fmt.Errorf("unsupported archive: %s", assetName)
	}
}

func extractTarGz(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) == name {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in tar.gz", name)
}

func extractZip(data []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in zip", name)
}

// VerifySHA256 compares the sha256 of data against want (lowercase hex).
func VerifySHA256(data []byte, want string) error {
	if want == "" {
		return nil
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, want)
	}
	return nil
}

// ReplaceBinary writes newBinary to a temp file next to target then renames
// over target. Returns the final path. On Windows the running binary
// can't be deleted while executing, so the previous binary is moved
// aside to target+".old" first.
func ReplaceBinary(target string, newBinary []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".claude-3p-helper-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		old := target + ".old"
		_ = os.Remove(old)
		if _, err := os.Stat(target); err == nil {
			if err := os.Rename(target, old); err != nil {
				return err
			}
		}
	}
	return os.Rename(tmpPath, target)
}

// NormaliseVersion strips a leading "v" so "v1.2.3" and "1.2.3" compare equal.
func NormaliseVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// Run performs the full update flow: fetch, compare, download, verify, swap.
// If dryRun is true, no files are written and the result reports what would
// have happened.
func Run(opts Options, targetBinary string, dryRun bool) (*Result, error) {
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}
	rel, err := FetchLatest(opts)
	if err != nil {
		return nil, err
	}
	result := &Result{
		Current: NormaliseVersion(opts.Current),
		Latest:  NormaliseVersion(rel.TagName),
		Path:    targetBinary,
	}
	if result.Current == result.Latest && result.Current != "dev" {
		result.UpToDate = true
		return result, nil
	}
	asset, err := PickAsset(rel, opts.GOOS, opts.GOARCH)
	if err != nil {
		return result, err
	}
	result.Asset = asset.Name

	if dryRun {
		return result, nil
	}

	want, err := FetchChecksum(rel, asset.Name, opts.HTTPClient)
	if err != nil {
		return result, err
	}
	archive, err := Download(asset, opts.HTTPClient)
	if err != nil {
		return result, err
	}
	if err := VerifySHA256(archive, want); err != nil {
		return result, err
	}
	bin, err := ExtractBinary(archive, asset.Name)
	if err != nil {
		return result, err
	}
	if err := ReplaceBinary(targetBinary, bin); err != nil {
		return result, err
	}
	result.Replaced = true
	return result, nil
}
