package service

import (
	"fmt"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/factories"
	"github.com/spf13/cobra"
)

func newServiceDeployCmd(locator *factories.SharedServicesLocator) *cobra.Command {
	var serviceID, env string

	deployImageCmd := &cobra.Command{
		Use:   "deploy [name]",
		Short: "Deploy a service to the cloud provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceID == "" {
				return fmt.Errorf("service is required")
			}
			if env == "" {
				return fmt.Errorf("environment is required")
			}

			envSpecificConfig, err := locator.Config.WithEnvironment(env)
			if err != nil {
				return fmt.Errorf("loading environment specific config: %w", err)
			}

			serviceFactory := factories.NewServiceFactory(serviceID, locator.WithConfig(envSpecificConfig))

			serviceProvider, err := serviceFactory.NewCloudProvider()
			if err != nil {
				return fmt.Errorf("getting provider for service %s: %w", serviceID, err)
			}

			imageSvc, err := serviceFactory.NewImageService()
			if err != nil {
				return fmt.Errorf("getting image for service %s: %w", serviceID, err)
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

	deployImageCmd.PersistentFlags().StringVar(&serviceID, "name", "", "Service to deploy")
	deployImageCmd.PersistentFlags().StringVar(&env, "env", "", "Target environment")

	return deployImageCmd
}
