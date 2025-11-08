package registry

import (
	"fmt"
	"os"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/google/go-containerregistry/pkg/authn"
)

const (
	GHCRDomain            = "ghcr.io"
	accessTokenStorageKey = "ghcr_access_token"
)

// GithubContainerRegistryConfig -  Github container registry destination config
type GithubContainerRegistryConfig struct {
	Username   string `mapstructure:"username"`
	Owner      string `mapstructure:"owner"`
	Repository string `mapstructure:"repository"`
	Tag        string `mapstructure:"tag"`
}

type GithubContainerRegistry struct {
	storage         CredentialsStorage
	config          GithubContainerRegistryConfig
	accessTokenEnvs []string
}

func NewGithubContainerRegistry(storage CredentialsStorage, config GithubContainerRegistryConfig, accessTokenEnvs []string) *GithubContainerRegistry {
	return &GithubContainerRegistry{
		storage:         storage,
		config:          config,
		accessTokenEnvs: accessTokenEnvs,
	}
}

func (r *GithubContainerRegistry) GetAuthentication() (authn.Authenticator, error) {
	authToken, err := r.storage.Get(accessTokenStorageKey)
	if err != nil {
		return nil, fmt.Errorf("retrieving ghcr access token from storage: %w", err)
	}

	if authToken == "" {
		for _, envKey := range r.accessTokenEnvs {
			authToken = strings.TrimSpace(os.Getenv(envKey))
			if authToken != "" {
				break
			}
		}
	}

	if authToken == "" {
		authToken, err = lib.RequestSecretInput(os.Stdin, os.Stdout, "Please provide Github Personal Access Token (PAT)")
		if err != nil {
			return nil, fmt.Errorf("requesting github token input: %w", err)
		}
	}

	if authToken == "" {
		return nil, fmt.Errorf("no github token provided for ghcr authentication")
	}

	if err := r.storage.Set(accessTokenStorageKey, authToken); err != nil {
		return nil, fmt.Errorf("storing ghcr access token: %w", err)
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: r.config.Username,
		Password: authToken,
	}), nil
}

func (r *GithubContainerRegistry) GetImageRef() string {
	return fmt.Sprintf("%s/%s/%s:%s", GHCRDomain, r.config.Owner, r.config.Repository, r.config.Tag)
}
