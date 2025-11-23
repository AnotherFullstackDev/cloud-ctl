package registry

import (
	"fmt"

	ecrhelper "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	ecrapi "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/google/go-containerregistry/pkg/authn"
)

type AwsECRConfig struct {
	Region     string `mapstructure:"region"`
	AccountID  string `mapstructure:"account_id"`
	Repository string `mapstructure:"repository"`
	Tag        string `mapstructure:"tag"`
}

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

func (r *AwsECR) GetImageRef() string {
	// Example: <aws_account_id>.dkr.ecr.<region>.amazonaws.com/<repository>:<tag>
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", r.config.AccountID, r.config.Region, r.config.Repository, r.config.Tag)
}
