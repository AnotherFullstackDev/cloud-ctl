package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Services map[string]ServiceConfig `mapstructure:"services"`
	v        *viper.Viper
}

type ServiceConfig struct {
	//Image        ImageConfig                  `mapstructure:"image"`
	Environments map[string]EnvironmentConfig `mapstructure:"environments"`
	Extras       map[string]any               `mapstructure:",remain"`
}

//type ImageConfig struct {
//	Repository string         `mapstructure:"repository"`
//	Tag        string         `mapstructure:"tag"`
//	Build      map[string]any `mapstructure:"build"`
//}

type EnvironmentConfig struct {
	Extras map[string]any `mapstructure:",remain"`
}

func NewConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	cfg.v = v
	return &cfg, nil
}

func LoadVariableServiceConfigPart[T any](c *Config, service, partKey string, extraKeys ...string) (T, error) {
	keyParts := []string{"services", service, partKey}
	if len(extraKeys) > 0 {
		keyParts = append(keyParts, extraKeys...)
	}
	key := strings.Join(keyParts, ".")
	if !c.v.IsSet(key) {
		return *new(T), fmt.Errorf("provider config not found for service %s and provider %s", service, partKey)
	}

	var cfg T
	if err := c.v.UnmarshalKey(key, &cfg); err != nil {
		return *new(T), fmt.Errorf("unmarshaling provider config: %w", err)
	}

	return cfg, nil
}
