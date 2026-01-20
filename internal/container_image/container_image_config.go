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

// CompressionAlgorithm represents the compression algorithm for image layers
type CompressionAlgorithm string

const (
	// CompressionGzip uses gzip compression (default, widely compatible)
	CompressionGzip CompressionAlgorithm = "gzip"
	// CompressionZstd uses zstd compression (better ratio, faster decompression)
	CompressionZstd CompressionAlgorithm = "zstd"
	// CompressionNone disables compression (fastest push, larger transfer size)
	CompressionNone CompressionAlgorithm = "none"
)

// CompressionConfig configures how image layers are compressed before push.
// Using zstd typically results in 20-30% smaller images and faster decompression.
type CompressionConfig struct {
	// Algorithm specifies the compression algorithm: "gzip" (default), "zstd", or "none"
	Algorithm CompressionAlgorithm `mapstructure:"algorithm"`
	// Level specifies compression level. For gzip: 1-9 (default 6), for zstd: 1-22 (default 3)
	Level int `mapstructure:"level"`
}

type Config struct {
	Image       string             `mapstructure:"image"`
	Build       *BuildConfig       `mapstructure:"build"`
	Registry    RegistryConfig     `mapstructure:"registry"`
	Compression *CompressionConfig `mapstructure:"compression"`
}

type RegistryConfig struct {
	Ghcr   *registry.GithubContainerRegistryConfig `mapstructure:"ghcr"`
	AWSEcr *registry.AwsECRConfig                  `mapstructure:"aws_ecr"`
}
