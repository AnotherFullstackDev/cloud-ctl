package image

import "github.com/AnotherFullstackDev/cloud-ctl/internal/image/registry"

type BuildConfig struct {
	Cmd []string          `mapstructure:"cmd"`
	Env map[string]string `mapstructure:"env"`
	Dir string            `mapstructure:"dir"`
}

type Config struct {
	Repository string                                  `mapstructure:"repository"`
	Tag        string                                  `mapstructure:"tag"`
	Ghcr       *registry.GithubContainerRegistryConfig `mapstructure:"ghcr"`
	Build      BuildConfig                             `mapstructure:"build"`
}
