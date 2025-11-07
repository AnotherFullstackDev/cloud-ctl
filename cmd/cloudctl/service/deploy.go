package service

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/spf13/cobra"
)

func newServiceDeployCmd(providers map[string]clouds.CloudProvider, imageConfigs map[string]image.Config, imageSvc *image.Service) *cobra.Command {
	deployImageCmd := &cobra.Command{
		Use:   "deploy [name]",
		Short: "Deploy a service to the cloud provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			availableServices := slices.Sorted(maps.Keys(imageConfigs))

			serviceID := args[0]
			serviceProvider, ok := providers[serviceID]
			if !ok {
				return fmt.Errorf("service %s is not found. Available services are: \n%s",
					serviceID, strings.Join(availableServices, "\n"))
			}

			imageConfig, ok := imageConfigs[serviceID]
			if !ok {
				return fmt.Errorf("image build config for service %s is not found. Available services are: \n%s",
					serviceID, strings.Join(availableServices, "\n"))
			}

			ctx := cmd.Context()

			if err := imageSvc.BuildImage(ctx, imageConfig.Build); err != nil {
				return fmt.Errorf("building image for service %s: %w", serviceID, err)
			}

			registryAuth, err := imageSvc.EnsureRegistryAuth(ctx, imageConfig)
			if err != nil {
				return fmt.Errorf("ensuring registry auth for service %s: %w", serviceID, err)
			}

			if err := imageSvc.PushImage(ctx, imageConfig, registryAuth); err != nil {
				return fmt.Errorf("pushing image for service %s: %w", serviceID, err)
			}

			return serviceProvider.DeployService(cmd.Context())
		},
	}

	return deployImageCmd
}
