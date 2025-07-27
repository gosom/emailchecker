package edu

import "context"

type repo interface {
	IsEducationalDomain(ctx context.Context, domain string) (bool, error)
	UpdateEducationalDomains(ctx context.Context, domains []string) error
	NeedsEduRefresh(ctx context.Context) (bool, error)
}

type fetcher interface {
	FetchEducationalDomains(ctx context.Context) ([]string, error)
}

type EducationalDomainChecker struct {
	repo    repo
	fetcher fetcher
}

func New(repo repo, fetcher fetcher) (*EducationalDomainChecker, error) {
	ans := EducationalDomainChecker{
		repo:    repo,
		fetcher: fetcher,
	}

	if err := ans.UpdateEducationalDomains(context.Background()); err != nil {
		return nil, err
	}

	return &ans, nil
}

func (e *EducationalDomainChecker) IsEducationalDomain(ctx context.Context, domain string) (bool, error) {
	return e.repo.IsEducationalDomain(ctx, domain)
}

func (e *EducationalDomainChecker) UpdateEducationalDomains(ctx context.Context) error {
	needsRefresh, err := e.repo.NeedsEduRefresh(ctx)
	if err != nil {
		return err
	}

	if !needsRefresh {
		return nil
	}

	domains, err := e.fetcher.FetchEducationalDomains(ctx)
	if err != nil {
		return err
	}

	return e.repo.UpdateEducationalDomains(ctx, domains)
}
