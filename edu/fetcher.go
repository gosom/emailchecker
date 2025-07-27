package edu

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type EduFetcher struct {
	client *http.Client
}

func NewEduFetcher(client *http.Client) *EduFetcher {
	return &EduFetcher{
		client: client,
	}
}

func (f *EduFetcher) FetchEducationalDomains(ctx context.Context) ([]string, error) {
	const u = "https://raw.githubusercontent.com/Hipo/university-domains-list/master/world_universities_and_domains.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	type item struct {
		Domains []string `json:"domains"`
	}

	var items []item
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	ans := make([]string, 0, len(items))
	for _, i := range items {
		ans = append(ans, i.Domains...)
	}

	if len(ans) == 0 {
		return nil, nil
	}

	return ans, nil
}
