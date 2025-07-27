package emailchecker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type EmailChecker struct {
	disposableSvc   DisposableChecker
	dnsSvc          DNSChecker
	wellKnownSvc    WellKnownChecker
	educationalSvc  EducationalDomainChecker
	emailPatternSvc EmailPatternChecker
	analysisSvc     Analyzer
}

func New(cfg *Config) (*EmailChecker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ans := EmailChecker{
		disposableSvc:   cfg.DisposableService,
		dnsSvc:          cfg.DNSService,
		wellKnownSvc:    cfg.WellKnownService,
		educationalSvc:  cfg.EducationalDomainService,
		emailPatternSvc: cfg.EmailPatternService,
		analysisSvc:     cfg.AnalysisService,
	}

	return &ans, nil
}

func (e *EmailChecker) Close() error {
	return nil
}

func (e *EmailChecker) Check(ctx context.Context, params EmailCheckParams) (EmailCheckResult, error) {
	start := time.Now()
	if params.DisposableTimeout == 0 {
		params.DisposableTimeout = 200 * time.Millisecond
	}

	result := EmailCheckResult{
		Email: params.Email,
	}
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	email := params.Email

	idx := strings.LastIndex(email, "@")
	if idx < 0 || idx >= len(email)-1 {
		return EmailCheckResult{}, fmt.Errorf("invalid email address: %s", email)
	}

	domain := email[idx+1:]

	e.performDNSCheck(ctx, params, &wg, &result, &mu, domain)
	e.performDisposableCheck(ctx, params, &wg, &result, &mu, domain)
	e.performWellKnownCheck(ctx, params, &wg, &result, &mu, domain)
	e.performEducationalCheck(ctx, params, &wg, &result, &mu, domain)
	e.performEmailPatternCheck(ctx, params, &wg, &result, &mu, email)

	wg.Wait()
	result.Elapsed = time.Since(start)

	result.Analysis = e.analysisSvc.Analyze(ctx, &result)

	return result, nil
}

func (e *EmailChecker) UpdateDisposableDB(ctx context.Context) error {
	return nil
}

func (e *EmailChecker) performDisposableCheck(ctx context.Context, params EmailCheckParams, wg *sync.WaitGroup, result *EmailCheckResult, mu *sync.Mutex, domain string) {
	if params.SkipDisposable {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		start := time.Now()
		disposableCtx, disposableCancel := context.WithTimeout(ctx, params.DisposableTimeout)
		defer disposableCancel()

		isDisposable, err := e.disposableSvc.IsDisposable(disposableCtx, domain)

		elapsed := time.Since(start)

		mu.Lock()
		defer mu.Unlock()
		result.Disposable.Checked = true
		result.Disposable.Elapsed = elapsed

		if err != nil {
			result.Disposable.Err = err
		} else {
			result.Disposable.Value = isDisposable
		}
	}()
}

func (e *EmailChecker) performDNSCheck(ctx context.Context, params EmailCheckParams, wg *sync.WaitGroup, result *EmailCheckResult, mu *sync.Mutex, domain string) {
	if params.SkipDNS {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		start := time.Now()
		raw, err := e.dnsSvc.GetDNSValidationResult(ctx, domain)
		if err == nil && raw != nil {
			for i := range raw.MXRecords {
				isDisposable, err := e.disposableSvc.IsDisposable(ctx, raw.MXRecords[i].Value)
				if err == nil {
					raw.MXRecords[i].Disposable = isDisposable
				}
			}
		}

		elapsed := time.Since(start)

		mu.Lock()
		defer mu.Unlock()

		result.DNS.Checked = true
		if err != nil {
			result.DNS.Err = err
		} else {
			result.DNS.Value = *raw
		}

		result.DNS.Elapsed = elapsed
	}()
}

func (e *EmailChecker) performWellKnownCheck(ctx context.Context, params EmailCheckParams, wg *sync.WaitGroup, result *EmailCheckResult, mu *sync.Mutex, domain string) {
	if params.SkipWellKnown {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		start := time.Now()
		isWellKnown, err := e.wellKnownSvc.IsWellKnown(ctx, domain)

		elapsed := time.Since(start)

		mu.Lock()
		defer mu.Unlock()

		result.WellKnown.Checked = true
		result.WellKnown.Elapsed = elapsed

		if err != nil {
			result.WellKnown.Err = err
		} else {
			result.WellKnown.Value = isWellKnown
		}
	}()
}

func (e *EmailChecker) performEmailPatternCheck(ctx context.Context, params EmailCheckParams, wg *sync.WaitGroup, result *EmailCheckResult, mu *sync.Mutex, email string) {
	if params.SkipPatternCheck {
		return
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		start := time.Now()
		patternResult, err := e.emailPatternSvc.Check(ctx, email)

		elapsed := time.Since(start)

		mu.Lock()
		defer mu.Unlock()

		result.Pattern.Checked = true
		result.Pattern.Elapsed = elapsed

		if err != nil {
			result.Pattern.Err = err
		} else {
			result.Pattern.Value = *patternResult
		}
	}()
}

func (e *EmailChecker) performEducationalCheck(ctx context.Context, params EmailCheckParams, wg *sync.WaitGroup, result *EmailCheckResult, mu *sync.Mutex, domain string) {
	if params.SkipEducationalDomains {
		return
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		start := time.Now()
		isEducational, err := e.educationalSvc.IsEducationalDomain(ctx, domain)

		elapsed := time.Since(start)

		mu.Lock()
		defer mu.Unlock()

		result.Educational.Checked = true
		result.Educational.Elapsed = elapsed

		if err != nil {
			result.Educational.Err = err
		} else {
			result.Educational.Value = isEducational
		}
	}()
}
