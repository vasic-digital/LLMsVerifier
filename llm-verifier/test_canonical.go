package main

import (
	"fmt"
	"strings"
)

func canonicalHeaderKey(s string) string {
	// From Go source: textproto.CanonicalMIMEHeaderKey
	// Common headers already canonicalized
	common := map[string]string{
		"accept":          "Accept",
		"accept-charset":  "Accept-Charset",
		"accept-encoding": "Accept-Encoding",
		"accept-language": "Accept-Language",
		"accept-ranges":   "Accept-Ranges",
		"cache-control":   "Cache-Control",
		"cc":              "Cc",
		"connection":      "Connection",
		"content-type":    "Content-Type",
		"cookie":          "Cookie",
		"date":            "Date",
		"etag":            "Etag",
		"expect":          "Expect",
		"expires":         "Expires",
		"from":            "From",
		"host":            "Host",
		"if-match":        "If-Match",
		"if-modified-since": "If-Modified-Since",
		"if-none-match":   "If-None-Match",
		"if-range":        "If-Range",
		"if-unmodified-since": "If-Unmodified-Since",
		"last-modified":   "Last-Modified",
		"max-forwards":    "Max-Forwards",
		"pragma":          "Pragma",
		"proxy-authenticate": "Proxy-Authenticate",
		"proxy-authorization": "Proxy-Authorization",
		"range":           "Range",
		"referer":         "Referer",
		"refresh":         "Refresh",
		"retry-after":     "Retry-After",
		"server":          "Server",
		"set-cookie":      "Set-Cookie",
		"te":              "Te",
		"trailer":         "Trailer",
		"transfer-encoding": "Transfer-Encoding",
		"upgrade":         "Upgrade",
		"user-agent":      "User-Agent",
		"vary":            "Vary",
		"via":             "Via",
		"warning":         "Warning",
		"www-authenticate": "Www-Authenticate",
	}
	
	if canonical, ok := common[strings.ToLower(s)]; ok {
		return canonical
	}
	
	// Capitalize first letter and letters after hyphens
	upper := true
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if upper && 'a' <= c && c <= 'z' {
			c -= 'a' - 'A'
		} else if !upper && 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result = append(result, c)
		upper = c == '-'
	}
	return string(result)
}

func main() {
	tests := []string{
		"x-ratelimit-limit-requests",
		"x-ratelimit-limit-tokens",
		"x-ratelimit-remaining-requests",
		"x-ratelimit-remaining-tokens",
		"x-ratelimit-reset",
		"anthropic-ratelimit-requests-limit",
		"anthropic-ratelimit-tokens-limit",
		"anthropic-ratelimit-requests-remaining",
		"anthropic-ratelimit-tokens-remaining",
		"anthropic-ratelimit-reset",
		"x-rate-limit-limit",
		"x-rate-limit-limit-tokens",
		"x-rate-limit-remaining",
		"x-rate-limit-reset",
		"x-rate-limit-limit-requests-per-hour",
		"x-rate-limit-limit-requests-per-day",
	}
	
	for _, test := range tests {
		canonical := canonicalHeaderKey(test)
		fmt.Printf("%-45s -> %-45s\n", test, canonical)
	}
}
