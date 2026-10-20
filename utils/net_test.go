package utils

import "testing"

// URL schemes are case-insensitive (RFC 3986 §3.1).
func TestIsHttpSchemeCase(t *testing.T) {
	cases := map[string]bool{
		"HTTP://example.com":  true,
		"Https://example.com": true,
		"hTtPs://example.com": true,
		"ftp://example.com":   false,
		"httpx://example.com": false,
		"FTP://example.com":   false,
	}
	for in, want := range cases {
		if got := IsHttp(in); got != want {
			t.Errorf("IsHttp(%q) = %v, want %v", in, got, want)
		}
	}
}
