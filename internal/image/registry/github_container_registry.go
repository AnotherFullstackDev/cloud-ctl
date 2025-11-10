package registry

import (
	"fmt"
	"os"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/google/go-containerregistry/pkg/authn"
)

const (
	GHCRDomain              = "ghcr.io"
	accessTokenStorageKey   = "ghcr_access_token"
	accessTokenStorageLabel = "GHCR Access Token"
)

// GithubContainerRegistryConfig -  Github container registry destination config
type GithubContainerRegistryConfig struct {
	Username   string `mapstructure:"username"`
	Owner      string `mapstructure:"owner"`
	Repository string `mapstructure:"repository"`
	Tag        string `mapstructure:"tag"`
}

type GithubContainerRegistry struct {
	storage         lib.CredentialsStorage
	config          GithubContainerRegistryConfig
	accessTokenEnvs []string
}

func NewGithubContainerRegistry(storage lib.CredentialsStorage, config GithubContainerRegistryConfig, accessTokenEnvs []string) *GithubContainerRegistry {
	return &GithubContainerRegistry{
		storage:         storage,
		config:          config,
		accessTokenEnvs: accessTokenEnvs,
	}
}

func (r *GithubContainerRegistry) GetAuthentication() (authn.Authenticator, error) {
	// TODO: need to add a mechanism for access token invalidation in case the registry rejects the authentication
	authToken, err := lib.GetSecretFromEnvOrInput(r.storage, accessTokenStorageKey, accessTokenStorageLabel, r.accessTokenEnvs, os.Stdin, os.Stdout, "Please provide Github Personal Access Token (PAT)")
	if err != nil {
		return nil, fmt.Errorf("requesting ghrc access token: %w", err)
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: r.config.Username,
		Password: authToken,
	}), nil
}

func (r *GithubContainerRegistry) GetImageRef() string {
	return fmt.Sprintf("%s/%s/%s:%s", GHCRDomain, r.config.Owner, r.config.Repository, r.config.Tag)
}
