package aws

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apprunner"
	"github.com/aws/aws-sdk-go-v2/service/apprunner/types"
)

type AppRunnerProvider struct {
	config    AppRunnerConfig
	apprunner *apprunner.Client
}

func NewAppRunnerProvider(config AppRunnerConfig) (*AppRunnerProvider, error) {
	parsedARN, err := arn.Parse(config.ARN)
	if err != nil {
		return nil, fmt.Errorf("%w - parsing App Runner service ARN: %w", lib.BadUserInputError, err)
	}

	cfg, err := aws_config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	apprunnerClient := apprunner.NewFromConfig(cfg, func(o *apprunner.Options) {
		o.Region = parsedARN.Region
	})
	return &AppRunnerProvider{config, apprunnerClient}, nil
}

func (p *AppRunnerProvider) DeployServiceFromImage(ctx context.Context, registry clouds.ImageRegistry) error {
	imageRef, err := registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference for service %s: %w", p.config.ARN, err)
	}
	if imageRef == "" {
		return fmt.Errorf("image reference is empty for service %s", p.config.ARN)
	}

	slog.DebugContext(ctx, "fetching current App Runner service configuration",
		"arn", p.config.ARN)

	describeOutput, err := p.apprunner.DescribeService(ctx, &apprunner.DescribeServiceInput{
		ServiceArn: &p.config.ARN,
	})
	if err != nil {
		return fmt.Errorf("describing App Runner service: %w", err)
	}

	service := describeOutput.Service
	if service == nil {
		return fmt.Errorf("App Runner service with ARN %s not found", p.config.ARN)
	}

	slog.DebugContext(ctx, "retrieved App Runner service",
		"arn", p.config.ARN,
		"name", *service.ServiceName,
		"status", service.Status)

	// Preserve existing source configuration and only update the image identifier
	sourceConfig := service.SourceConfiguration
	if sourceConfig == nil || sourceConfig.ImageRepository == nil {
		return fmt.Errorf("%w - App Runner service %s is not configured with an image repository", lib.BadUserInputError, p.config.ARN)
	}

	currentImage := ""
	if sourceConfig.ImageRepository.ImageIdentifier != nil {
		currentImage = *sourceConfig.ImageRepository.ImageIdentifier
	}

	slog.InfoContext(ctx, "updating App Runner service image",
		"service", *service.ServiceName,
		"from", currentImage,
		"to", imageRef)

	// Build the updated source configuration preserving all existing settings
	updatedSourceConfig := &types.SourceConfiguration{
		AuthenticationConfiguration: sourceConfig.AuthenticationConfiguration,
		AutoDeploymentsEnabled:      sourceConfig.AutoDeploymentsEnabled,
		ImageRepository: &types.ImageRepository{
			ImageIdentifier:     &imageRef,
			ImageRepositoryType: sourceConfig.ImageRepository.ImageRepositoryType,
			ImageConfiguration:  sourceConfig.ImageRepository.ImageConfiguration,
		},
	}

	updateOutput, err := p.apprunner.UpdateService(ctx, &apprunner.UpdateServiceInput{
		ServiceArn:                  &p.config.ARN,
		AutoScalingConfigurationArn: service.AutoScalingConfigurationSummary.AutoScalingConfigurationArn,
		HealthCheckConfiguration:    service.HealthCheckConfiguration,
		InstanceConfiguration:       service.InstanceConfiguration,
		NetworkConfiguration:        service.NetworkConfiguration,
		ObservabilityConfiguration:  service.ObservabilityConfiguration,
		SourceConfiguration:         updatedSourceConfig,
	})
	if err != nil {
		return fmt.Errorf("updating App Runner service: %w", err)
	}

	operationID := ""
	if updateOutput.OperationId != nil {
		operationID = *updateOutput.OperationId
	}

	slog.InfoContext(ctx, "waiting for App Runner service deployment to complete",
		"service", *service.ServiceName,
		"operation_id", operationID)

	// Wait for the service to reach RUNNING status
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	waitTicker := time.NewTicker(10 * time.Second)
	defer waitTicker.Stop()

waiterLoop:
	for {
		describeOutput, err := p.apprunner.DescribeService(ctx, &apprunner.DescribeServiceInput{
			ServiceArn: &p.config.ARN,
		})
		if err != nil {
			return fmt.Errorf("describing App Runner service: %w", err)
		}

		if describeOutput.Service.Status == types.ServiceStatusRunning {
			break waiterLoop
		}
		if describeOutput.Service.Status != types.ServiceStatusOperationInProgress {
			return fmt.Errorf("App Runner service %s is %s - not expected", *describeOutput.Service.ServiceName, describeOutput.Service.Status)
		}

		select {
		case <-waitCtx.Done():
			return fmt.Errorf("waiting for App Runner service deployment to complete: %w", waitCtx.Err())
		case <-waitTicker.C:
		}
	}

	slog.InfoContext(ctx, "App Runner service deployment completed",
		"service", *service.ServiceName,
		"image", imageRef)

	return nil
}
