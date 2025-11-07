package main

import (
	"fmt"
	"log"

	"github.com/AnotherFullstackDev/cloud-ctl/cmd/cloudctl/service"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "Cloudctl is a CLI tool for interacting with cloud resources.",
}

const (
	renderProviderKey = "render"
)

func main() {
	cfg, err := config.NewConfig("./cloudctl.yaml")
	if err != nil {
		log.Fatal(fmt.Errorf("error loading config: %w", err))
	}

	providerPerService := map[string]clouds.CloudProvider{}
	imageConfigPerService := map[string]image.Config{}
	for key, svc := range cfg.Services {
		imageConfig, err := config.LoadVariableServiceConfigPart[image.Config](cfg, key, "image")
		if err != nil {
			log.Fatal(fmt.Errorf("error loading image build config: %w", err))
		}
		imageConfigPerService[key] = imageConfig

		if _, ok := svc.Extras[renderProviderKey]; ok {
			renderCfg, err := config.LoadVariableServiceConfigPart[render.Config](cfg, key, renderProviderKey)
			if err != nil {
				log.Fatal(fmt.Errorf("error loading render config: %w", err))
			}

			providerPerService[key] = render.MustNewProvider(key, renderCfg)
		}

		if _, ok := providerPerService[key]; !ok {
			log.Fatalf("service %s has no valid cloud provider configured", key)
		}
	}

	RootCmd.AddCommand(
		service.NewServiceCmd(providerPerService, imageConfigPerService, image.MustNewService()),
	)

	if err := RootCmd.Execute(); err != nil {
		log.Fatal(fmt.Errorf("error executing command: %w", err))
	}
}
