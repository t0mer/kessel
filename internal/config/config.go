// Package config loads Kessel configuration from flags, environment
// variables (KESSEL_ prefix), and an optional YAML file, with precedence
// flags > env > YAML.
package config

import "fmt"

// Config holds startup configuration. Runtime-changeable settings (API key,
// retention, thresholds) are stored in the DB and are not part of this struct.
type Config struct {
	Port           int
	DataDir        string
	PSIAPIKey      string
	PSIConcurrency int
	LogLevel       string
	LogFormat      string
}

// Defaults returns the built-in default configuration.
func Defaults() Config {
	return Config{
		Port:           8080,
		DataDir:        "/data",
		PSIAPIKey:      "",
		PSIConcurrency: 2,
		LogLevel:       "info",
		LogFormat:      "json",
	}
}

var validLevels = map[string]bool{"debug": true, "info": true, "warning": true, "error": true}
var validFormats = map[string]bool{"json": true, "text": true}

// Validate checks that the configuration is internally consistent.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d out of range 1-65535", c.Port)
	}
	if c.DataDir == "" {
		return fmt.Errorf("data-dir must not be empty")
	}
	if c.PSIConcurrency < 1 {
		return fmt.Errorf("psi-concurrency %d must be >= 1", c.PSIConcurrency)
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log-level %q (want debug|info|warning|error)", c.LogLevel)
	}
	if !validFormats[c.LogFormat] {
		return fmt.Errorf("invalid log-format %q (want json|text)", c.LogFormat)
	}
	return nil
}
