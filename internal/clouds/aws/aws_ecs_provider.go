package aws

import (
	"context"
	"fmt"
	"slices"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type EcsProvider struct {
	config EcsConfig
	ecs    *ecs.Client
}

func NewEcsProvider(config EcsConfig) *EcsProvider {
	ecsClient := ecs.New(ecs.Options{})
	return &EcsProvider{config, ecsClient}
}

func (p *EcsProvider) DeployServiceFromImage(ctx context.Context, registry clouds.ImageRegistry) error {
	// get service by arn
	// get default task definition
	// create new task definition revision with updated image
	// update service to use new task definition revision

	services, err := p.ecs.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Services: []string{p.config.ARN},
	})
	if err != nil {
		return fmt.Errorf("error describing ECS services: %s", err)
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

	newContainerDefs := make([]types.ContainerDefinition, len(taskDef.ContainerDefinitions))
	copy(newContainerDefs, taskDef.ContainerDefinitions)

	imageRef := registry.GetImageRef()
	if imageRef == "" {
		return fmt.Errorf("image reference is empty for service %s", p.config.ARN)
	}

	// TODO: container name must be configurable but optional
	containerName := "default"
	for i, containerDef := range newContainerDefs {
		if *containerDef.Name == containerName {
			newContainerDefs[i].Image = &imageRef
		}
	}

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
		Tags:                    []types.Tag{}, // TODO: fix this
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

	return nil
}
