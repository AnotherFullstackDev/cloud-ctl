package lib

import "fmt"

const (
	EnvKeyPrefix = "CLOUDCTL"
)

var (
	GHCRAccessKeyEnv = fmt.Sprintf("%s_%s", EnvKeyPrefix, "GHCR_ACCESS_KEY")
	GithubTokenEnv   = "GITHUB_TOKEN"
)

var (
	RenderApiKeyEnv       = fmt.Sprintf("%s_%s", EnvKeyPrefix, "RENDER_API_KEY")
	RenderNativeApiKeyEnv = "RENDER_API_KEY"
)
