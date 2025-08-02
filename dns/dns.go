package dns

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"emailchecker"
)

type repo interface {
	GetDNSRecord(ctx context.Context, domain string) (*emailchecker.DNSRecord, error)
	UpsertDNSRecord(ctx context.Context, domain string, data []byte) error
}

type inFlightRequest struct {
	wg  sync.WaitGroup
	res *emailchecker.DNSValidationResult
	err error
}

type Resolver struct {
	dnsClient *Client
	inflight  map[string]*inFlightRequest
	mu        sync.Mutex
	repo      repo
	sem       chan struct{}
}

func NewResolver(dnsClient *Client, repo repo) *Resolver {
	return &Resolver{
		dnsClient: dnsClient,
		inflight:  make(map[string]*inFlightRequest),
		repo:      repo,
		sem:       make(chan struct{}, 100),
	}
}

func (r *Resolver) GetDNSValidationResult(ctx context.Context, domain string) (*emailchecker.DNSValidationResult, error) {
	cachedRec, _ := r.repo.GetDNSRecord(ctx, domain)

	if cachedRec != nil && time.Since(cachedRec.CreatedAt) < 24*time.Hour {
		var result emailchecker.DNSValidationResult
		if err := json.Unmarshal(cachedRec.Data, &result); err == nil {
			return &result, nil
		}
	}

	r.mu.Lock()
	if req, ok := r.inflight[domain]; ok {
		r.mu.Unlock()
		req.wg.Wait()
		return req.res, req.err
	}

	r.sem <- struct{}{}
	defer func() { <-r.sem }()

	req := &inFlightRequest{}
	req.wg.Add(1)
	r.inflight[domain] = req
	r.mu.Unlock()

	freshResult, fetchErr := r.dnsClient.GetDNSValidation(ctx, domain)

	req.res = freshResult
	req.err = fetchErr
	req.wg.Done()

	r.mu.Lock()
	delete(r.inflight, domain)
	r.mu.Unlock()

	if fetchErr == nil {
		jsonData, marshalErr := json.Marshal(freshResult)
		if marshalErr == nil {
			_ = r.repo.UpsertDNSRecord(ctx, domain, jsonData)
		}
	}

	return freshResult, fetchErr
}
