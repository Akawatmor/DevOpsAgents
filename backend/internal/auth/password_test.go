package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"too short", "ab12", ErrPasswordTooShort},
		{"no number", "abcdefgh", ErrPasswordNoNumber},
		{"valid simple", "abcdefg1", nil},
		{"valid complex", "P@ssw0rd!", nil},
		{"exact 8 chars with digit", "1234abcd", nil},
		{"empty", "", ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	if err := ValidateUsername(""); !errors.Is(err, ErrUsernameEmpty) {
		t.Fatalf("expected ErrUsernameEmpty, got %v", err)
	}
	if err := ValidateUsername("kong"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
