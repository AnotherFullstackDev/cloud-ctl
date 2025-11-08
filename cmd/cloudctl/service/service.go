package service

import (
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/spf13/cobra"
)

func NewServiceCmd(providers map[string]clouds.CloudProvider, images map[string]*image.Service) *cobra.Command {
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Deploy a service to the cloud provider",
	}

	serviceCmd.AddCommand(newServiceDeployCmd(providers, images))

	return serviceCmd
}
