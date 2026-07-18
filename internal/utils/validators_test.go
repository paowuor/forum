package utils_test

import (
	"testing"

	"forum/internal/utils"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"alice@example.com",
		"a.b+tag@sub.example.co",
		"user_name@example-domain.io",
	}
	for _, email := range valid {
		if err := utils.ValidateEmail(email); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", email, err)
		}
	}

	invalid := []string{
		"",
		"notanemail",
		"missing-at.com",
		"@example.com",
		"alice@",
		"alice@nodot",
		"alice example@example.com",
	}
	for _, email := range invalid {
		if err := utils.ValidateEmail(email); err == nil {
			t.Errorf("expected %q to be invalid, got no error", email)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	// Boundary lengths matter most here: the check is len < 3 || len > 20.
	cases := []struct {
		username string
		wantErr  bool
	}{
		{"ab", true},                       // 2 chars: too short
		{"abc", false},                     // 3 chars: exactly the minimum
		{"12345678901234567890", false},    // 20 chars: exactly the maximum
		{"123456789012345678901", true},    // 21 chars: one over the maximum
		{"", true},
	}
	for _, c := range cases {
		err := utils.ValidateUsername(c.username)
		if c.wantErr && err == nil {
			t.Errorf("username %q (len %d): expected an error, got nil", c.username, len(c.username))
		}
		if !c.wantErr && err != nil {
			t.Errorf("username %q (len %d): expected no error, got %v", c.username, len(c.username), err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		password string
		wantErr  bool
	}{
		{"1234567", true},   // 7 chars: too short
		{"12345678", false}, // 8 chars: exactly the minimum
		{"a-fairly-long-passphrase", false},
		{"", true},
	}
	for _, c := range cases {
		err := utils.ValidatePassword(c.password)
		if c.wantErr && err == nil {
			t.Errorf("password of length %d: expected an error, got nil", len(c.password))
		}
		if !c.wantErr && err != nil {
			t.Errorf("password of length %d: expected no error, got %v", len(c.password), err)
		}
	}
}
