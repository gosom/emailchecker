package wellknown

import (
	"context"
	"fmt"
)

type repo interface {
	IsTop(context.Context, string) (bool, error)
	TopNeedsRefresh(context.Context) (bool, error)
	UpdateTopDomains(context.Context, []string) error
}

type fetcher interface {
	GetTopList(ctx context.Context) ([]string, error)
}

type WellKnownDomainChecker struct {
	repo    repo
	fetcher fetcher
}

func New(repo repo, fetcher fetcher) (*WellKnownDomainChecker, error) {
	ans := WellKnownDomainChecker{
		repo:    repo,
		fetcher: fetcher,
	}

	if err := ans.UpdateWellKnownList(context.Background()); err != nil {
		return nil, fmt.Errorf("could not update well-known list: %w", err)
	}

	return &ans, nil
}

func (w *WellKnownDomainChecker) IsWellKnown(ctx context.Context, domain string) (bool, error) {
	return w.repo.IsTop(ctx, domain)
}

func (w *WellKnownDomainChecker) UpdateWellKnownList(ctx context.Context) error {
	needsRefresh, err := w.repo.TopNeedsRefresh(ctx)
	if err != nil {
		return fmt.Errorf("could not check if top domains need refresh: %w", err)
	}

	if !needsRefresh {
		return nil
	}

	topList, err := w.fetcher.GetTopList(ctx)
	if err != nil {
		return fmt.Errorf("could not fetch top domains: %w", err)
	}

	if len(topList) == 0 {
		return fmt.Errorf("top list is empty")
	}

	err = w.repo.UpdateTopDomains(ctx, topList)
	if err != nil {
		return fmt.Errorf("could not update top domains: %w", err)
	}

	return nil
}
