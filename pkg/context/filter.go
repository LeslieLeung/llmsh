package context

import (
	"regexp"
)

var sensitivePatterns = []*regexp.Regexp{
	// API Keys
	regexp.MustCompile(`(?i)(api[_-]?key|token)\s*[=:]\s*['"]?([a-zA-Z0-9_-]{20,})`),

	// AWS Keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// Passwords
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*['"]?([^\s'"]+)`),

	// Private Keys
	regexp.MustCompile(`-----BEGIN\s+.*PRIVATE\s+KEY-----`),

	// JWT
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),

	// Database URLs with credentials
	regexp.MustCompile(`(?i)(mysql|postgres|mongodb|redis)://[^:]+:[^@]+@`),

	// Bearer tokens
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_-]{20,}`),

	// SSH private key content
	regexp.MustCompile(`(?i)BEGIN\s+(RSA|DSA|EC|OPENSSH)\s+PRIVATE\s+KEY`),

	// Generic secrets (SECRET=value, secret: value)
	regexp.MustCompile(`(?i)(secret|credential)\s*[=:]\s*['"]?([^\s'"]+)`),

	// OAuth tokens
	regexp.MustCompile(`(?i)(oauth|access_token|refresh_token)\s*[=:]\s*['"]?([a-zA-Z0-9_-]{20,})`),
}

// FilterSensitive filters sensitive information from a list of commands
func FilterSensitive(commands []string) []string {
	filtered := make([]string, len(commands))
	for i, cmd := range commands {
		filtered[i] = filterCommand(cmd)
	}
	return filtered
}

// filterCommand filters sensitive information from a single command
func filterCommand(cmd string) string {
	result := cmd
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// IsSensitive checks if a command contains sensitive information
func IsSensitive(cmd string) bool {
	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(cmd) {
			return true
		}
	}
	return false
}
