package registry

import "github.com/google/go-containerregistry/pkg/authn"

type Registry interface {
	GetAuthentication() (authn.Authenticator, error)
	GetImageRef() string
}
