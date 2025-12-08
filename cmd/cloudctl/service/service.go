package service

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/factories"
	"github.com/spf13/cobra"
)

func NewServiceCmd(locator *factories.SharedServicesLocator) *cobra.Command {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Deploy a service to the cloud provider",
	}

	serviceCmd.AddCommand(newServiceDeployCmd(locator))
	serviceCmd.AddCommand(newServiceBuildCmd(locator))

	return serviceCmd
}
