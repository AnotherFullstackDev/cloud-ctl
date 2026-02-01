package gcp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

type CloudRunProvider struct {
	config CloudRunConfig
	client *run.ServicesClient
}

func NewCloudRunProvider(config CloudRunConfig) (*CloudRunProvider, error) {
	if config.ServiceName == "" {
		return nil, fmt.Errorf("%w - Cloud Run service name is required", lib.BadUserInputError)
	}
	if config.ProjectID == "" {
		return nil, fmt.Errorf("%w - Cloud Run project ID is required", lib.BadUserInputError)
	}
	if config.Region == "" {
		return nil, fmt.Errorf("%w - Cloud Run region is required", lib.BadUserInputError)
	}

	// Create the Cloud Run services client using Application Default Credentials
	client, err := run.NewServicesClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("creating Cloud Run services client: %w", err)
	}

	return &CloudRunProvider{
		config: config,
		client: client,
	}, nil
}

func (p *CloudRunProvider) DeployServiceFromImage(ctx context.Context, registry clouds.ImageRegistry) error {
	imageRef, err := registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference for service %s: %w", p.config.ServiceName, err)
	}
	if imageRef == "" {
		return fmt.Errorf("image reference is empty for service %s", p.config.ServiceName)
	}

	// Build the full service name in the format: projects/{project}/locations/{location}/services/{service}
	serviceName := fmt.Sprintf("projects/%s/locations/%s/services/%s",
		p.config.ProjectID, p.config.Region, p.config.ServiceName)

	slog.DebugContext(ctx, "fetching current Cloud Run service configuration",
		"service", serviceName)

	// Get the existing service to preserve all configuration
	service, err := p.client.GetService(ctx, &runpb.GetServiceRequest{
		Name: serviceName,
	})
	if err != nil {
		return fmt.Errorf("getting Cloud Run service %s: %w", serviceName, err)
	}

	if service.Template == nil {
		return fmt.Errorf("%w - Cloud Run service %s has no template configured", lib.BadUserInputError, serviceName)
	}
	if len(service.Template.Containers) == 0 {
		return fmt.Errorf("%w - Cloud Run service %s has no containers configured", lib.BadUserInputError, serviceName)
	}

	currentImage := service.Template.Containers[0].Image

	slog.InfoContext(ctx, "updating Cloud Run service image",
		"service", p.config.ServiceName,
		"from", currentImage,
		"to", imageRef)

	// Update only the container image, preserving all other configuration
	// This is critical to not lose any existing settings
	service.Template.Containers[0].Image = imageRef

	slog.DebugContext(ctx, "sending update request to Cloud Run",
		"service", serviceName)

	// Update the service - this returns a long-running operation
	op, err := p.client.UpdateService(ctx, &runpb.UpdateServiceRequest{
		Service: service,
	})
	if err != nil {
		return fmt.Errorf("updating Cloud Run service %s: %w", serviceName, err)
	}

	slog.InfoContext(ctx, "waiting for Cloud Run service deployment to complete",
		"service", p.config.ServiceName)

	// Wait for the operation to complete with a timeout
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	updatedService, err := op.Wait(waitCtx)
	if err != nil {
		return fmt.Errorf("waiting for Cloud Run service deployment to complete: %w", err)
	}

	slog.InfoContext(ctx, "Cloud Run service deployment completed",
		"service", p.config.ServiceName,
		"image", imageRef,
		"uri", updatedService.Uri)

	return nil
}

// Close closes the Cloud Run client connection
func (p *CloudRunProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
