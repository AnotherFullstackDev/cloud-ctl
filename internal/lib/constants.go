package lib

import "fmt"

const (
	EnvKeyPrefix = "CLOUDCTL"
)

var (
	GHCRAccessKeyEnv = fmt.Sprintf("%s_%s", EnvKeyPrefix, "GHCR_ACCESS_KEY")
	GithubTokenEnv   = "GITHUB_TOKEN"
)
