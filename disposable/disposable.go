package disposable

import (
	"context"
	"time"
)

type repo interface {
	IsDisposable(context.Context, string) (bool, error)
	UpdateDomains(context.Context, []string) error
	NeedsRefresh(context.Context) (bool, error)
}

type fetcher interface {
	FetchDisposableDomains(ctx context.Context) ([]string, error)
}

type DisposableChecker struct {
	repo    repo
	fetcher fetcher
}

func New(repo repo, fetcher fetcher) (*DisposableChecker, error) {
	ans := DisposableChecker{
		repo:    repo,
		fetcher: fetcher,
	}

	refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := ans.UpdateDisposableList(refreshCtx); err != nil {
		return nil, err
	}

	return &ans, nil
}

func (d *DisposableChecker) IsDisposable(ctx context.Context, domain string) (bool, error) {
	return d.repo.IsDisposable(ctx, domain)
}

func (d *DisposableChecker) UpdateDisposableList(ctx context.Context) error {
	needRefresh, err := d.repo.NeedsRefresh(ctx)
	if err != nil {
		return err
	}

	if !needRefresh {
		return nil
	}

	domains, err := d.fetcher.FetchDisposableDomains(ctx)
	if err != nil {
		return err
	}

	return d.repo.UpdateDomains(ctx, domains)
}
