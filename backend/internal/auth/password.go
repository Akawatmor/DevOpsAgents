package auth

import (
	"errors"
	"unicode"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordNoNumber = errors.New("password must contain at least 1 number")
	ErrUsernameEmpty    = errors.New("username must not be empty")
)

// ValidatePassword enforces the rules:
//   - At least 8 characters
//   - At least 1 number
func ValidatePassword(pw string) error {
	if len(pw) < 8 {
		return ErrPasswordTooShort
	}
	hasNumber := false
	for _, r := range pw {
		if unicode.IsDigit(r) {
			hasNumber = true
			break
		}
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	return nil
}

func ValidateUsername(u string) error {
	if len(u) == 0 {
		return ErrUsernameEmpty
	}
	return nil
}
