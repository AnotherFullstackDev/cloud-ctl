package lib

type KeyExtras struct {
	Label, Description string
}

type CredentialsStorage interface {
	Set(key string, value string, extra KeyExtras) error
	Get(key string) (string, error)
	Remove(key string) error
}
