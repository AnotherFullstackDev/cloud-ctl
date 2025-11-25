package registry

import (
	"fmt"
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

func NewAwsECR(config AwsECRConfig) *AwsECR {
	return &AwsECR{config}
}

func (r *AwsECR) GetAuthType() AuthType {
	return AuthTypeKeychain
}

func (r *AwsECR) GetAuthentication() (authn.Authenticator, error) {
	return nil, nil
}

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

	registryURL := mainParts[0]
	registryParts := strings.Split(registryURL, ".")
	if len(registryParts) != 6 {
		return "", fmt.Errorf("%w - invalid registry URL: %s", lib.BadUserInputError, registryURL)
	}

	repositoryRef := registryParts[1]
	registryParts = strings.Split(repositoryRef, ":")
	if len(registryParts) != 2 {
		return "", fmt.Errorf("%w - invalid repository: %s", lib.BadUserInputError, repositoryRef)
	}

	return imageID, nil
}
