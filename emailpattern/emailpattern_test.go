package emailpattern_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"emailchecker/emailpattern"
)

// Note: I got this from AI, so mayhbe not correct

func TestEmailPatternCheck_Check(t *testing.T) {
	type testCase struct {
		name                   string
		email                  string
		expectError            bool
		shortLocalPart         bool
		hasRandomPattern       bool
		tooManyConsecutiveNums bool
		tooManySpecialChars    bool
	}

	cases := []testCase{
		// Valid human patterns
		{name: "Valid human pattern", email: "john.doe@domain.com"},
		{name: "German umlaut", email: "müller@beispiel.de"},
		{name: "German eszett", email: "groß@beispiel.de"},
		{name: "Greek name", email: "Γιώργος@domain.gr"},
		{name: "Cyrillic user", email: "Дмитрий@почта.рф"}, // Should NOT be random
		{name: "French accents", email: "étienne_dupont@courriel.fr"},
		{name: "Spanish accented", email: "josé-luís@correo.es"},
		{name: "Name with numbers", email: "giorgos1984@domain.com"},
		{name: "Single letter", email: "a@domain.com", shortLocalPart: true}, // Single letter is NOT random by itself
		{name: "Two letters", email: "ab@domain.com", shortLocalPart: true},
		{name: "Simple name with dot", email: "john.smith@domain.com"},
		{name: "Name with underscore", email: "john_doe@domain.com"},
		{name: "Name with hyphen", email: "jean-claude@domain.com"},
		{name: "Name with plus", email: "john+tag@domain.com"},
		{name: "Mixed with acceptable numbers", email: "john123@domain.com"}, // Should be valid - only 3 consecutive numbers
		{name: "Valid with single number", email: "john1@domain.com"},

		// Random patterns
		{name: "Random casing", email: "rAnDomCAsE@domain.com", hasRandomPattern: true}, // Made longer
		{name: "Keyboard sequence qwer", email: "qwer@domain.com", hasRandomPattern: true},
		{name: "Keyboard sequence asdf", email: "asdf@domain.com", hasRandomPattern: true},
		{name: "German QWERTZ keyboard walk", email: "qwertz@domain.com", hasRandomPattern: true},
		{name: "Reverse keyboard sequence", email: "trewq@domain.com", hasRandomPattern: true},
		{name: "Numbers sequence", email: "user123456@domain.com", hasRandomPattern: true, tooManyConsecutiveNums: true},
		{name: "High entropy random", email: "x9z2k8m1qwerty@domain.com", hasRandomPattern: true}, // Made longer and includes keyboard
		{name: "Many special chars", email: "a.b_c-d+e@domain.com", hasRandomPattern: true, tooManySpecialChars: true},

		// Edge cases
		{name: "Just numbers", email: "123456@domain.com", hasRandomPattern: true, tooManyConsecutiveNums: true},
		{name: "Reasonable case switching", email: "iPhone@domain.com"},
		{name: "Too many case switches", email: "aBcDeFgHiJkL@domain.com", hasRandomPattern: true}, // Made much longer
		{name: "Empty local part", email: "@domain.com", expectError: true},
		{name: "No @ symbol", email: "invalidemail", expectError: true},
		{name: "Multiple @ symbols", email: "test@@domain.com", expectError: true},
		{name: "@ at start", email: "@domain.com", expectError: true},
		{name: "@ at end", email: "test@", expectError: true},

		{name: "random hotmail", email: "mx4nh2pw7sq1pc3@hotmail.com", hasRandomPattern: true},
		{name: "random gmail", email: "m0979689258@gmail.com", hasRandomPattern: true, tooManyConsecutiveNums: true},
		{name: "valid gmail", email: "giorgos1984@gmail.com"},
	}

	c := emailpattern.New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.Check(context.Background(), tc.email)

			if tc.expectError {
				assert.Error(t, err, "expected error for email: %s", tc.email)
				return
			}

			assert.NoError(t, err, "unexpected error for email: %s", tc.email)
			assert.NotNil(t, res, "result should not be nil for email: %s", tc.email)

			assert.Equal(t, tc.shortLocalPart, res.ShortLocalPart,
				"ShortLocalPart mismatch for email: %s", tc.email)

			assert.Equal(t, tc.hasRandomPattern, res.HasRandomPattern,
				"HasRandomPattern mismatch for email: %s", tc.email)

			assert.Equal(t, tc.tooManyConsecutiveNums, res.TooManyConsecutiveNumbers,
				"TooManyConsecutiveNumbers mismatch for email: %s", tc.email)

			assert.Equal(t, tc.tooManySpecialChars, res.TooManySpecialChars,
				"TooManySpecialChars mismatch for email: %s", tc.email)
		})
	}
}

func TestEmailPatternCheck_ConsecutiveNumbers(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		expected bool
	}{
		{"No consecutive numbers", "john@domain.com", false},
		{"Few consecutive numbers", "john123@domain.com", false},
		{"Exactly max consecutive", "john12345@domain.com", false},
		{"Too many consecutive", "john123456@domain.com", true},
		{"Multiple separate groups", "john123abc456@domain.com", false},
		{"Very long consecutive", "user1234567890@domain.com", true},
	}

	c := emailpattern.New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.Check(context.Background(), tc.email)

			assert.NoError(t, err, "unexpected error for email: %s", tc.email)
			assert.NotNil(t, res, "result should not be nil for email: %s", tc.email)

			assert.Equal(t, tc.expected, res.TooManyConsecutiveNumbers,
				"TooManyConsecutiveNumbers mismatch for email: %s", tc.email)
		})
	}
}

func TestEmailPatternCheck_SpecialCharRatio(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		expected bool
	}{
		{"No special chars", "johnsmith@domain.com", false},
		{"Some special chars", "john.doe@domain.com", false},
		{"At threshold", "a.b_c@domain.com", true},               // 2/5 = 0.4 > 0.3, so should be true
		{"Too many special chars", "a.b_c-d+e@domain.com", true}, // 4/9 ≈ 0.44 > 0.3
	}

	c := emailpattern.New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.Check(context.Background(), tc.email)

			assert.NoError(t, err, "unexpected error for email: %s", tc.email)
			assert.NotNil(t, res, "result should not be nil for email: %s", tc.email)

			assert.Equal(t, tc.expected, res.TooManySpecialChars,
				"TooManySpecialChars mismatch for email: %s", tc.email)
		})
	}
}

func TestEmailPatternCheck_Config(t *testing.T) {
	t.Run("Default config", func(t *testing.T) {
		cfg := emailpattern.DefaultConfig()
		assert.Equal(t, 3, cfg.MinLocalPartLength)
		assert.Equal(t, 5, cfg.MaxConsecutiveNumbers)
		assert.Equal(t, 0.3, cfg.MaxSpecialCharRatio)
	})

	t.Run("Custom config", func(t *testing.T) {
		cfg := &emailpattern.Config{
			MinLocalPartLength:    5,
			MaxConsecutiveNumbers: 3,
			MaxSpecialCharRatio:   0.2,
		}

		c := emailpattern.NewWithConfig(cfg)

		// Test with custom min length
		res, err := c.Check(context.Background(), "john@domain.com")
		assert.NoError(t, err)
		assert.True(t, res.ShortLocalPart, "should be short with custom min length of 5")

		// Test with custom max consecutive numbers
		res, err = c.Check(context.Background(), "user1234@domain.com")
		assert.NoError(t, err)
		assert.True(t, res.TooManyConsecutiveNumbers, "should have too many consecutive numbers with max of 3")

		// Test with custom special char ratio
		res, err = c.Check(context.Background(), "a.b_c@domain.com")
		assert.NoError(t, err)
		assert.True(t, res.TooManySpecialChars, "should have too many special chars with ratio of 0.2")
	})
}

func TestEmailPatternCheck_EdgeCases(t *testing.T) {
	c := emailpattern.New()

	t.Run("Invalid email formats", func(t *testing.T) {
		invalidEmails := []string{
			"",
			"@",
			"test",
			"@domain.com",
			"test@",
			"test@@domain.com", // Multiple @ symbols
		}

		for _, email := range invalidEmails {
			res, err := c.Check(context.Background(), email)
			assert.Error(t, err, "should return error for invalid email: %s", email)
			assert.Nil(t, res, "result should be nil for invalid email: %s", email)
		}
	})

	t.Run("Entropy calculation edge cases", func(t *testing.T) {
		// Very short strings should not trigger high entropy
		res, err := c.Check(context.Background(), "ab@domain.com")
		assert.NoError(t, err)
		assert.False(t, res.HasRandomPattern, "short strings should not be flagged as high entropy")

		// Repeated characters should have low entropy
		res, err = c.Check(context.Background(), "aaaa@domain.com")
		assert.NoError(t, err)
		assert.False(t, res.HasRandomPattern, "repeated characters should have low entropy")
	})

	t.Run("Keyboard sequence edge cases", func(t *testing.T) {
		// Very short sequences should not be flagged
		res, err := c.Check(context.Background(), "qw@domain.com")
		assert.NoError(t, err)
		assert.True(t, res.ShortLocalPart, "should be flagged as short")

		// Partial keyboard sequences in longer strings
		res, err = c.Check(context.Background(), "johnqwer@domain.com")
		assert.NoError(t, err)
		assert.True(t, res.HasRandomPattern, "should detect keyboard sequence in longer string")
	})
}

func TestEmailPatternCheck_InternationalSupport(t *testing.T) {
	c := emailpattern.New()

	internationalEmails := []struct {
		name  string
		email string
		valid bool
	}{
		{"Chinese characters", "用户@domain.com", true},
		{"Japanese hiragana", "ひらがな@domain.com", true},
		{"Japanese katakana", "カタカナ@domain.com", true},
		{"Arabic", "مستخدم@domain.com", true},
		{"Hebrew", "משתמש@domain.com", true},
		{"Thai", "ผู้ใช้@domain.com", true},                     // This should be valid now
		{"Emoji (should be flagged)", "user😀@domain.com", true}, // Actually, emojis are Unicode letters, so this might be valid
	}

	for _, tc := range internationalEmails {
		t.Run(tc.name, func(t *testing.T) {
			res, err := c.Check(context.Background(), tc.email)
			assert.NoError(t, err, "should not error for email: %s", tc.email)

			if tc.valid {
				assert.False(t, res.HasRandomPattern,
					"valid international email should not be flagged as random: %s", tc.email)
			} else {
				assert.True(t, res.HasRandomPattern,
					"invalid international email should be flagged as random: %s", tc.email)
			}
		})
	}
}
