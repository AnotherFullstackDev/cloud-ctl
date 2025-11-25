package registry

import "github.com/google/go-containerregistry/pkg/authn"

type AuthType string

const (
	AuthTypeAuthenticator AuthType = "authenticator"
	AuthTypeKeychain      AuthType = "keychain"
)

// TODO: consider migrating everything to use keychain only,
// The approach with keychain seemd to be an industry standard as per go-containerregistry docs
type Registry interface {
	GetAuthType() AuthType
	GetKeychain() authn.Keychain
	GetAuthentication() (authn.Authenticator, error)
	GetImageRef() (string, error)
}
