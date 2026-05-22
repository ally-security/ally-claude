package source

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchHTTP issues an unauthenticated GET. Used for the public
// raw.githubusercontent.com path.
func fetchHTTP(url string) ([]byte, error) {
	return doGet(url, nil)
}

// fetchAPI issues an authenticated GET against the GitHub API for
// fetching a file from a (potentially private) repo. Accept header
// tells the contents endpoint to return the raw file bytes instead
// of the default JSON envelope.
func fetchAPI(url, token string) ([]byte, error) {
	return doGet(url, map[string]string{
		"Authorization": "token " + token,
		"Accept":        "application/vnd.github.raw",
	})
}

func doGet(url string, headers map[string]string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
