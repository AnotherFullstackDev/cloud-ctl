package service

import (
	"fmt"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/spf13/cobra"
)

func newServiceDeployCmd(providers map[string]clouds.CloudProvider, images map[string]*image.Service) *cobra.Command {
	deployImageCmd := &cobra.Command{
		Use:   "deploy [name]",
		Short: "Deploy a service to the cloud provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceID := args[0]
			serviceProvider, ok := providers[serviceID]
			if !ok {
				return fmt.Errorf("provider for service %s is not found", serviceID)
			}

			imageSvc, ok := images[serviceID]
			if !ok {
				return fmt.Errorf("image for service %s is not found", serviceID)
			}

			ctx := cmd.Context()

			if err := imageSvc.BuildImage(ctx); err != nil {
				return fmt.Errorf("building image for service %s: %w", serviceID, err)
			}

			if err := imageSvc.PushImage(ctx); err != nil {
				return fmt.Errorf("pushing image for service %s: %w", serviceID, err)
			}

			return serviceProvider.DeployServiceFromImage(cmd.Context(), imageSvc.GetRegistry())
		},
	}

	return deployImageCmd
}
