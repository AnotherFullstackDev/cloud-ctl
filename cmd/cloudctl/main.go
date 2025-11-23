package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/cmd/cloudctl/service"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/config"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/keyring"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "Cloudctl is a CLI tool for interacting with cloud resources.",
}

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

	cfg, err := config.NewConfigFromPath("./cloudctl.yaml")
	if err != nil {
		log.Fatal(fmt.Errorf("error loading config: %w", err))
	}

	registryCredentialsStorage := keyring.MustNewService("container-registry")
	cloudApiCredentialsStorage := keyring.MustNewService("cloud-api-credentials")

	RootCmd.AddCommand(
		service.NewServiceCmd(cfg, registryCredentialsStorage, cloudApiCredentialsStorage),
	)

	if err := RootCmd.Execute(); err != nil {
		log.Fatal(fmt.Errorf("error executing command: %w", err))
	}
}
