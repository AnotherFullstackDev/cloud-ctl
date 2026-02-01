package lib

import "fmt"

const (
	RenderProviderKey       = "render"
	AwsEcsProviderKey       = "aws_ecs"
	AwsAppRunnerProviderKey = "aws_apprunner"
)

const (
	EnvKeyPrefix = "CLOUDCTL"
)

var (
	LogLevelEnv = fmt.Sprintf("%s_%s", EnvKeyPrefix, "LOG_LEVEL")
)

var (
	GHCRAccessKeyEnv = fmt.Sprintf("%s_%s", EnvKeyPrefix, "GHCR_ACCESS_KEY")
	GithubTokenEnv   = "GITHUB_TOKEN"
)

var (
	RenderApiKeyEnv       = fmt.Sprintf("%s_%s", EnvKeyPrefix, "RENDER_API_KEY")
	RenderNativeApiKeyEnv = "RENDER_API_KEY"
)

type Platform string

const (
	PlatformLinuxAmd64 Platform = "linux/amd64"
	PlatformLinuxArm64 Platform = "linux/arm64"
)
