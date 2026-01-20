package aws

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type EcsProvider struct {
	config EcsConfig
	ecs    *ecs.Client
}

func NewEcsProvider(config EcsConfig) (*EcsProvider, error) {
	parsedARN, err := arn.Parse(config.ARN)
	if err != nil {
		return nil, fmt.Errorf("%w - parsing ECS service ARN: %w", lib.BadUserInputError, err)
	}

	cfg, err := aws_config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	ecsClient := ecs.NewFromConfig(cfg, func(o *ecs.Options) {
		o.Region = parsedARN.Region
	})
	return &EcsProvider{config, ecsClient}, nil
}

func (p *EcsProvider) DeployServiceFromImage(ctx context.Context, registry clouds.ImageRegistry) error {
	// get service by arn
	// get default task definition
	// create new task definition revision with updated image
	// update service to use new task definition revision
	serviceArn, err := arn.Parse(p.config.ARN)
	if err != nil {
		return fmt.Errorf("parsing ECS service ARN: %w", err)
	}
	serviceResourceParts := strings.Split(serviceArn.Resource, "/")
	if len(serviceResourceParts) != 3 {
		slog.ErrorContext(ctx, "invalid ECS service ARN",
			"arn", p.config.ARN,
			"resource", serviceArn.Resource)
		return fmt.Errorf("%w - invalid ECS service ARN: %s", lib.BadUserInputError, p.config.ARN)
	}
	slog.DebugContext(ctx, "parsed ECS service ARN",
		"arn", p.config.ARN,
		"cluster", serviceResourceParts[1],
		"service", serviceResourceParts[2])

	cluster := serviceResourceParts[1]
	serviceName := serviceResourceParts[2]
	services, err := p.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Services: []string{serviceName},
		Cluster:  &cluster,
	})
	if err != nil {
		return fmt.Errorf("describing ECS services: %s", err)
	}
	serviceIdx := slices.IndexFunc(services.Services, func(s types.Service) bool {
		return *s.ServiceArn == p.config.ARN
	})
	if serviceIdx == -1 {
		return fmt.Errorf("ECS service with ARN %s not found", p.config.ARN)
	}
	service := services.Services[serviceIdx]

	taskDefOutput, err := p.ecs.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: service.TaskDefinition,
		Include:        []types.TaskDefinitionField{types.TaskDefinitionFieldTags},
	})
	if err != nil {
		return fmt.Errorf("error describing ECS task definition: %s", err)
	}
	taskDef := taskDefOutput.TaskDefinition

	newContainerDefs := make([]types.ContainerDefinition, 0, len(taskDef.ContainerDefinitions))
	newContainerDefs = append(newContainerDefs, taskDef.ContainerDefinitions...)

	imageRef, err := registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference for service %s: %w", p.config.ARN, err)
	}
	if imageRef == "" {
		return fmt.Errorf("image reference is empty for service %s", p.config.ARN)
	}

	containerName := "default"
	if p.config.ContainerName != nil {
		containerName = *p.config.ContainerName
	}

	containerIdx := slices.IndexFunc(newContainerDefs, func(c types.ContainerDefinition) bool {
		return *c.Name == containerName
	})
	if containerIdx < 0 {
		return fmt.Errorf("%w - container %s not found in task definition", lib.BadUserInputError, containerName)
	}
	newContainerDefs[containerIdx].Image = &imageRef

	newTaskDefInput := taskDef
	newTaskDefInput.ContainerDefinitions = newContainerDefs

	// TODO: creation of new task definition must be optional
	registerOutput, err := p.ecs.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions:    newTaskDefInput.ContainerDefinitions,
		Family:                  newTaskDefInput.Family,
		Cpu:                     newTaskDefInput.Cpu,
		EnableFaultInjection:    newTaskDefInput.EnableFaultInjection,
		EphemeralStorage:        newTaskDefInput.EphemeralStorage,
		ExecutionRoleArn:        newTaskDefInput.ExecutionRoleArn,
		InferenceAccelerators:   newTaskDefInput.InferenceAccelerators,
		IpcMode:                 newTaskDefInput.IpcMode,
		Memory:                  newTaskDefInput.Memory,
		NetworkMode:             newTaskDefInput.NetworkMode,
		PidMode:                 newTaskDefInput.PidMode,
		PlacementConstraints:    newTaskDefInput.PlacementConstraints,
		ProxyConfiguration:      newTaskDefInput.ProxyConfiguration,
		RequiresCompatibilities: newTaskDefInput.RequiresCompatibilities,
		RuntimePlatform:         newTaskDefInput.RuntimePlatform,
		Tags:                    taskDefOutput.Tags,
		TaskRoleArn:             newTaskDefInput.TaskRoleArn,
		Volumes:                 newTaskDefInput.Volumes,
	})
	if err != nil {
		return fmt.Errorf("error registering new ECS task definition: %s", err)
	}

	_, err = p.ecs.UpdateService(ctx, &ecs.UpdateServiceInput{
		Service:            service.ServiceName,
		Cluster:            service.ClusterArn,
		TaskDefinition:     registerOutput.TaskDefinition.TaskDefinitionArn,
		ForceNewDeployment: true,
	})
	if err != nil {
		return fmt.Errorf("error updating ECS service to new task definition: %s", err)
	}

	slog.InfoContext(ctx, "waiting for ECS service to be stable",
		"service", *service.ServiceName,
		"cluster", *service.ClusterArn)

	waiter := ecs.NewServicesStableWaiter(p.ecs)
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	waitInput := &ecs.DescribeServicesInput{
		Cluster:  service.ClusterArn,
		Services: []string{*service.ServiceName},
	}
	maxWaitDuration := 10 * time.Minute
	extraOptions := func(o *ecs.ServicesStableWaiterOptions) {
		o.MinDelay = 10 * time.Second
		o.MaxDelay = 30 * time.Second
	}
	err = waiter.Wait(waitCtx, waitInput, maxWaitDuration, extraOptions)
	if err != nil {
		return fmt.Errorf("waiting for ECS service to be stable: %w", err)
	}

	slog.InfoContext(ctx, "ECS service is stable",
		"service", *service.ServiceName,
		"cluster", *service.ClusterArn)

	return nil
}
