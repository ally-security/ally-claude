package source

import (
	"fmt"
	"os"
	"strings"
)

// Resolve tries to load the policy bytes from a local path first; if that
// fails it interprets the arg as <user>/<repo>/<path> and fetches from
// raw.githubusercontent.com. Returns the bytes and a human-readable origin
// string describing where the data came from.
func Resolve(arg, branch string) ([]byte, string, error) {
	if data, err := os.ReadFile(arg); err == nil {
		return data, "local:" + arg, nil
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(arg, "~/") {
		if data, err := os.ReadFile(home + arg[1:]); err == nil {
			return data, "local:" + home + arg[1:], nil
		}
	}

	parts := strings.SplitN(arg, "/", 3)
	if len(parts) < 3 {
		return nil, "", fmt.Errorf("not a local file and not in user/repo/path form: %q", arg)
	}
	user, repo, path := parts[0], parts[1], parts[2]
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", user, repo, branch, path)
	data, err := fetchHTTP(url)
	if err != nil {
		return nil, "", fmt.Errorf("local read failed and github fetch failed: %w", err)
	}
	return data, "github:" + url, nil
}
