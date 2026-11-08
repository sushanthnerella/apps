package auth

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{"Canonical E.164", "+919849012345", "+919849012345", false},
		{"Spaced 2-5-5", "+91 98490 12345", "+919849012345", false},
		{"Hyphenated", "+91-98490-12345", "+919849012345", false},
		{"10 digits standard", "9849012345", "+919849012345", false},
		{"11 digits with 0 trunk", "09849012345", "+919849012345", false},
		{"12 digits with 91 prefix", "919849012345", "+919849012345", false},
		{"Parentheses format", "(098490) 12345", "+919849012345", false},
		{"International E.164 US", "+14155552671", "+14155552671", false},
		{"Empty string", "", "", true},
		{"Whitespace only", "   ", "", true},
		{"Too short", "12345", "", true},
		{"Non-numeric characters", "98490ABCDE", "", true},
		{"Too long", "984901234512345678", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhone(tt.input)
			if (err != nil) != tt.expectErr {
				t.Fatalf("NormalizePhone(%q) error = %v, expectErr %v", tt.input, err, tt.expectErr)
			}
			if !tt.expectErr && got != tt.expected {
				t.Errorf("NormalizePhone(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPhoneLookupVariants(t *testing.T) {
	variants, err := PhoneLookupVariants("+91 98490 12345")
	if err != nil {
		t.Fatalf("PhoneLookupVariants failed: %v", err)
	}

	// Must contain canonical and legacy spaced formats
	expectedMembers := []string{
		"+919849012345",
		"+91 98490 12345",
		"9849012345",
	}

	variantMap := make(map[string]bool)
	for _, v := range variants {
		variantMap[v] = true
	}

	for _, exp := range expectedMembers {
		if !variantMap[exp] {
			t.Errorf("expected variants to contain %q, but variants were %v", exp, variants)
		}
	}
}

func TestMaskPhone(t *testing.T) {
	masked := MaskPhone("+919849012345")
	if masked == "+919849012345" {
		t.Errorf("MaskPhone failed to mask phone number: got %s", masked)
	}
	if len(masked) == 0 {
		t.Errorf("MaskPhone returned empty string")
	}
}
