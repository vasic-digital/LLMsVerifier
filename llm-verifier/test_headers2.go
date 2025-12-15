package main

import (
	"fmt"
	"net/http"
)

func main() {
	headers := http.Header{}
	headers.Set("x-ratelimit-limit-requests", "60")
	headers.Set("X-Ratelimit-Limit-Tokens", "40000")
	
	fmt.Printf("Get('x-ratelimit-limit-requests'): %s\n", headers.Get("x-ratelimit-limit-requests"))
	fmt.Printf("Get('X-Ratelimit-Limit-Requests'): %s\n", headers.Get("X-Ratelimit-Limit-Requests"))
	fmt.Printf("Get('x-ratelimit-limit-tokens'): %s\n", headers.Get("x-ratelimit-limit-tokens"))
	fmt.Printf("Get('X-Ratelimit-Limit-Tokens'): %s\n", headers.Get("X-Ratelimit-Limit-Tokens"))
	
	// Check all keys
	fmt.Printf("\nAll keys:\n")
	for k, v := range headers {
		fmt.Printf("  %s: %v\n", k, v)
	}
}
