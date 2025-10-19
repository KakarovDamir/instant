// Package config provides environment configuration validation
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ValidateEnv validates that all required environment variables are set
func ValidateEnv(requiredVars []string) error {
	var missing []string

	for _, varName := range requiredVars {
		value := os.Getenv(varName)
		if value == "" {
			missing = append(missing, varName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateSessionSecret ensures SESSION_SECRET meets minimum security requirements
func ValidateSessionSecret() error {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return errors.New("SESSION_SECRET is required")
	}

	return nil
}

// GetEnvOrDefault retrieves an environment variable or returns a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// MustGetEnv retrieves an environment variable or panics
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
