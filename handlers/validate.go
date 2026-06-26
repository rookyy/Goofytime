package handlers

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) bool {
	if email == "" {
		return true
	}
	return emailRegex.MatchString(email)
}

func validateLength(s string, maxLen int) string {
	if utf8.RuneCountInString(s) > maxLen {
		r := []rune(s)
		return string(r[:maxLen])
	}
	return s
}

func validateEmails(emails string) bool {
	if emails == "" {
		return true
	}
	for _, e := range strings.Split(emails, ",") {
		e = strings.TrimSpace(e)
		if e != "" && !emailRegex.MatchString(e) {
			return false
		}
	}
	return true
}
