package wellknown

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Tranco struct {
	client *http.Client
}

func NewTranco(netClient *http.Client) *Tranco {
	return &Tranco{
		client: netClient,
	}
}

func (t *Tranco) GetTopList(ctx context.Context) ([]string, error) {
	yesteday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	id, err := t.getTrancoListID(ctx, yesteday)
	if err != nil {
		return nil, err
	}

	return t.fetchTrancoList(ctx, id)
}

func (t *Tranco) getTrancoListID(ctx context.Context, date string) (string, error) {
	urlObject := url.URL{
		Scheme: "https",
		Host:   "tranco-list.eu",
		Path:   "daily_list_id",
	}
	query := urlObject.Query()
	query.Set("date", date)
	query.Set("subdomains", strconv.FormatBool(true))
	urlObject.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlObject.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch Tranco list ID: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if bytes.Equal(body, []byte("null")) {
		return "", fmt.Errorf("no Tranco list ID found for date: %s", date)
	}

	if bytes.Equal(body, []byte("500 Internal Server Error")) {
		return "", fmt.Errorf("Tranco server error for date: %s", date)
	}

	return string(body), nil
}

func (t *Tranco) fetchTrancoList(ctx context.Context, listID string) ([]string, error) {
	u := fmt.Sprintf("https://tranco-list.eu/download/%s/1000000", listID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for Tranco list: %w", err)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Tranco list: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from Tranco: %d", resp.StatusCode)
	}

	domainList := make([]string, 0, 1000000)
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		idx := strings.Index(line, ",")
		if idx < 0 || idx >= len(line)-1 {
			continue
		}

		domain := line[idx+1:]

		domainList = append(domainList, domain)
	}

	return domainList, nil
}
