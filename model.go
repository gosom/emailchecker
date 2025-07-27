package emailchecker

import "time"

type EmailPatternCheckResult struct {
	ShortLocalPart            bool
	HasRandomPattern          bool
	TooManyConsecutiveNumbers bool
	TooManySpecialChars       bool
}

type DNSValidationResult struct {
	Domain      string
	HasMX       bool
	HasSPF      bool
	HasDMARC    bool
	IsParked    bool
	ARecords    []string
	NSRecords   []string
	MXRecords   []MXRecord
	SPFRecord   string
	DMARCRecord string
}

type MXRecord struct {
	Value      string
	Priority   int
	Disposable bool
}

type DNSRecord struct {
	Domain    string
	Data      []byte
	CreatedAt time.Time
}

type EmailCheckResult struct {
	Email       string
	Disposable  SubCheckResult[bool]
	WellKnown   SubCheckResult[bool]
	Educational SubCheckResult[bool]
	DNS         SubCheckResult[DNSValidationResult]
	Elapsed     time.Duration
	Pattern     SubCheckResult[EmailPatternCheckResult]
	Analysis    *AnalysisReport
}

type SubCheckResult[T any] struct {
	Checked bool
	Value   T
	Err     error
	Elapsed time.Duration
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
