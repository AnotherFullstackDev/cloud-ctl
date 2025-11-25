package factories

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
)

type SharedServicesLocator struct {
	Config                                                 *config.Config
	RegistryCredentialsStorage, CloudApiCredentialsStorage lib.CredentialsStorage
	PlaceholdersService                                    *placeholders.Service
}

func NewSharedServicesLocator(config *config.Config, registryCredentialsStorage, cloudApiCredentialsStorage lib.CredentialsStorage, placeholders *placeholders.Service) *SharedServicesLocator {
	return &SharedServicesLocator{
		config,
		registryCredentialsStorage,
		cloudApiCredentialsStorage,
		placeholders,
	}
}

func (l *SharedServicesLocator) WithConfig(config *config.Config) *SharedServicesLocator {
	return &SharedServicesLocator{
		config,
		l.RegistryCredentialsStorage,
		l.CloudApiCredentialsStorage,
		l.PlaceholdersService,
	}
}
