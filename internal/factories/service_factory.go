package factories

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/aws"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image/registry"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

type ServiceFactory struct {
	service                    string
	config                     *config.Config
	registryCredentialsStorage lib.CredentialsStorage
	cloudApiCredentialsStorage lib.CredentialsStorage
}

func NewServiceFactory(service string, config *config.Config, registryCredentialsStorage, cloudApiCredentialsStorage lib.CredentialsStorage) *ServiceFactory {
	return &ServiceFactory{
		service:                    service,
		config:                     config,
		registryCredentialsStorage: registryCredentialsStorage,
		cloudApiCredentialsStorage: cloudApiCredentialsStorage,
	}
}

func (f *ServiceFactory) NewImageService() (*image.Service, error) {
	var imageConfig image.Config
	if err := f.config.LoadVariableServiceConfigPart(&imageConfig, f.service, "image"); err != nil {
		return nil, fmt.Errorf("error loading image build config: %w", err)
	}

	var containerRegistry registry.Registry
	switch {
	case imageConfig.Ghcr != nil:
		containerRegistry = registry.NewGithubContainerRegistry(f.registryCredentialsStorage, *imageConfig.Ghcr, []string{
			lib.GHCRAccessKeyEnv,
			lib.GithubTokenEnv,
		})
	case imageConfig.AWSEcr != nil:
		containerRegistry = registry.NewAwsECR(*imageConfig.AWSEcr)
	default:
		log.Fatalf("no registry configured for image: %s:%s", imageConfig.Repository, imageConfig.Tag)
	}

	return image.NewService(imageConfig, containerRegistry), nil
}

func (f *ServiceFactory) NewCloudProvider() (clouds.CloudProvider, error) {
	svc, ok := f.config.Services[f.service]
	if !ok {
		return nil, fmt.Errorf("service %s not found in config", f.service)
	}

	var cloudProvider clouds.CloudProvider

	if _, ok := svc.Extras[lib.RenderProviderKey]; ok {
		slog.Info("loading Render provider for service", "service", f.service)

		var renderCfg render.Config
		if err := f.config.LoadVariableServiceConfigPart(&renderCfg, f.service, lib.RenderProviderKey); err != nil {
			return nil, fmt.Errorf("error loading render config: %w", err)
		}

		cloudProvider = render.MustNewProvider(f.service, renderCfg, f.cloudApiCredentialsStorage, []string{
			lib.RenderApiKeyEnv,
			lib.RenderNativeApiKeyEnv,
		})
	}

	if _, ok := svc.Extras[lib.AwsEcsProviderKey]; ok {
		slog.Info("loading AWS ECS provider for service", "service", f.service)

		var ecsCfg aws.EcsConfig
		if err := f.config.LoadVariableServiceConfigPart(&ecsCfg, f.service, lib.AwsEcsProviderKey); err != nil {
			return nil, fmt.Errorf("error loading AWS ECS config: %s", err)
		}

		cloudProvider = aws.NewEcsProvider(ecsCfg)
	}

	if cloudProvider == nil {
		return nil, fmt.Errorf("service %s has no valid cloud provider configured", f.service)
	}

	return cloudProvider, nil
}
