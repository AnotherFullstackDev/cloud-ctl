package config

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func configToReader(config string) io.Reader {
	return io.NopCloser(strings.NewReader(config))
}

const configYAML = `
services:
  service1:
    some_key: 'some_value'
    some_other_key: 42
    nested_key:
      nested_value: 123
    some_other_nested_key:
      another_nested_value: true
    environments:
      dev:
        env_key: 'dev_value'
        some_other_key: 100
        some_other_nested_key:
          another_nested_value: false
`

func TestConfig(t *testing.T) {
	r := require.New(t)

	t.Run("must parse config", func(t *testing.T) {
		cfg, err := NewConfigFromReader(configToReader(configYAML))
		r.NoError(err)
		r.Equal(cfg.Services["service1"].Extras["some_key"], "some_value")
		r.Equal(cfg.Services["service1"].Extras["nested_key"].(map[string]any)["nested_value"], 123)
		r.Equal(cfg.Services["service1"].Environments["dev"].Extras["env_key"], "dev_value")
	})

	t.Run("must parse config with environment", func(t *testing.T) {
		cfg, err := NewConfigFromReader(configToReader(configYAML))
		r.NoError(err)

		cfgWithEnv, err := cfg.WithEnvironment("dev")
		r.NoError(err)

		r.Equal(cfgWithEnv.Services["service1"].Extras["some_key"], "some_value")
		r.Equal(cfgWithEnv.Services["service1"].Extras["some_other_key"], 100)
		r.Equal(cfgWithEnv.Services["service1"].Extras["nested_key"].(map[string]any)["nested_value"], 123)
		r.Equal(cfgWithEnv.Services["service1"].Extras["some_other_nested_key"].(map[string]any)["another_nested_value"], false)
		r.Equal(cfgWithEnv.Services["service1"].Extras["env_key"], "dev_value")

		r.Equal(cfg.Services["service1"].Extras["some_key"], "some_value")
		r.Equal(cfg.Services["service1"].Extras["nested_key"].(map[string]any)["nested_value"], 123)
		r.Equal(cfg.Services["service1"].Environments["dev"].Extras["env_key"], "dev_value")
	})
}
