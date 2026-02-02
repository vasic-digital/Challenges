package env

import (
	"net/url"
	"strings"
)

// RedactAPIKey masks an API key, showing only the first 4 and last 4 characters.
func RedactAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}

// RedactURL masks credentials in a URL string.
func RedactURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.User != nil {
		password, hasPassword := u.User.Password()
		if hasPassword {
			u.User = url.UserPassword(u.User.Username(), RedactAPIKey(password))
		}
	}
	return u.String()
}

// RedactHeaders masks sensitive header values.
func RedactHeaders(headers map[string]string) map[string]string {
	sensitive := map[string]bool{
		"authorization":       true,
		"x-api-key":           true,
		"api-key":             true,
		"x-auth-token":        true,
		"cookie":              true,
		"set-cookie":          true,
		"proxy-authorization": true,
	}

	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if sensitive[strings.ToLower(k)] {
			result[k] = RedactAPIKey(v)
		} else {
			result[k] = v
		}
	}
	return result
}

// ValidateAPIKeyFormat checks if an API key matches known provider formats.
func ValidateAPIKeyFormat(key string) bool {
	if key == "" {
		return false
	}
	knownPrefixes := []string{
		"sk-ant-", // Anthropic
		"sk-",     // OpenAI
		"gsk_",    // Groq
		"xai-",    // xAI
	}
	for _, prefix := range knownPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	// If no known prefix, accept if length >= 20
	return len(key) >= 20
}
