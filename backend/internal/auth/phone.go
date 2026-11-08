package auth

import (
	"fmt"
	"strings"
	"unicode"
)

// NormalizePhone normalizes phone numbers into canonical E.164 format (+91XXXXXXXXXX for Indian numbers).
// It safely handles space-separated, hyphenated, leading-zero, and 10-digit formats.
func NormalizePhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("phone number cannot be empty")
	}

	// Remove common separators (spaces, hyphens, parentheses, dots)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' || r == '.' || r == '(' || r == ')' {
			return -1
		}
		return r
	}, trimmed)

	// Check if starting with '+'
	if strings.HasPrefix(cleaned, "+") {
		digits := cleaned[1:]
		if !isAllDigits(digits) {
			return "", fmt.Errorf("invalid characters in phone number: %s", MaskPhone(trimmed))
		}
		if len(digits) < 10 || len(digits) > 15 {
			return "", fmt.Errorf("phone number must contain between 10 and 15 digits")
		}
		// If Indian E.164 (+91 followed by 10 digits)
		if strings.HasPrefix(digits, "91") && len(digits) == 12 {
			return "+" + digits, nil
		}
		return "+" + digits, nil
	}

	// Not starting with '+'
	if !isAllDigits(cleaned) {
		return "", fmt.Errorf("invalid characters in phone number: %s", MaskPhone(trimmed))
	}

	// 10-digit Indian mobile number
	if len(cleaned) == 10 {
		return "+91" + cleaned, nil
	}

	// 11 digits starting with '0' (trunk prefix)
	if len(cleaned) == 11 && strings.HasPrefix(cleaned, "0") {
		return "+91" + cleaned[1:], nil
	}

	// 12 digits starting with '91'
	if len(cleaned) == 12 && strings.HasPrefix(cleaned, "91") {
		return "+" + cleaned, nil
	}

	return "", fmt.Errorf("unsupported phone number format: %s (must be 10 digits or E.164)", MaskPhone(trimmed))
}

// PhoneLookupVariants returns all likely database representations of a phone number
// to safely resolve legacy records stored with or without spaces/country-codes.
func PhoneLookupVariants(raw string) ([]string, error) {
	canonical, err := NormalizePhone(raw)
	if err != nil {
		return nil, err
	}

	variantMap := make(map[string]struct{})
	variantMap[canonical] = struct{}{}
	variantMap[strings.TrimSpace(raw)] = struct{}{}

	// If canonical is +91XXXXXXXXXX (13 chars)
	if strings.HasPrefix(canonical, "+91") && len(canonical) == 13 {
		tenDigits := canonical[3:] // XXXXXXXXXX
		variantMap[tenDigits] = struct{}{}
		variantMap["0"+tenDigits] = struct{}{}
		variantMap["91"+tenDigits] = struct{}{}
		variantMap["+91 "+tenDigits] = struct{}{}

		// 2-5-5 space separated format: "+91 98490 12345"
		spaced255 := fmt.Sprintf("+91 %s %s", tenDigits[:5], tenDigits[5:])
		variantMap[spaced255] = struct{}{}

		// 5-5 space separated format: "98490 12345"
		variantMap[fmt.Sprintf("%s %s", tenDigits[:5], tenDigits[5:])] = struct{}{}
	}

	variants := make([]string, 0, len(variantMap))
	for v := range variantMap {
		if v != "" {
			variants = append(variants, v)
		}
	}
	return variants, nil
}

// MaskPhone safely masks a phone number for audit logging without exposing PII.
func MaskPhone(phone string) string {
	s := strings.TrimSpace(phone)
	if len(s) <= 4 {
		return "****"
	}
	if len(s) <= 8 {
		return s[:2] + "****" + s[len(s)-2:]
	}
	return s[:4] + "****" + s[len(s)-3:]
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
