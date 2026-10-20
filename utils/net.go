package utils

import "strings"

func IsHttp(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
