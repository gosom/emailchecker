package emailchecker

import "time"

type EmailPatternCheckResult struct {
	ShortLocalPart            bool `json:"short_local_part"`
	HasRandomPattern          bool `json:"has_random_pattern"`
	TooManyConsecutiveNumbers bool `json:"too_many_consecutive_numbers"`
	TooManySpecialChars       bool `json:"too_many_special_chars"`
}

type DNSValidationResult struct {
	Domain      string     `json:"domain"`
	HasMX       bool       `json:"has_mx"`
	HasSPF      bool       `json:"has_spf"`
	HasDMARC    bool       `json:"has_dmarc"`
	IsParked    bool       `json:"is_parked"`
	ARecords    []string   `json:"a_records"`
	NSRecords   []string   `json:"ns_records"`
	MXRecords   []MXRecord `json:"mx_records"`
	SPFRecord   string     `json:"spf_record"`
	DMARCRecord string     `json:"dmarc_record"`
}

type MXRecord struct {
	Value      string `json:"value"`
	Priority   int    `json:"priority"`
	Disposable bool   `json:"disposable"`
}

type DNSRecord struct {
	Domain    string
	Data      []byte
	CreatedAt time.Time
}

type EmailCheckResult struct {
	Email       string                                  `json:"email"`
	Disposable  SubCheckResult[bool]                    `json:"disposable"`
	WellKnown   SubCheckResult[bool]                    `json:"well_known"`
	Educational SubCheckResult[bool]                    `json:"educational"`
	DNS         SubCheckResult[DNSValidationResult]     `json:"dns"`
	Elapsed     time.Duration                           `json:"elapsed"`
	Pattern     SubCheckResult[EmailPatternCheckResult] `json:"pattern"`
	Analysis    *AnalysisReport                         `json:"prediction"`
}

type SubCheckResult[T any] struct {
	Checked bool          `json:"checked"`
	Value   T             `json:"value"`
	Err     error         `json:"error"`
	Elapsed time.Duration `json:"elapsed"`
}

type EmailCheckParams struct {
	Email string
	// SkipDisposable indicates whether to skip the disposable email check.
	SkipDisposable bool
	// DisposableTimeout is the timeout for checking disposable emails.
	// If not set, a default value of 200ms will be used.
	DisposableTimeout time.Duration
	// If true, the disposable check will be strict.
	// Default is false.
	DisposableStrict bool
	// SkipDNS indicates whether to skip the DNS check.
	SkipDNS bool
	// SkipWellKnown indicates whether to skip the well-known email provider check.
	SkipWellKnown bool
	// SkipPattern indicates whether to skip the email pattern check.
	SkipPatternCheck bool
	// SkipEducationalDomains indicates whether to skip the educational domain check.
	SkipEducationalDomains bool
}

type AnalysisReport struct {
	RiskLevel RiskLevel `json:"risk_level"`
	Score     float64   `json:"score"`
	Reasons   []string  `json:"reasons"`
}

type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)
