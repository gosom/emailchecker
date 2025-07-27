package analyzer

import (
	"context"
	"math"
	"strings"

	"emailchecker"
)

const (
	ReasonDisposableBlocked                  = "Disposable email provider blocked"
	ReasonDomainCannotReceiveEmail           = "Domain cannot receive email"
	ReasonSuspiciousEmailPatternDetected     = "Suspicious email pattern detected"
	ReasonShortLocalPart                     = "Email has unusually short local part"
	ReasonTooManyConsecutiveNumbers          = "Email has too many consecutive numbers"
	ReasonEmailHasExcessiveSpecialChars      = "Email has excessive special characters"
	ReasonMultipleSuspiciousPatternsDetected = "Multiple suspicious patterns detected - likely automated"
	ReasonRandomPatternOnWellKnownDomain     = "Random pattern on well-known domain - likely bot generated"
	ReasonRandomPatternOnUnknownDomain       = "Random pattern on unknown domain - likely bot generated"
	ReasonWellKnownEmailProvider             = "Well-known email provider"
	ReasonUnknownEmailProvider               = "Unknown email provider"
	ReasonOnlyOneMXRecord                    = "Domain has only one MX record"
	ReasonLackSPFRecord                      = "Domain lacks SPF record"
	ReasonStrictSPFPolicy                    = "Domain has strict SPF policy"
	ReasonDomainLacksDMARC                   = "Domain lacks DMARC record"
	ReasonHasStrongDMARCPolicy               = "Domain has strong DMARC policy"
	ReasonNoSupiciousSignalsDetected         = "No suspicious signals detected"
	ReasonEducationalInstitutionDomain       = "Email from educational institution domain"
	ReasonStudentIDStaffIDPatternDetected    = "Student/Staff ID pattern detected"
)

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Analyze(ctx context.Context, result *emailchecker.EmailCheckResult) *emailchecker.AnalysisReport {
	report := &emailchecker.AnalysisReport{
		Score:   0.0,
		Reasons: []string{},
	}

	isEducational := result.Educational.Checked && result.Educational.Value

	if result.Disposable.Checked && result.Disposable.Value {
		report.Score = 1.0
		report.RiskLevel = emailchecker.RiskLevelHigh
		report.Reasons = append(report.Reasons, ReasonDisposableBlocked)
		return report
	}

	if result.DNS.Checked && !result.DNS.Value.HasMX {
		report.Score = 1.0
		report.RiskLevel = emailchecker.RiskLevelHigh
		report.Reasons = append(report.Reasons, ReasonDomainCannotReceiveEmail)
		return report
	}

	suspicionLevel := 0
	hasRandomPattern := false

	if result.Pattern.Checked {
		pattern := result.Pattern.Value

		if pattern.HasRandomPattern {
			if isEducational {
				report.Reasons = append(report.Reasons, ReasonEducationalInstitutionDomain)
			} else {
				hasRandomPattern = true
				suspicionLevel++
				report.Reasons = append(report.Reasons, ReasonSuspiciousEmailPatternDetected)
			}
		}

		if pattern.ShortLocalPart {
			if !isEducational {
				suspicionLevel++
				report.Reasons = append(report.Reasons, ReasonShortLocalPart)
			}
		}

		if pattern.TooManyConsecutiveNumbers {
			if !isEducational {
				suspicionLevel++
				report.Reasons = append(report.Reasons, ReasonTooManyConsecutiveNumbers)
			} else {
				report.Reasons = append(report.Reasons, ReasonStudentIDStaffIDPatternDetected)
			}
		}

		if pattern.TooManySpecialChars {
			suspicionLevel++
			report.Reasons = append(report.Reasons, ReasonEmailHasExcessiveSpecialChars)
		}

		blockThreshold := 3
		if isEducational {
			blockThreshold = 4
		}

		if suspicionLevel >= blockThreshold {
			report.Score = 0.9
			report.RiskLevel = emailchecker.RiskLevelHigh
			report.Reasons = append(report.Reasons, ReasonMultipleSuspiciousPatternsDetected)
			return report
		}

		if hasRandomPattern && !isEducational {
			report.Score = 0.8
			report.RiskLevel = emailchecker.RiskLevelHigh

			if result.WellKnown.Checked && result.WellKnown.Value {
				report.Reasons = append(report.Reasons, ReasonRandomPatternOnWellKnownDomain)
			} else {
				report.Reasons = append(report.Reasons, ReasonRandomPatternOnUnknownDomain)
			}
			return report
		}
	}

	patternScore := 0.0

	if result.Pattern.Checked {
		pattern := result.Pattern.Value

		if pattern.ShortLocalPart && !isEducational {
			patternScore += 0.2
		}

		if pattern.TooManyConsecutiveNumbers && !isEducational {
			patternScore += 0.2
		}

		if pattern.TooManySpecialChars {
			patternScore += 0.15
		}
	}

	domainScore := 0.0
	if result.WellKnown.Checked {
		if result.WellKnown.Value {
			domainScore -= 0.15
			report.Reasons = append(report.Reasons, ReasonWellKnownEmailProvider)
		} else {
			domainScore += 0.25
			report.Reasons = append(report.Reasons, ReasonUnknownEmailProvider)
		}
	}

	if isEducational {
		domainScore -= 0.2
		report.Reasons = append(report.Reasons, ReasonEducationalInstitutionDomain)
	}

	dnsScore := 0.0
	if result.DNS.Checked {
		dns := result.DNS.Value

		if len(dns.MXRecords) == 1 {
			dnsScore += 0.1
			report.Reasons = append(report.Reasons, ReasonOnlyOneMXRecord)
		}

		if !dns.HasSPF {
			dnsScore += 0.1
			report.Reasons = append(report.Reasons, ReasonLackSPFRecord)
		} else if strings.Contains(dns.SPFRecord, "-all") {
			dnsScore -= 0.05
			report.Reasons = append(report.Reasons, ReasonStrictSPFPolicy)
		}

		if !dns.HasDMARC {
			dnsScore += 0.1
			report.Reasons = append(report.Reasons, ReasonDomainLacksDMARC)
		} else if strings.Contains(dns.DMARCRecord, "p=reject") || strings.Contains(dns.DMARCRecord, "p=quarantine") {
			dnsScore -= 0.1
			report.Reasons = append(report.Reasons, ReasonHasStrongDMARCPolicy)
		}
	}

	report.Score = patternScore + domainScore + dnsScore
	report.Score = math.Max(0, math.Min(1, report.Score))

	switch {
	case report.Score >= 0.7:
		report.RiskLevel = emailchecker.RiskLevelHigh
	case report.Score >= 0.4:
		report.RiskLevel = emailchecker.RiskLevelMedium
	default:
		report.RiskLevel = emailchecker.RiskLevelLow
	}

	if len(report.Reasons) == 0 {
		report.Reasons = append(report.Reasons, ReasonNoSupiciousSignalsDetected)
	}

	return report
}
