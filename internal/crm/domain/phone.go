package domain

import (
	"regexp"
	"strings"
)

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

var phoneReplacer = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

// NormalizePhone trims whitespace, strips spaces/hyphens/parentheses,
// and validates E.164 format (+ followed by 7 to 15 digits, non-zero first digit).
func NormalizePhone(raw string) (string, error) {
	cleaned := phoneReplacer.Replace(strings.TrimSpace(raw))
	if !e164Pattern.MatchString(cleaned) {
		return "", InvalidFieldError{
			Field:  "phone",
			Reason: "must be in E.164 format (+ followed by 7 to 15 digits)",
		}
	}
	return cleaned, nil
}
