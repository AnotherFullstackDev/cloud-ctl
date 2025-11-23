package service

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/spf13/cobra"
)

func NewServiceCmd(config *config.Config, registryCredentialsStorage, cloudApiCredentialsStorage lib.CredentialsStorage) *cobra.Command {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Deploy a service to the cloud provider",
	}

	serviceCmd.AddCommand(newServiceDeployCmd(config, registryCredentialsStorage, cloudApiCredentialsStorage))

	return serviceCmd
}
