package config

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Services map[string]ServiceConfig `mapstructure:"services"`
	v        *viper.Viper
}

type ServiceConfig struct {
	Environments map[string]EnvironmentConfig `mapstructure:"environments"`
	Extras       map[string]any               `mapstructure:",remain"`
}

type EnvironmentConfig struct {
	Extras map[string]any `mapstructure:",remain"`
}

func newConfigFromViper(v *viper.Viper) (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	cfg.v = v
	return &cfg, nil
}

func NewConfigFromPath(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return newConfigFromViper(v)
}

func NewConfigFromReader(reader io.Reader) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(reader); err != nil {
		return nil, fmt.Errorf("reading config from reader: %w", err)
	}

	return newConfigFromViper(v)
}

func (c *Config) WithEnvironment(env string) (*Config, error) {
	newV := viper.New()

	if err := newV.MergeConfigMap(c.v.AllSettings()); err != nil {
		return nil, fmt.Errorf("merging config map from global config instance: %w", err)
	}

	envConfig := map[string]any{
		"services": map[string]any{},
	}
	for k, service := range c.Services {
		envPart, ok := service.Environments[env]
		if !ok {
			return nil, fmt.Errorf("environment '%s' not found in config", env)
		}
		envConfig["services"].(map[string]any)[k] = envPart.Extras
	}
	if err := newV.MergeConfigMap(envConfig); err != nil {
		return nil, fmt.Errorf("merging environment config map: %w", err)
	}

	var cfg Config
	if err := newV.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config with environment: %w", err)
	}

	cfg.v = newV
	return &cfg, nil
}

func (c *Config) LoadVariableServiceConfigPart(cfg any, service, partKey string, extraKeys ...string) error {
	keyParts := []string{"services", service, partKey}
	if len(extraKeys) > 0 {
		keyParts = append(keyParts, extraKeys...)
	}
	key := strings.Join(keyParts, ".")
	if !c.v.IsSet(key) {
		return fmt.Errorf("provider config not found for service %s and provider %s", service, partKey)
	}

	if err := c.v.UnmarshalKey(key, cfg); err != nil {
		return fmt.Errorf("unmarshaling provider config: %w", err)
	}

	return nil
}
