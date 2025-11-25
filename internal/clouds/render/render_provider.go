package render

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	api2 "github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render/api"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render/api/deploys"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render/api/services"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

const (
	renderApiSecretKey   = "render_api_key"
	renderApiSecretLabel = "Render API Key"
)

type Provider struct {
	serviceID string
	config    Config
	api       *api2.Client
}

func MustNewProvider(serviceID string, cfg Config, storage lib.CredentialsStorage, authEnvKeys []string) *Provider {
	apiKey, err := lib.GetSecretFromEnvOrInput(storage, renderApiSecretKey, renderApiSecretLabel, authEnvKeys, os.Stdin, os.Stdout, "Please provide Render API Key")
	if err != nil {
		log.Fatalf("Error getting render_api_key: %v", err)
	}
	api := api2.MustNewClient("https://api.render.com/v1", apiKey)

	return &Provider{
		serviceID: serviceID,
		config:    cfg,
		api:       api,
	}
}

func (p *Provider) DeployServiceFromImage(ctx context.Context, registry clouds.ImageRegistry) error {
	imageRef, err := registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference for service %s: %w", p.config.ServiceID, err)
	}
	if imageRef == "" {
		return fmt.Errorf("image reference is empty for service %s", p.config.ServiceID)
	}

	service, err := p.api.RetrieveService(ctx, p.config.ServiceID)
	if err != nil {
		return fmt.Errorf("retrieving service %s: %w", p.config.ServiceID, err)
	}
	slog.DebugContext(ctx, "retrieved service details", "service_id", p.config.ServiceID, "service", service)

	if service.Type == services.ServiceTypeStaticSite {
		return fmt.Errorf("service %s is static site - not supported for container image deployment", p.config.ServiceID)
	}

	// TODO: add check for deployment credentials validity
	if service.ImagePath != imageRef {
		slog.InfoContext(ctx, "updating service image", "service_id", p.config.ServiceID, "from", service.ImagePath, "to", imageRef)

		err = p.api.UpdateService(ctx, p.config.ServiceID, services.UpdateServiceInput{
			Image: &services.UpdateServiceImage{
				OwnerID:              service.OwnerId,
				ImagePath:            imageRef,
				RegistryCredentialID: service.RegistryCredential.ID,
			},
		})
		if err != nil {
			return fmt.Errorf("updating service %s image from %s to %s: %w", p.config.ServiceID, service.ImagePath, imageRef, err)
		}
	}

	slog.InfoContext(ctx, "deploying image to service", "service_id", p.config.ServiceID, "image", imageRef)

	err = p.api.TriggerDeploy(ctx, p.config.ServiceID, deploys.TriggerDeployInput{ImageID: imageRef})
	if err != nil {
		return fmt.Errorf("deploying service %s: %w", p.config.ServiceID, err)
	}
	return nil
}
