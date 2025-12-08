package service

import (
	"fmt"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/factories"
	"github.com/spf13/cobra"
)

func newServiceBuildCmd(locator *factories.SharedServicesLocator) *cobra.Command {
	var serviceID, env string

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build a service's container image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceID == "" {
				return fmt.Errorf("must provide a service ID")
			}
			if env == "" {
				return fmt.Errorf("must provide a environment name")
			}

			envSpecificConfig, err := locator.Config.WithEnvironment(env)
			if err != nil {
				return fmt.Errorf("loading environment specific config: %w", err)
			}

			serviceFactory := factories.NewServiceFactory(serviceID, locator.WithConfig(envSpecificConfig))

			imageSvc, err := serviceFactory.NewImageService()
			if err != nil {
				return fmt.Errorf("getting image for service %s: %w", serviceID, err)
			}

			ctx := cmd.Context()

			if err := imageSvc.BuildImage(ctx); err != nil {
				return fmt.Errorf("building image for service %s: %w", serviceID, err)
			}

			return nil
		},
	}

	buildCmd.PersistentFlags().StringVar(&serviceID, "name", "", "Service to build")
	buildCmd.PersistentFlags().StringVar(&env, "env", "", "Target environment")

	return buildCmd
}
