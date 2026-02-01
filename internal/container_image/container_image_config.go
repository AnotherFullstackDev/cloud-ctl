package container_image

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/build/pipeline"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image/registry"
)

type BuildConfig struct {
	// cmd build definition
	Cmd []string          `mapstructure:"cmd"`
	Env map[string]string `mapstructure:"env"`
	Dir string            `mapstructure:"dir"`

	// pipeline build definition
	Pipeline *pipeline.Config `mapstructure:"pipeline"`
}

type CompressionAlgorithm string

const (
	CompressionGzip CompressionAlgorithm = "gzip"
	CompressionZstd CompressionAlgorithm = "zstd"
	CompressionNone CompressionAlgorithm = "none"
)

type CompressionConfig struct {
	Algorithm CompressionAlgorithm `mapstructure:"algorithm"`
	Level     int                  `mapstructure:"level"` // For gzip: 1-9 (default 6), for zstd: 1-22 (default 3)
}

type Config struct {
	Image       string             `mapstructure:"image"`
	Build       *BuildConfig       `mapstructure:"build"`
	Registry    RegistryConfig     `mapstructure:"registry"`
	Compression *CompressionConfig `mapstructure:"compression"`
}

type RegistryConfig struct {
	Ghcr   *registry.GithubContainerRegistryConfig   `mapstructure:"ghcr"`
	AWSEcr *registry.AwsECRConfig                    `mapstructure:"aws_ecr"`
	GcpAr  *registry.GcpArtifactRegistryConfig       `mapstructure:"gcp_ar"`
}
