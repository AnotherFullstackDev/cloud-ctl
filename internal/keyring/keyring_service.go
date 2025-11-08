package keyring

import (
	"errors"
	"fmt"
	"log"

	ring "github.com/99designs/keyring"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

type Service struct {
	ring ring.Keyring
}

func MustNewService(name string) *Service {
	rind, err := ring.Open(ring.Config{
		ServiceName:  name,
		KeychainName: "login",
		AllowedBackends: []ring.BackendType{
			ring.SecretServiceBackend,
			ring.KeychainBackend,
			ring.WinCredBackend,
			ring.KeyCtlBackend,
			ring.KWalletBackend,
			ring.PassBackend,
		},
	})
	if err != nil {
		log.Fatalf("creating keyring: %s", err)
	}

	return &Service{
		ring: rind,
	}
}

func (s *Service) Get(key string) (string, error) {
	value, err := s.ring.Get(key)
	if errors.Is(err, ring.ErrKeyNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting key %q: %w", key, err)
	}
	return string(value.Data), nil
}

// Set sets a key-value pair in the keyring.
// key: The key to set.
// value: The value to associate with the key.
// label: An optional label for the key. Might be shown to the user in the system prompt when the item is accessed.
// description: An optional description for the key.
func (s *Service) Set(key, value string, extra lib.KeyExtras) error {
	item := ring.Item{
		Key:  key,
		Data: []byte(value),
	}
	if extra.Label != "" {
		item.Label = extra.Label
	}
	if extra.Description != "" {
		item.Description = extra.Description
	}
	err := s.ring.Set(item)
	if err != nil {
		return fmt.Errorf("setting key %q: %w", key, err)
	}
	return nil
}
