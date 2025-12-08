package container_image

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/build/pipeline"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image/registry"
)

type BuildConfig struct {
	Cmd      []string          `mapstructure:"cmd"`
	Pipeline *pipeline.Config  `mapstructure:"pipeline"`
	Env      map[string]string `mapstructure:"env"`
	Dir      string            `mapstructure:"dir"`
}

type Config struct {
	Image    string         `mapstructure:"image"`
	Build    BuildConfig    `mapstructure:"build"`
	Registry RegistryConfig `mapstructure:"registry"`
}

type RegistryConfig struct {
	Ghcr   *registry.GithubContainerRegistryConfig `mapstructure:"ghcr"`
	AWSEcr *registry.AwsECRConfig                  `mapstructure:"aws_ecr"`
}
