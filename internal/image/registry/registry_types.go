package registry

import "github.com/google/go-containerregistry/pkg/authn"

type CredentialsStorage interface {
	Set(key string, value string) error
	Get(key string) (string, error)
}

type Registry interface {
	GetAuthentication() (authn.Authenticator, error)
	GetImageRef() string
}
