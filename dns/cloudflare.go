package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"emailchecker"

	"golang.org/x/sync/errgroup"
)

type CloudflareResponse struct {
	Status int      `json:"Status"`
	Answer []Answer `json:"Answer"`
}

type Answer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

type Client struct {
	httpClient *http.Client
}

func New(netClient *http.Client) *Client {
	return &Client{
		httpClient: netClient,
	}
}

func (c *Client) Lookup(ctx context.Context, domain, recordType string) (*CloudflareResponse, error) {
	const cloudflareAPIEndpoint = "https://one.one.one.one/dns-query"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cloudflareAPIEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create DoH request: %w", err)
	}

	q := req.URL.Query()
	q.Add("name", domain)
	q.Add("type", recordType)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/dns-json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DoH request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		defer resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code from Cloudflare: %s", resp.Status)
	}

	var result CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode DoH JSON response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetDNSValidation(ctx context.Context, domain string) (*emailchecker.DNSValidationResult, error) {
	result := &emailchecker.DNSValidationResult{Domain: domain}

	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		reps, err := c.Lookup(ctx, domain, "A")
		if err != nil {
			return err
		}

		if reps.Status == 0 && len(reps.Answer) > 0 {
			mu.Lock()
			defer mu.Unlock()
			for _, ans := range reps.Answer {
				if ans.Type == 1 {
					result.ARecords = append(result.ARecords, ans.Data)
				}
			}
		}

		return nil
	})

	g.Go(func() error {
		resp, err := c.Lookup(ctx, domain, "NS")
		if err != nil {
			return err
		}

		if resp.Status == 0 && len(resp.Answer) > 0 {
			mu.Lock()
			defer mu.Unlock()
			for _, ans := range resp.Answer {
				if ans.Type == 2 {
					result.NSRecords = append(result.NSRecords, ans.Data)
					if isParkedDomainNs(ans.Data) {
						result.IsParked = true
					}
				}
			}
		}

		return nil
	})

	g.Go(func() error {
		resp, err := c.Lookup(ctx, domain, "MX")
		if err != nil {
			return err
		}

		if resp.Status == 0 && len(resp.Answer) > 0 {
			mu.Lock()
			defer mu.Unlock()
			result.HasMX = true
			for _, ans := range resp.Answer {
				parts := strings.Fields(ans.Data)
				if len(parts) == 2 {
					record := emailchecker.MXRecord{
						Value:    parts[1],
						Priority: 0,
					}
					record.Priority, _ = strconv.Atoi(parts[0])
					result.MXRecords = append(result.MXRecords, record)
				}
			}
		}

		return nil
	})

	g.Go(func() error {
		resp, err := c.Lookup(ctx, domain, "TXT")
		if err != nil {
			return err
		}
		if resp.Status == 0 {
			for _, ans := range resp.Answer {
				if strings.HasPrefix(ans.Data, `"v=spf1`) {
					mu.Lock()
					defer mu.Unlock()
					result.HasSPF = true
					result.SPFRecord = strings.Trim(ans.Data, `"`)
					break
				}
			}
		}

		return nil
	})

	g.Go(func() error {
		dmarcDomain := "_dmarc." + domain
		resp, err := c.Lookup(ctx, dmarcDomain, "TXT")
		if err != nil {
			return err
		}
		if resp.Status == 0 {
			for _, ans := range resp.Answer {
				if strings.HasPrefix(ans.Data, `"v=DMARC1`) {
					mu.Lock()
					defer mu.Unlock()
					result.HasDMARC = true
					result.DMARCRecord = strings.Trim(ans.Data, `"`)
					break
				}
			}
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}
