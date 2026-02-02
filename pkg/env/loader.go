package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Loader defines the interface for environment variable management.
type Loader interface {
	// Load reads environment variables from a .env file.
	Load(filepath string) error
	// Get retrieves an environment variable value.
	Get(key string) string
	// GetRequired retrieves a required environment variable or returns error.
	GetRequired(key string) (string, error)
	// GetWithDefault retrieves an environment variable with a default fallback.
	GetWithDefault(key, defaultValue string) string
	// GetAPIKey retrieves an API key for a named provider.
	GetAPIKey(provider string) string
	// Set sets an environment variable.
	Set(key, value string) error
	// All returns all loaded environment variables.
	All() map[string]string
}

// DefaultLoader implements Loader with .env file support and provider mappings.
type DefaultLoader struct {
	mu       sync.RWMutex
	vars     map[string]string
	loaded   bool
	mappings map[string]string // provider name -> env var name
}

// NewLoader creates a new DefaultLoader with standard provider API key mappings.
func NewLoader() *DefaultLoader {
	return &DefaultLoader{
		vars: make(map[string]string),
		mappings: map[string]string{
			"claude":     "ANTHROPIC_API_KEY",
			"anthropic":  "ANTHROPIC_API_KEY",
			"deepseek":   "DEEPSEEK_API_KEY",
			"gemini":     "GEMINI_API_KEY",
			"google":     "GEMINI_API_KEY",
			"mistral":    "MISTRAL_API_KEY",
			"openrouter": "OPENROUTER_API_KEY",
			"qwen":       "QWEN_API_KEY",
			"zai":        "ZAI_API_KEY",
			"cerebras":   "CEREBRAS_API_KEY",
			"openai":     "OPENAI_API_KEY",
			"ollama":     "OLLAMA_API_KEY",
		},
	}
}

// NewLoaderWithMappings creates a loader with custom provider-to-env-var mappings.
func NewLoaderWithMappings(mappings map[string]string) *DefaultLoader {
	l := NewLoader()
	for k, v := range mappings {
		l.mappings[strings.ToLower(k)] = v
	}
	return l
}

func (l *DefaultLoader) Load(filepath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("open env file %s: %w", filepath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove surrounding quotes
		value = strings.Trim(value, `"'`)
		l.vars[key] = value
	}

	l.loaded = true
	return scanner.Err()
}

func (l *DefaultLoader) Get(key string) string {
	// OS env takes precedence
	if v := os.Getenv(key); v != "" {
		return v
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.vars[key]
}

func (l *DefaultLoader) GetRequired(key string) (string, error) {
	v := l.Get(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func (l *DefaultLoader) GetWithDefault(key, defaultValue string) string {
	if v := l.Get(key); v != "" {
		return v
	}
	return defaultValue
}

func (l *DefaultLoader) GetAPIKey(provider string) string {
	l.mu.RLock()
	envVar, ok := l.mappings[strings.ToLower(provider)]
	l.mu.RUnlock()
	if !ok {
		// Try uppercase provider + _API_KEY
		envVar = strings.ToUpper(provider) + "_API_KEY"
	}
	return l.Get(envVar)
}

func (l *DefaultLoader) Set(key, value string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.vars[key] = value
	return os.Setenv(key, value)
}

func (l *DefaultLoader) All() map[string]string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make(map[string]string, len(l.vars))
	for k, v := range l.vars {
		result[k] = v
	}
	return result
}
