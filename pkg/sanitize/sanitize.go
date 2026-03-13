package sanitize

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Sanitizer provides input sanitization methods
type Sanitizer struct {
	// Strict mode enables more aggressive sanitization
	strict bool
}

// New creates a new sanitizer
func New() *Sanitizer {
	return &Sanitizer{
		strict: false,
	}
}

// NewStrict creates a new strict sanitizer
func NewStrict() *Sanitizer {
	return &Sanitizer{
		strict: true,
	}
}

// SanitizeString sanitizes a string input
func (s *Sanitizer) SanitizeString(input string) string {
	if input == "" {
		return ""
	}

	// Trim whitespace
	result := strings.TrimSpace(input)

	// Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	// In strict mode, apply additional sanitization
	if s.strict {
		// Remove control characters except newline, tab, carriage return
		result = removeControlChars(result)
	}

	return result
}

// SanitizeHTML sanitizes HTML to prevent XSS attacks
func (s *Sanitizer) SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}

	// First remove potentially dangerous patterns (before escaping)
	result := s.removeScriptTags(input)
	result = s.removeEventHandlers(result)
	result = s.removeDangerousProtocols(result)

	// Then escape HTML entities
	result = html.EscapeString(result)

	return result
}

// SanitizeURL sanitizes a URL
func (s *Sanitizer) SanitizeURL(input string) string {
	if input == "" {
		return ""
	}

	// Parse URL
	parsedURL, err := url.Parse(input)
	if err != nil {
		// If parsing fails, return empty string in strict mode
		if s.strict {
			return ""
		}
		// Otherwise, sanitize as string
		return s.SanitizeString(input)
	}

	// Check for dangerous protocols
	if s.isDangerousProtocol(parsedURL.Scheme) {
		if s.strict {
			return ""
		}
		parsedURL.Scheme = "https"
	}

	// Sanitize host
	parsedURL.Host = s.SanitizeString(parsedURL.Host)

	// Remove fragment and user info in strict mode
	if s.strict {
		parsedURL.Fragment = ""
		parsedURL.User = nil
	}

	return parsedURL.String()
}

// SanitizeSQL sanitizes input for SQL queries to prevent SQL injection
// Note: This is a basic sanitization. Always use prepared statements when possible.
func (s *Sanitizer) SanitizeSQL(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// Escape single quotes
	result = strings.ReplaceAll(result, "'", "''")

	// Remove SQL comments
	result = strings.ReplaceAll(result, "--", "")
	result = strings.ReplaceAll(result, "/*", "")
	result = strings.ReplaceAll(result, "*/", "")

	// Remove common SQL injection patterns
	result = s.removeSQLInjectionPatterns(result)

	return result
}

// SanitizeFilename sanitizes a filename to prevent path traversal
func (s *Sanitizer) SanitizeFilename(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// Remove path separators
	result = strings.ReplaceAll(result, "/", "")
	result = strings.ReplaceAll(result, "\\", "")

	// Remove double dots
	result = strings.ReplaceAll(result, "..", "")

	// Remove drive letters (Windows)
	if len(result) > 1 && result[1] == ':' {
		result = result[2:]
	}

	// Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	// Trim whitespace and dots
	result = strings.Trim(result, " .")

	// In strict mode, only allow alphanumeric, underscore, hyphen, and dot
	if s.strict {
		result = cleanFilename(result)
	}

	// Limit length to 255 characters
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// SanitizeEmail sanitizes an email address
func (s *Sanitizer) SanitizeEmail(input string) string {
	if input == "" {
		return ""
	}

	result := strings.ToLower(strings.TrimSpace(input))

	// Remove dangerous characters
	result = strings.ReplaceAll(result, "\n", "")
	result = strings.ReplaceAll(result, "\r", "")
	result = strings.ReplaceAll(result, "\x00", "")

	// Basic validation
	if !s.isValidEmail(result) {
		if s.strict {
			return ""
		}
	}

	return result
}

// SanitizePhoneNumber sanitizes a phone number
func (s *Sanitizer) SanitizePhoneNumber(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// Keep only digits, plus, hyphen, space, and parentheses
	var sb strings.Builder
	for _, r := range result {
		if unicode.IsDigit(r) || r == '+' || r == '-' || r == ' ' || r == '(' || r == ')' {
			sb.WriteRune(r)
		}
	}

	result = strings.TrimSpace(sb.String())

	return result
}

// SanitizeJSON sanitizes JSON input
func (s *Sanitizer) SanitizeJSON(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// Escape control characters
	result = escapeJSONControlChars(result)

	// Remove null bytes
	result = strings.ReplaceAll(result, "\x00", "")

	return result
}

// SanitizeCommandLine sanitizes command line input to prevent command injection
func (s *Sanitizer) SanitizeCommandLine(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// Remove command separators
	result = strings.ReplaceAll(result, ";", "")
	result = strings.ReplaceAll(result, "&", "")
	result = strings.ReplaceAll(result, "|", "")
	result = strings.ReplaceAll(result, "`", "")

	// Remove $() and ${} patterns
	result = strings.ReplaceAll(result, "$", "")

	// Remove pipes and redirects
	result = strings.ReplaceAll(result, ">", "")
	result = strings.ReplaceAll(result, "<", "")

	// Remove newlines
	result = strings.ReplaceAll(result, "\n", "")
	result = strings.ReplaceAll(result, "\r", "")

	// Remove parentheses
	result = strings.ReplaceAll(result, "(", "")
	result = strings.ReplaceAll(result, ")", "")

	// Trim whitespace
	result = strings.TrimSpace(result)

	return result
}

// SanitizeInput is a general-purpose sanitization function
func (s *Sanitizer) SanitizeInput(input string, inputType string) string {
	if input == "" {
		return ""
	}

	switch strings.ToLower(inputType) {
	case "html", "xhtml":
		return s.SanitizeHTML(input)
	case "url", "uri":
		return s.SanitizeURL(input)
	case "sql":
		return s.SanitizeSQL(input)
	case "filename", "file":
		return s.SanitizeFilename(input)
	case "email":
		return s.SanitizeEmail(input)
	case "phone", "telephone":
		return s.SanitizePhoneNumber(input)
	case "json":
		return s.SanitizeJSON(input)
	case "command", "cmd", "shell":
		return s.SanitizeCommandLine(input)
	default:
		return s.SanitizeString(input)
	}
}

// removeScriptTags removes script tags and their content
func (s *Sanitizer) removeScriptTags(input string) string {
	// Remove <script> tags and their content (non-greedy)
	re := regexp.MustCompile(`(?i)<\s*script[^>]*>.*?<\s*/\s*script\s*>`)
	return re.ReplaceAllString(input, "")
}

// removeEventHandlers removes inline event handlers
func (s *Sanitizer) removeEventHandlers(input string) string {
	// Remove event handlers like onclick, onerror, etc.
	re := regexp.MustCompile(`(?i)\s+on\w+\s*=\s*["'][^"']*["']`)
	return re.ReplaceAllString(input, "")
}

// removeDangerousProtocols removes dangerous URL protocols
func (s *Sanitizer) removeDangerousProtocols(input string) string {
	// Remove javascript:, vbscript:, data: protocols (even within attributes)
	re := regexp.MustCompile(`(?i)(javascript|vbscript|data):[^"'\s>]*`)
	return re.ReplaceAllString(input, "")
}

// isDangerousProtocol checks if a protocol is dangerous
func (s *Sanitizer) isDangerousProtocol(scheme string) bool {
	dangerousSchemes := []string{
		"javascript", "vbscript", "data", "file", "mailto", "ftp",
	}

	scheme = strings.ToLower(scheme)
	for _, dangerous := range dangerousSchemes {
		if scheme == dangerous {
			return true
		}
	}

	return false
}

// removeSQLInjectionPatterns removes common SQL injection patterns
func (s *Sanitizer) removeSQLInjectionPatterns(input string) string {
	// First, replace common injection markers
	result := input

	// Remove dangerous SQL keywords and their following content
	re := regexp.MustCompile(`(?i)\b(DROP|DELETE|INSERT|UPDATE|EXEC|EXECUTE)\b\s+\w+`)
	result = re.ReplaceAllString(result, "")

	// Remove UNION SELECT
	re = regexp.MustCompile(`(?i)UNION\s+SELECT`)
	result = re.ReplaceAllString(result, "")

	// Remove OR/AND with conditions
	re = regexp.MustCompile(`(?i)\b(OR|AND)\s+['"]?\w+['"]?\s*[=<>]`)
	result = re.ReplaceAllString(result, "")

	// Remove standalone OR/AND followed by non-whitespace
	re = regexp.MustCompile(`(?i)\b(OR|AND)\b\S*`)
	result = re.ReplaceAllString(result, "")

	// Remove comments
	result = strings.ReplaceAll(result, "--", "")
	result = strings.ReplaceAll(result, "/*", "")
	result = strings.ReplaceAll(result, "*/", "")

	// Remove quotes from injections
	result = strings.ReplaceAll(result, "'", "")
	result = strings.ReplaceAll(result, "\"", "")

	// Remove semicolons
	result = strings.ReplaceAll(result, ";", "")

	return result
}

// removeControlChars removes control characters except \n, \t, \r
func removeControlChars(input string) string {
	var sb strings.Builder
	for _, r := range input {
		if r == '\n' || r == '\t' || r == '\r' {
			sb.WriteRune(r)
		} else if !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// cleanFilename keeps only safe characters in filename
func cleanFilename(input string) string {
	var sb strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// isValidEmail performs basic email validation
func (s *Sanitizer) isValidEmail(email string) bool {
	// Basic email regex pattern
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// escapeJSONControlChars escapes control characters for JSON
func escapeJSONControlChars(input string) string {
	replacements := map[rune]string{
		'\x00': "\\u0000",
		'\x01': "\\u0001",
		'\x02': "\\u0002",
		'\x03': "\\u0003",
		'\x04': "\\u0004",
		'\x05': "\\u0005",
		'\x06': "\\u0006",
		'\x07': "\\u0007",
		'\x08': "\\u0008",
		'\x0B': "\\u000B",
		'\x0C': "\\u000C",
		'\x0E': "\\u000E",
		'\x0F': "\\u000F",
		'\x10': "\\u0010",
		'\x11': "\\u0011",
		'\x12': "\\u0012",
		'\x13': "\\u0013",
		'\x14': "\\u0014",
		'\x15': "\\u0015",
		'\x16': "\\u0016",
		'\x17': "\\u0017",
		'\x18': "\\u0018",
		'\x19': "\\u0019",
		'\x1A': "\\u001A",
		'\x1B': "\\u001B",
		'\x1C': "\\u001C",
		'\x1D': "\\u001D",
		'\x1E': "\\u001E",
		'\x1F': "\\u001F",
	}

	result := input
	for char, replacement := range replacements {
		result = strings.ReplaceAll(result, string(char), replacement)
	}

	return result
}

// ValidateString validates that a string doesn't contain dangerous patterns
func (s *Sanitizer) ValidateString(input string) bool {
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		return false
	}

	// Check for path traversal attempts
	if strings.Contains(input, "../") || strings.Contains(input, "..\\") {
		return false
	}

	// In strict mode, check for control characters
	if s.strict {
		for _, r := range input {
			if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
				return false
			}
		}
	}

	return true
}

// ValidateHTML validates HTML content
func (s *Sanitizer) ValidateHTML(input string) bool {
	// Check for script tags
	re := regexp.MustCompile(`(?i)<script\b`)
	if re.MatchString(input) {
		return false
	}

	// Check for dangerous protocols
	if s.hasDangerousProtocols(input) {
		return false
	}

	// Check for event handlers
	re = regexp.MustCompile(`(?i)\s+on\w+\s*=`)
	if re.MatchString(input) {
		return false
	}

	return true
}

// hasDangerousProtocols checks if input contains dangerous protocols
func (s *Sanitizer) hasDangerousProtocols(input string) bool {
	re := regexp.MustCompile(`(?i)(javascript|vbscript|data):`)
	return re.MatchString(input)
}

// SanitizeMap sanitizes all string values in a map
func (s *Sanitizer) SanitizeMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}

	result := make(map[string]string, len(input))
	for key, value := range input {
		sanitizedKey := s.SanitizeString(key)
		sanitizedValue := s.SanitizeString(value)
		result[sanitizedKey] = sanitizedValue
	}

	return result
}

// SanitizeSlice sanitizes all strings in a slice
func (s *Sanitizer) SanitizeSlice(input []string) []string {
	if input == nil {
		return nil
	}

	result := make([]string, len(input))
	for i, value := range input {
		result[i] = s.SanitizeString(value)
	}

	return result
}

// DeepSanitize recursively sanitizes nested structures
func (s *Sanitizer) DeepSanitize(input interface{}) interface{} {
	switch v := input.(type) {
	case string:
		return s.SanitizeString(v)
	case []string:
		return s.SanitizeSlice(v)
	case map[string]string:
		return s.SanitizeMap(v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = s.DeepSanitize(item)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			sanitizedKey := s.SanitizeString(key)
			result[sanitizedKey] = s.DeepSanitize(value)
		}
		return result
	default:
		return v
	}
}
