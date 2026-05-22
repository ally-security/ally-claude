package source

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Resolve tries to load the policy bytes from a local path first; if that
// fails it interprets the arg as <user>/<repo>/<path> and fetches from
// GitHub. When GITHUB_TOKEN (or GH_TOKEN) is set, the GitHub Contents
// API is used so private repos work; otherwise the public
// raw.githubusercontent.com URL is used. Returns the bytes and a
// human-readable origin string describing where the data came from.
func Resolve(arg, branch string) ([]byte, string, error) {
	slog.Debug("source: trying local path", "arg", arg)
	if data, err := os.ReadFile(arg); err == nil {
		return data, "local:" + arg, nil
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(arg, "~/") {
		expanded := home + arg[1:]
		slog.Debug("source: trying ~-expanded path", "path", expanded)
		if data, err := os.ReadFile(expanded); err == nil {
			return data, "local:" + expanded, nil
		}
	}

	parts := strings.SplitN(arg, "/", 3)
	if len(parts) < 3 {
		return nil, "", fmt.Errorf("not a local file and not in user/repo/path form: %q", arg)
	}
	user, repo, path := parts[0], parts[1], parts[2]

	if token := githubToken(); token != "" {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", user, repo, path, branch)
		slog.Debug("source: fetching from GitHub Contents API", "url", url)
		data, err := fetchAPI(url, token)
		if err != nil {
			return nil, "", fmt.Errorf("local read failed and github API fetch failed: %w", err)
		}
		return data, "github-api:" + url, nil
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", user, repo, branch, path)
	slog.Debug("source: falling back to raw.githubusercontent.com", "url", url)
	data, err := fetchHTTP(url)
	if err != nil {
		return nil, "", fmt.Errorf("local read failed and github fetch failed: %w", err)
	}
	return data, "github:" + url, nil
}

func githubToken() string {
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GH_TOKEN")
}
