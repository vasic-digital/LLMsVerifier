package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Test 1: Map literal
	headers1 := http.Header{
		"x-ratelimit-limit-requests": []string{"60"},
	}
	
	fmt.Printf("Test 1 - Map literal:\n")
	fmt.Printf("  headers1.Get('x-ratelimit-limit-requests'): '%s'\n", headers1.Get("x-ratelimit-limit-requests"))
	fmt.Printf("  headers1.Get('X-Ratelimit-Limit-Requests'): '%s'\n", headers1.Get("X-Ratelimit-Limit-Requests"))
	
	// Test 2: Using Set
	headers2 := http.Header{}
	headers2.Set("x-ratelimit-limit-requests", "60")
	
	fmt.Printf("\nTest 2 - Using Set:\n")
	fmt.Printf("  headers2.Get('x-ratelimit-limit-requests'): '%s'\n", headers2.Get("x-ratelimit-limit-requests"))
	fmt.Printf("  headers2.Get('X-Ratelimit-Limit-Requests'): '%s'\n", headers2.Get("X-Ratelimit-Limit-Requests"))
	
	// Check internal representation
	fmt.Printf("\nHeaders1 keys: ")
	for k := range headers1 {
		fmt.Printf("'%s' ", k)
	}
	fmt.Printf("\nHeaders2 keys: ")
	for k := range headers2 {
		fmt.Printf("'%s' ", k)
	}
	fmt.Println()
}
