package keyring

import (
	"errors"
	"fmt"
	"log"

	ring "github.com/99designs/keyring"
)

type Service struct {
	ring ring.Keyring
}

func MustNewService(name string) *Service {
	rind, err := ring.Open(ring.Config{
		ServiceName: name,
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

func (s *Service) Set(key, value string) error {
	err := s.ring.Set(ring.Item{
		Key:  key,
		Data: []byte(value),
	})
	if err != nil {
		return fmt.Errorf("setting key %q: %w", key, err)
	}
	return nil
}
