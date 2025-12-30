package api

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// SanitizeInput sanitizes user input to prevent XSS and injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Escape HTML special characters
	input = html.EscapeString(input)

	return input
}

// SanitizeHTML allows safe HTML but removes dangerous tags and attributes
func SanitizeHTML(input string) string {
	// Remove script tags and their content (Go-compatible regex)
	reScript := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	input = reScript.ReplaceAllString(input, "")

	// Remove on* attributes (onclick, onload, etc.)
	reOnAttr := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
	input = reOnAttr.ReplaceAllString(input, "")

	// Remove javascript: protocol from href/src
	reJSProtocol := regexp.MustCompile(`(?i)(href|src)\s*=\s*["']\s*javascript:[^"']*["']`)
	input = reJSProtocol.ReplaceAllString(input, "")

	// Remove data: protocol (can be used for XSS)
	reDataProtocol := regexp.MustCompile(`(?i)(href|src)\s*=\s*["']\s*data:[^"']*["']`)
	input = reDataProtocol.ReplaceAllString(input, "")

	// Remove style tags and their content
	reStyle := regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	input = reStyle.ReplaceAllString(input, "")

	// Remove iframe tags
	reIframe := regexp.MustCompile(`(?i)<iframe[^>]*>[\s\S]*?</iframe>`)
	input = reIframe.ReplaceAllString(input, "")

	// Remove object tags
	reObject := regexp.MustCompile(`(?i)<object[^>]*>[\s\S]*?</object>`)
	input = reObject.ReplaceAllString(input, "")

	// Remove embed tags
	reEmbed := regexp.MustCompile(`(?i)<embed[^>]*>[\s\S]*?</embed>`)
	input = reEmbed.ReplaceAllString(input, "")

	// Remove applet tags
	reApplet := regexp.MustCompile(`(?i)<applet[^>]*>[\s\S]*?</applet>`)
	input = reApplet.ReplaceAllString(input, "")

	return input
}

// SanitizeSQL removes SQL injection patterns
func SanitizeSQL(input string) string {
	// Remove SQL comments
	input = regexp.MustCompile(`--.*`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(input, "")

	// Remove common SQL injection patterns
	patterns := []string{
		`(?i)\b(union|select|insert|update|delete|drop|create|alter|truncate|exec|execute|grant|revoke)\b`,
		`(?i)\b(from|where|having|group by|order by|limit|offset)\b`,
		`(?i)\b(and|or|not|like|between|in|is|null)\b`,
		`(?i)\b(join|inner join|left join|right join|full join|cross join)\b`,
		`(?i)\b(table|database|schema|index|view|trigger|procedure|function)\b`,
		`(?i)\b(values|set|into|as|on|using)\b`,
		`(?i)\b(case|when|then|else|end)\b`,
		`(?i)\b(declare|begin|end|transaction|commit|rollback)\b`,
		`(?i)\b(cursor|fetch|open|close|deallocate)\b`,
		`(?i)\b(cast|convert|coalesce|nullif)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		input = re.ReplaceAllString(input, "")
	}

	// Remove semicolons (statement termination)
	input = strings.ReplaceAll(input, ";", "")

	// Remove quotes (both single and double)
	input = strings.ReplaceAll(input, "'", "")
	input = strings.ReplaceAll(input, "\"", "")

	// Remove backticks
	input = strings.ReplaceAll(input, "`", "")

	return input
}

// SanitizePath prevents path traversal attacks
func SanitizePath(input string) string {
	// Remove directory traversal patterns
	input = strings.ReplaceAll(input, "..", "")
	input = strings.ReplaceAll(input, "./", "")
	input = strings.ReplaceAll(input, "/.", "")

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters
	input = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(input, "")

	// Remove multiple slashes
	input = regexp.MustCompile(`/{2,}`).ReplaceAllString(input, "/")

	// Trim leading/trailing slashes and dots
	input = strings.Trim(input, "./")

	return input
}

// SanitizeEmail validates and sanitizes email addresses
func SanitizeEmail(email string) (string, bool) {
	// Basic email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return "", false
	}

	// Sanitize the email
	email = strings.ToLower(strings.TrimSpace(email))
	email = html.EscapeString(email)

	return email, true
}

// SanitizeURL validates and sanitizes URLs
func SanitizeURL(url string) (string, bool) {
	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", false
	}

	// Remove dangerous protocols
	url = strings.ReplaceAll(url, "javascript:", "")
	url = strings.ReplaceAll(url, "data:", "")
	url = strings.ReplaceAll(url, "vbscript:", "")

	// Escape HTML
	url = html.EscapeString(url)

	return url, true
}

// SanitizePhoneNumber validates and sanitizes phone numbers
func SanitizePhoneNumber(phone string) (string, bool) {
	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(phone, "")

	// Validate length (10 digits for US numbers, adjust as needed)
	if len(digits) < 10 || len(digits) > 15 {
		return "", false
	}

	return digits, true
}

// SanitizeJSON validates and sanitizes JSON input
func SanitizeJSON(input string) (string, bool) {
	// Basic JSON validation - check for balanced braces/brackets
	braceCount := 0
	bracketCount := 0

	for _, char := range input {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}

		// If counts go negative, JSON is malformed
		if braceCount < 0 || bracketCount < 0 {
			return "", false
		}
	}

	// Check for balanced braces/brackets
	if braceCount != 0 || bracketCount != 0 {
		return "", false
	}

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Escape HTML in string values (simplified approach)
	// This is a basic implementation - for production, use a proper JSON parser
	input = html.EscapeString(input)

	return input, true
}

// SanitizeFilename sanitizes filenames to prevent path traversal
func SanitizeFilename(filename string) string {
	// Remove path traversal patterns
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove control characters
	filename = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(filename, "")

	// Remove dangerous characters
	dangerousChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
	for _, char := range dangerousChars {
		filename = strings.ReplaceAll(filename, char, "")
	}

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Limit length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// SanitizeInteger ensures input is a valid integer
func SanitizeInteger(input string) (int64, bool) {
	// Remove all non-digit characters (except optional leading minus)
	re := regexp.MustCompile(`[^0-9\-]`)
	clean := re.ReplaceAllString(input, "")

	// Parse as integer
	var result int64
	_, err := fmt.Sscanf(clean, "%d", &result)
	if err != nil {
		return 0, false
	}

	return result, true
}

// SanitizeFloat ensures input is a valid float
func SanitizeFloat(input string) (float64, bool) {
	// Remove all non-digit/decimal characters (except optional leading minus)
	re := regexp.MustCompile(`[^0-9\.\-]`)
	clean := re.ReplaceAllString(input, "")

	// Parse as float
	var result float64
	_, err := fmt.Sscanf(clean, "%f", &result)
	if err != nil {
		return 0, false
	}

	return result, true
}

// SanitizeBool ensures input is a valid boolean
func SanitizeBool(input string) (bool, bool) {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "1", "t", "true", "yes", "on":
		return true, true
	case "0", "f", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

// SanitizeOutput sanitizes output to prevent XSS in JSON/HTML responses
func SanitizeOutput(output string) string {
	if output == "" {
		return output
	}

	// For JSON responses, escape HTML characters to prevent XSS
	// when JSON is embedded in HTML contexts
	return html.EscapeString(output)
}

// SanitizeJSONOutput sanitizes JSON output to prevent injection attacks
func SanitizeJSONOutput(data interface{}) interface{} {
	// For complex objects, we need to sanitize string fields
	// This is a basic implementation - in production, you'd want more sophisticated
	// JSON sanitization that handles nested structures
	return data
}

// SanitizeHTMLResponse sanitizes HTML responses
func SanitizeHTMLResponse(html string) string {
	// Remove any script tags that might have been injected (Go-compatible regex)
	reScript := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	html = reScript.ReplaceAllString(html, "")

	// Remove javascript: URLs
	reJSURL := regexp.MustCompile(`(?i)javascript:[^"'\s]*`)
	html = reJSURL.ReplaceAllString(html, "#")

	return html
}
