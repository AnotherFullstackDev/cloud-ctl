package registry

import (
	"fmt"
	"os"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/google/go-containerregistry/pkg/authn"
)

const (
	GHCRDomain              = "ghcr.io"
	usernameStorageKey      = "ghcr_username"
	usernameStorageLabel    = "Github Username"
	accessTokenStorageKey   = "ghcr_access_token"
	accessTokenStorageLabel = "Github Personal Access Token"
)

// GithubContainerRegistryConfig -  Github container registry destination config
type GithubContainerRegistryConfig string

type GithubContainerRegistry struct {
	storage         lib.CredentialsStorage
	config          GithubContainerRegistryConfig
	accessTokenEnvs []string
}

func NewGithubContainerRegistry(storage lib.CredentialsStorage, config GithubContainerRegistryConfig, accessTokenEnvs []string) Registry {
	return &GithubContainerRegistry{
		storage:         storage,
		config:          config,
		accessTokenEnvs: accessTokenEnvs,
	}
}

func (r *GithubContainerRegistry) GetAuthType() AuthType {
	return AuthTypeAuthenticator
}

func (r *GithubContainerRegistry) GetAuthentication() (authn.Authenticator, error) {
	// TODO: come up with a better way to store and get username - it could be already in the system and CLI can offer the user options to pick from (github cli, git config, etc)
	//
	// TODO: work on the mechanism of getting info from the environment
	// Instead of only getting extra information from ENV it is possible to extend this to a general interface and pass as a list of options to try in order to get the value from.
	// Terminal, keyring, env are possible options.
	username, err := lib.GetSecretFromEnvOrInput(r.storage, usernameStorageKey, usernameStorageLabel, nil, os.Stdin, os.Stdout, "Please provide Github Username for GHCR")
	if err != nil {
		return nil, fmt.Errorf("requesting ghrc username: %w", err)
	}

	// TODO: need to add a mechanism for access token invalidation in case the registry rejects the authentication
	authToken, err := lib.GetSecretFromEnvOrInput(r.storage, accessTokenStorageKey, accessTokenStorageLabel, r.accessTokenEnvs, os.Stdin, os.Stdout, "Please provide Github Personal Access Token (PAT)")
	if err != nil {
		return nil, fmt.Errorf("requesting ghrc access token: %w", err)
	}

	return authn.FromConfig(authn.AuthConfig{
		Username: username,
		Password: authToken,
	}), nil
}

func (r *GithubContainerRegistry) ResetAuthentication() error {
	if err := r.storage.Remove(usernameStorageKey); err != nil {
		return fmt.Errorf("resetting ghrc username: %w", err)
	}
	if err := r.storage.Remove(accessTokenStorageKey); err != nil {
		return fmt.Errorf("resetting ghrc access token: %w", err)
	}

	return nil
}

func (r *GithubContainerRegistry) GetKeychain() authn.Keychain {
	return nil
}

func (r *GithubContainerRegistry) GetImageRef() (string, error) {
	// Required format: ghcr.io/<owner>/<repository>:<tag>
	imageID := string(r.config)
	parts := strings.Split(imageID, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("%w - invalid ghcr image format: %s, expected format: ghcr.io/<owner>/<repository>:<tag>", lib.BadUserInputError, imageID)
	}
	if !strings.EqualFold(parts[0], GHCRDomain) {
		return "", fmt.Errorf("%w - invalid ghcr image format: %s, expected domain: %s", lib.BadUserInputError, imageID, GHCRDomain)
	}

	repositoryAndTag := strings.SplitN(parts[2], ":", 2)
	if len(repositoryAndTag) != 2 {
		return "", fmt.Errorf("%w - invalid ghcr image format: %s, missing tag", lib.BadUserInputError, imageID)
	}

	return imageID, nil
}
