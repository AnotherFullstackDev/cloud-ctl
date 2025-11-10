package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/cmd/cloudctl/service"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/clouds/render"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/image/registry"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/keyring"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "Cloudctl is a CLI tool for interacting with cloud resources.",
}

const (
	renderProviderKey = "render"
)

var logLevel = new(slog.LevelVar)

func main() {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(lib.LogLevelEnv))) {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "warning", "warn":
		logLevel.Set(slog.LevelWarn)
	case "error", "err":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	cfg, err := config.NewConfig("./cloudctl.yaml")
	if err != nil {
		log.Fatal(fmt.Errorf("error loading config: %w", err))
	}

	registryCredentialsStorage := keyring.MustNewService("container-registry")
	cloudApiCredentialsStorage := keyring.MustNewService("cloud-api-credentials")

	providerPerService := make(map[string]clouds.CloudProvider, len(cfg.Services))
	imagePerService := make(map[string]*image.Service, len(cfg.Services))
	for key, svc := range cfg.Services {
		imageConfig, err := config.LoadVariableServiceConfigPart[image.Config](cfg, key, "image")
		if err != nil {
			log.Fatal(fmt.Errorf("error loading image build config: %w", err))
		}

		var containerRegistry registry.Registry

		if imageConfig.Ghcr != nil {
			containerRegistry = registry.NewGithubContainerRegistry(registryCredentialsStorage, *imageConfig.Ghcr, []string{
				lib.GHCRAccessKeyEnv,
				lib.GithubTokenEnv,
			})
		}

		if containerRegistry == nil {
			log.Fatalf("no registry configured for image: %s:%s", imageConfig.Repository, imageConfig.Tag)
		}

		imagePerService[key] = image.MustNewService(imageConfig, containerRegistry)

		if _, ok := svc.Extras[renderProviderKey]; ok {
			renderCfg, err := config.LoadVariableServiceConfigPart[render.Config](cfg, key, renderProviderKey)
			if err != nil {
				log.Fatal(fmt.Errorf("error loading render config: %w", err))
			}

			providerPerService[key] = render.MustNewProvider(key, renderCfg, cloudApiCredentialsStorage, []string{
				lib.RenderApiKeyEnv,
				lib.RenderNativeApiKeyEnv,
			})
		}

		if _, ok := providerPerService[key]; !ok {
			log.Fatalf("service %s has no valid cloud provider configured", key)
		}
	}

	RootCmd.AddCommand(
		service.NewServiceCmd(providerPerService, imagePerService),
	)

	if err := RootCmd.Execute(); err != nil {
		log.Fatal(fmt.Errorf("error executing command: %w", err))
	}
}
