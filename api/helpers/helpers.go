package helpers

import (
	"net/url"
	"os"
	"strings"
)

func EnforceHTTP(url string) string {
	if url[:4] != "http" {
		return "http://" + url
	}
	return url
}

func RemoveDomainError(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false // treat unparseable URLs as unsafe
	}

	host := strings.TrimPrefix(parsed.Hostname(), "www.")
	domain := strings.TrimPrefix(os.Getenv("DOMAIN"), "www.")

	return host != domain
}
