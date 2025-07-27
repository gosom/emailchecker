package emailchecker

import "context"

type DisposableChecker interface {
	IsDisposable(ctx context.Context, domain string) (bool, error)
	UpdateDisposableList(ctx context.Context) error
}

type DNSChecker interface {
	GetDNSValidationResult(ctx context.Context, domain string) (*DNSValidationResult, error)
}

type WellKnownChecker interface {
	IsWellKnown(ctx context.Context, domain string) (bool, error)
	UpdateWellKnownList(ctx context.Context) error
}

type EducationalDomainChecker interface {
	IsEducationalDomain(ctx context.Context, domain string) (bool, error)
	UpdateEducationalDomains(ctx context.Context) error
}

type EmailPatternChecker interface {
	Check(ctx context.Context, email string) (*EmailPatternCheckResult, error)
}

type Analyzer interface {
	Analyze(ctx context.Context, result *EmailCheckResult) *AnalysisReport
}
