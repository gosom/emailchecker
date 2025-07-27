package emailpattern

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"unicode"

	"emailchecker"
)

const (
	defaultMinLocalPartLength    = 3
	defaultMaxConsecutiveNumbers = 5
	defaultMaxSpecialCharRatio   = 0.3
	highEntropyThreshold         = 3.5
	minKeyboardSeqLength         = 4
)

var (
	humanPattern = regexp.MustCompile(`^[\p{L}][\p{L}\p{N}\p{M}._+-]*$|^[\p{L}]\p{N}+$|^[\p{L}]$`)
	keyboardRows = []string{
		"qwertyuiop", "asdfghjkl", "zxcvbnm", // English
		"qwertzuiop", "asdfghjkl", "yxcvbnm", // German
		"1234567890", // Numbers
	}
)

type Config struct {
	MinLocalPartLength    int
	MaxConsecutiveNumbers int
	MaxSpecialCharRatio   float64
}

func DefaultConfig() *Config {
	return &Config{
		MinLocalPartLength:    defaultMinLocalPartLength,
		MaxConsecutiveNumbers: defaultMaxConsecutiveNumbers,
		MaxSpecialCharRatio:   defaultMaxSpecialCharRatio,
	}
}

type EmailPatternChecker struct {
	config *Config
}

func New() *EmailPatternChecker {
	return NewWithConfig(DefaultConfig())
}

func NewWithConfig(cfg *Config) *EmailPatternChecker {
	return &EmailPatternChecker{config: cfg}
}

func (c *EmailPatternChecker) Check(_ context.Context, email string) (*emailchecker.EmailPatternCheckResult, error) {
	atCount := strings.Count(email, "@")
	if atCount != 1 {
		return nil, errors.New("invalid email format")
	}

	at := strings.Index(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return nil, errors.New("invalid email format")
	}

	local := email[:at]
	runes := []rune(local)

	res := &emailchecker.EmailPatternCheckResult{
		ShortLocalPart:            len(runes) < c.config.MinLocalPartLength,
		TooManyConsecutiveNumbers: c.hasConsecutiveNumbers(local),
	}

	specialChars := 0
	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) {
			specialChars++
		}
	}

	if len(runes) > 0 && float64(specialChars)/float64(len(runes)) > c.config.MaxSpecialCharRatio {
		res.TooManySpecialChars = true
	}

	isHighEntropy := len(runes) >= 6 && calcEntropy(runes) > 3.5
	hasKeyboard := hasKeyboardSeq(local)
	manyCases := manyCaseSwitch(local)
	notHumanPattern := !humanPattern.MatchString(local)
	notHumanName := !looksLikeHumanName(local)

	if hasKeyboard ||
		(isHighEntropy && len(runes) >= 12 && notHumanName) ||
		(manyCases && len(runes) >= 6) ||
		(notHumanPattern && !isValidUnicodeScript(local)) ||
		res.TooManyConsecutiveNumbers ||
		res.TooManySpecialChars {
		res.HasRandomPattern = true
	}

	if hasKeyboard ||
		res.TooManyConsecutiveNumbers ||
		res.TooManySpecialChars ||
		(manyCases && len(runes) >= 6) ||
		(notHumanPattern && !isValidUnicodeScript(local)) ||
		(notHumanName && len(runes) >= 8) ||
		(notHumanName && isHighEntropy && len(runes) >= 6) {
		res.HasRandomPattern = true
	}

	return res, nil
}

func looksLikeHumanName(s string) bool {
	s = strings.ToLower(s)

	parts := strings.FieldsFunc(s, func(c rune) bool {
		return c == '.' || c == '_' || c == '-' || c == '+'
	})

	var allParts []string
	for _, part := range parts {
		subParts := splitLettersAndTrailingNumbers(part)
		allParts = append(allParts, subParts...)
	}

	for _, part := range allParts {
		if !isValidNamePart(part) {
			return false
		}
	}

	return true
}

func splitLettersAndTrailingNumbers(s string) []string {
	if len(s) == 0 {
		return []string{}
	}

	lastLetterPos := -1
	for i, r := range s {
		if unicode.IsLetter(r) || unicode.IsMark(r) {
			lastLetterPos = i
		}
	}

	if lastLetterPos == -1 {
		return []string{s}
	}

	runes := []rune(s)
	allDigitsAfter := true
	digitStart := lastLetterPos + 1

	if digitStart < len(runes) {
		for i := digitStart; i < len(runes); i++ {
			if !unicode.IsDigit(runes[i]) {
				allDigitsAfter = false
				break
			}
		}
	}

	if allDigitsAfter && digitStart < len(runes) && digitStart > 0 {
		letterPart := string(runes[:digitStart])
		digitPart := string(runes[digitStart:])
		return []string{letterPart, digitPart}
	}

	return []string{s}
}

func isValidNamePart(part string) bool {
	if len(part) == 0 {
		return false
	}

	if isLikelyYear(part) {
		return true
	}

	if isAllLetters(part) {
		return true
	}

	if hasNameWithTrailingNumbers(part) {
		return true
	}

	letterDigitSwitches := countLetterDigitSwitches(part)
	if letterDigitSwitches > 2 {
		return false // Too many switches = likely random
	}

	if len(part) > 6 {
		digitCount := 0
		for _, r := range part {
			if unicode.IsDigit(r) {
				digitCount++
			}
		}
		digitRatio := float64(digitCount) / float64(len([]rune(part)))
		if digitRatio > 0.4 { // More lenient threshold
			return false
		}
	}

	return true
}

func isLikelyYear(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s >= "1900" && s <= "2030"
}

func isAllLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsMark(r) {
			return false
		}
	}
	return len(s) > 0
}

func hasNameWithTrailingNumbers(s string) bool {
	// Pattern: letters followed by numbers (john123, mary2000)
	letterCount := 0
	digitCount := 0
	inDigitSection := false

	for _, r := range s {
		if unicode.IsDigit(r) {
			if !inDigitSection && letterCount == 0 {
				return false // Starts with numbers
			}
			inDigitSection = true
			digitCount++
		} else if unicode.IsLetter(r) || unicode.IsMark(r) {
			if inDigitSection {
				return false // Letters after digits
			}
			letterCount++
		} else {
			return false // Special characters
		}
	}

	return letterCount >= 2 && digitCount >= 1 && digitCount <= 4
}

func countLetterDigitSwitches(s string) int {
	switches := 0
	lastWasLetter := false
	lastWasDigit := false

	for _, r := range s {
		isLetter := unicode.IsLetter(r) || unicode.IsMark(r)
		isDigit := unicode.IsDigit(r)

		if isLetter && lastWasDigit {
			switches++
		} else if isDigit && lastWasLetter {
			switches++
		}

		lastWasLetter = isLetter
		lastWasDigit = isDigit
	}

	return switches
}

func isValidUnicodeScript(s string) bool {
	runes := []rune(s)
	if len(runes) == 0 {
		return false
	}

	letters := 0
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsMark(r) {
			letters++
		}
	}

	return float64(letters)/float64(len(runes)) > 0.6
}

func (c *EmailPatternChecker) hasConsecutiveNumbers(s string) bool {
	count := 0
	maxCount := 0

	for _, r := range s {
		if unicode.IsDigit(r) {
			count++
			if count > maxCount {
				maxCount = count
			}
		} else {
			count = 0
		}
	}

	return maxCount > c.config.MaxConsecutiveNumbers
}

func calcEntropy(runes []rune) float64 {
	if len(runes) <= 3 {
		return 0
	}

	freq := make(map[rune]int)
	for _, r := range runes {
		freq[unicode.ToLower(r)]++
	}

	ent := 0.0
	n := float64(len(runes))
	for _, count := range freq {
		p := float64(count) / n
		ent -= p * math.Log2(p)
	}

	return ent
}

func hasKeyboardSeq(s string) bool {
	if len(s) < minKeyboardSeqLength {
		return false
	}

	s = strings.ToLower(s)

	for _, row := range keyboardRows {
		if containsSequence(row, s, minKeyboardSeqLength) ||
			containsSequence(reverse(row), s, minKeyboardSeqLength) {
			return true
		}
	}
	return false
}

func containsSequence(row, s string, minLen int) bool {
	for i := 0; i <= len(s)-minLen; i++ {
		substr := s[i : i+minLen]
		if strings.Contains(row, substr) {
			return true
		}
	}
	return false
}

func reverse(s string) string {
	r := []rune(s)
	for i, n := 0, len(r); i < n/2; i++ {
		r[i], r[n-1-i] = r[n-1-i], r[i]
	}
	return string(r)
}

func manyCaseSwitch(s string) bool {
	if len(s) < 6 {
		return false
	}

	switches := 0
	lastCase := -1 // -1: unknown, 0: lower, 1: upper

	for _, r := range s {
		currentCase := -1
		if unicode.IsUpper(r) {
			currentCase = 1
		} else if unicode.IsLower(r) {
			currentCase = 0
		}

		if lastCase != -1 && currentCase != -1 && currentCase != lastCase {
			switches++
		}

		if currentCase != -1 {
			lastCase = currentCase
		}
	}

	return switches > len(s)/3
}
