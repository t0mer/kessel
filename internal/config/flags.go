package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RegisterFlags adds the configuration flags to cmd.
func RegisterFlags(cmd *cobra.Command) {
	d := Defaults()
	f := cmd.Flags()
	f.Int("port", d.Port, "HTTP listen port")
	f.String("data-dir", d.DataDir, "data directory (SQLite DB + reports)")
	f.String("psi-api-key", d.PSIAPIKey, "PageSpeed Insights API key")
	f.Int("psi-concurrency", d.PSIConcurrency, "max concurrent PSI checks")
	f.String("log-level", d.LogLevel, "log level: debug|info|warning|error")
	f.String("log-format", d.LogFormat, "log format: json|text")
	f.String("encryption-key", d.EncryptionKey, "hex-encoded 32-byte key for encrypting secrets at rest (else generated to data dir)")
	f.String("config", "", "path to YAML config file")
}

// Load resolves configuration with precedence flags > env (KESSEL_) > YAML.
func Load(cmd *cobra.Command) (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("KESSEL")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if err := v.BindPFlags(cmd.Flags()); err != nil {
		return Config{}, fmt.Errorf("binding flags: %w", err)
	}

	if cfgFile, _ := cmd.Flags().GetString("config"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("reading config %s: %w", cfgFile, err)
		}
	}

	c := Config{
		Port:           v.GetInt("port"),
		DataDir:        v.GetString("data-dir"),
		PSIAPIKey:      v.GetString("psi-api-key"),
		PSIConcurrency: v.GetInt("psi-concurrency"),
		LogLevel:       v.GetString("log-level"),
		LogFormat:      v.GetString("log-format"),
		EncryptionKey:  v.GetString("encryption-key"),
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return c, nil
}
