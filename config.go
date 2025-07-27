package emailchecker

import "fmt"

type Config struct {
	DisposableService        DisposableChecker
	DNSService               DNSChecker
	WellKnownService         WellKnownChecker
	EducationalDomainService EducationalDomainChecker
	EmailPatternService      EmailPatternChecker
	AnalysisService          Analyzer
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if c.DisposableService == nil {
		return fmt.Errorf("%w: disposable service is required", ErrInvalidConfig)
	}

	if c.DNSService == nil {
		return fmt.Errorf("%w: DNS service is required", ErrInvalidConfig)
	}

	if c.WellKnownService == nil {
		return fmt.Errorf("%w: well-known service is required", ErrInvalidConfig)
	}

	if c.EducationalDomainService == nil {
		return fmt.Errorf("%w: educational domain service is required", ErrInvalidConfig)
	}

	if c.EmailPatternService == nil {
		return fmt.Errorf("%w: email pattern service is required", ErrInvalidConfig)
	}

	if c.AnalysisService == nil {
		return fmt.Errorf("%w: analysis service is required", ErrInvalidConfig)
	}

	return nil
}
