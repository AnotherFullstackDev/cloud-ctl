package registry

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	ecrhelper "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	ecrapi "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/google/go-containerregistry/pkg/authn"
)

type AwsECRConfig string

type AwsECR struct {
	config AwsECRConfig
}

func NewAwsECR(config AwsECRConfig) Registry {
	return &AwsECR{config}
}

func (r *AwsECR) GetAuthType() AuthType {
	return AuthTypeKeychain
}

func (r *AwsECR) GetAuthentication() (authn.Authenticator, error) {
	return nil, nil
}

func (r *AwsECR) ResetAuthentication() error { return nil }

func (r *AwsECR) GetKeychain() authn.Keychain {
	helper := ecrhelper.NewECRHelper(ecrhelper.WithClientFactory(ecrapi.DefaultClientFactory{}))
	return authn.NewKeychainFromHelper(helper)
}

func (r *AwsECR) GetImageRef() (string, error) {
	// Required format: <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<repository>:<tag>
	imageID := string(r.config)
	mainParts := strings.Split(imageID, "/")
	if len(mainParts) != 2 {
		return "", fmt.Errorf("%w - invalid AWS ECR image format: %s", lib.BadUserInputError, imageID)
	}
	slog.Debug("split into main parts", "main_parts", mainParts)

	registryURL := mainParts[0]
	registryUrlParts := strings.Split(registryURL, ".")
	if len(registryUrlParts) != 6 {
		return "", fmt.Errorf("%w - invalid registry URL: %s", lib.BadUserInputError, registryURL)
	}
	slog.Debug("split into registry parts", "registry_parts", registryUrlParts)

	repositoryRef := mainParts[1]
	repositoryParts := strings.Split(repositoryRef, ":")
	if len(repositoryParts) != 2 {
		return "", fmt.Errorf("%w - invalid repository: %s", lib.BadUserInputError, repositoryRef)
	}
	slog.Debug("split into repository parts", "repository_parts", repositoryParts)

	return imageID, nil
}
