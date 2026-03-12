package canalbox

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var ErrSessionExpired = errors.New("session expired")

func wrapSessionExpired(message string) error {
	if strings.TrimSpace(message) == "" {
		return ErrSessionExpired
	}

	return fmt.Errorf("%w: %s", ErrSessionExpired, message)
}

func isSessionExpiredStatus(statusCode int) bool {
	return statusCode == http.StatusUnauthorized
}

func isSessionExpiredMessage(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}

	checks := []string{
		"invalid_session_id",
		"session expired",
		"session has expired",
		"session is invalid",
		"invalid session",
		"invalid session id",
		"not authenticated",
		"login required",
		"You do not have access",
	}

	for _, check := range checks {
		if strings.Contains(msg, check) {
			return true
		}
	}

	return false
}
