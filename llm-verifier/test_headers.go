package main

import (
	"fmt"
	"net/http"
)

func main() {
	headers := http.Header{
		"x-ratelimit-limit-requests":     []string{"60"},
		"x-ratelimit-limit-tokens":       []string{"40000"},
	}
	
	fmt.Printf("Get('x-ratelimit-limit-requests'): %s\n", headers.Get("x-ratelimit-limit-requests"))
	fmt.Printf("Get('X-Ratelimit-Limit-Requests'): %s\n", headers.Get("X-Ratelimit-Limit-Requests"))
	
	// Check all keys
	fmt.Printf("\nAll keys:\n")
	for k, v := range headers {
		fmt.Printf("  %s: %v\n", k, v)
	}
}
