package factories

import (
	"fmt"
	"log"
	"log/slog"
	"path/filepath"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/build/pipeline"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/aws"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image/registry"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
)

type ServiceFactory struct {
	service                    string
	config                     *config.Config
	registryCredentialsStorage lib.CredentialsStorage
	cloudApiCredentialsStorage lib.CredentialsStorage
	placeholdersService        *placeholders.Service
}

func NewServiceFactory(service string, executionCtx *SharedServicesLocator) *ServiceFactory {
	return &ServiceFactory{
		service:                    service,
		config:                     executionCtx.Config,
		registryCredentialsStorage: executionCtx.RegistryCredentialsStorage,
		cloudApiCredentialsStorage: executionCtx.CloudApiCredentialsStorage,
		placeholdersService:        executionCtx.PlaceholdersService,
	}
}

func (f *ServiceFactory) NewImageService() (*container_image.Service, error) {
	var imageConfig container_image.Config
	if err := f.config.LoadVariableServiceConfigPart(&imageConfig, f.service, "container"); err != nil {
		return nil, fmt.Errorf("error loading image build config: %w", err)
	}

	var containerRegistry registry.Registry
	switch {
	case imageConfig.Registry.Ghcr != nil:
		resolvedGhcr, err := f.placeholdersService.ResolvePlaceholders(string(*imageConfig.Registry.Ghcr))
		if err != nil {
			return nil, fmt.Errorf("resolving GHCR registry placeholder: %w", err)
		}

		containerRegistry = registry.NewGithubContainerRegistry(f.registryCredentialsStorage, registry.GithubContainerRegistryConfig(resolvedGhcr), []string{
			lib.GHCRAccessKeyEnv,
			lib.GithubTokenEnv,
		})
	case imageConfig.Registry.AWSEcr != nil:
		resolvedEcr, err := f.placeholdersService.ResolvePlaceholders(string(*imageConfig.Registry.AWSEcr))
		if err != nil {
			return nil, fmt.Errorf("resolving AWS ECR registry placeholder: %w", err)
		}

		containerRegistry = registry.NewAwsECR(registry.AwsECRConfig(resolvedEcr))
	default:
		log.Fatalf("no registry configured for image: %s", imageConfig.Image)
	}

	pipelineConfig := imageConfig.Build.Pipeline
	if pipelineConfig == nil {
		pipelineConfig = &pipeline.Config{}
	}
	repoRoot := pipelineConfig.Root
	if repoRoot == "" {
		return nil, fmt.Errorf("no repository root provided")
	}
	repoRoot = filepath.Clean(repoRoot)
	monorepoProvider := pipeline.NewPnpmMonorepo(repoRoot)
	pipelineService := pipeline.NewService(*pipelineConfig, repoRoot, monorepoProvider, f.placeholdersService)

	return container_image.NewService(imageConfig, containerRegistry, f.placeholdersService, pipelineService), nil
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
