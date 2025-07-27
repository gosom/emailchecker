package disposable

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GithubFetcher struct {
	client *http.Client
}

func NewGithubFetcher(client *http.Client) *GithubFetcher {
	return &GithubFetcher{
		client: client,
	}
}

func (g *GithubFetcher) FetchDisposableDomains(ctx context.Context) ([]string, error) {
	const disposableDomainsURL = "https://raw.githubusercontent.com/disposable/disposable-email-domains/master/domains.txt"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, disposableDomainsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch disposable domains: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var domains []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains found in response")
	}

	return domains, nil
}
