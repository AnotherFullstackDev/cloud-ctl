package container_image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/build/pipeline"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/container_image/registry"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"golang.org/x/term"
)

type Service struct {
	config               Config
	registry             registry.Registry
	placeholdersResolver *placeholders.Service
	pipelineService      *pipeline.Service
}

func NewService(config Config, registry registry.Registry, resolver *placeholders.Service, pipeline *pipeline.Service) *Service {
	return &Service{
		config,
		registry,
		resolver,
		pipeline,
	}
}

func (s *Service) GetRegistry() registry.Registry {
	return s.registry
}

func (s *Service) BuildImage(ctx context.Context) error {
	switch {
	case len(s.config.Build.Cmd) > 0:
		return s.buildImageViaCmd(ctx, s.config.Build.Cmd, s.config.Build.Env, s.config.Build.Dir)
	case s.config.Build.Pipeline != nil:
		return s.pipelineService.ProcessPipeline(ctx, s.config.Image)
	}

	return fmt.Errorf("no image build strategy configured")
}

func (s *Service) buildImageViaCmd(ctx context.Context, cmd []string, env map[string]string, dir string) error {
	if len(cmd) <= 0 {
		return fmt.Errorf("no command provided for image build")
	}

	resolvedCmd := make([]string, 0, len(cmd))
	for _, c := range cmd {
		resolvedC, err := s.placeholdersResolver.ResolvePlaceholders(c)
		if err != nil {
			return fmt.Errorf("resolving placeholder in build command '%s': %w", c, err)
		}
		resolvedCmd = append(resolvedCmd, resolvedC)
	}

	args := resolvedCmd
	if len(args) == 1 {
		args = []string{"sh", "-c", resolvedCmd[0]}
	}

	environment := os.Environ()
	for k, v := range env {
		resolvedValue, err := s.placeholdersResolver.ResolvePlaceholders(v)
		if err != nil {
			return fmt.Errorf("resolving placeholder in build env var '%s'='%s': %w", k, v, err)
		}

		environment = append(environment, fmt.Sprintf("%s=%s", k, resolvedValue))
	}

	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Env = environment
	command.Dir = dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	slog.InfoContext(ctx, "running image build command", "args", command.Args)

	if err := command.Run(); err != nil {
		return fmt.Errorf("running image build command: %w", err)
	}

	return nil
}

// TODO: add check for image architecture compatibility with target registry/platform
func (s *Service) PushImage(ctx context.Context) error {
	destRef, err := s.registry.GetImageRef()
	if err != nil {
		return fmt.Errorf("getting image reference from registry: %w", err)
	}
	if destRef == "" {
		return fmt.Errorf("container registry returned empty image reference")
	}

	resolvedImage, err := s.placeholdersResolver.ResolvePlaceholders(s.config.Image)
	srcRef, err := name.NewTag(resolvedImage)
	if err != nil {
		return fmt.Errorf("parsing source image tag: %w", err)
	}

	image, err := daemon.Image(srcRef, daemon.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("getting image from local daemon: %w", err)
	}

	destTag, err := name.NewTag(destRef)
	if err != nil {
		return fmt.Errorf("parsing destination image tag: %w", err)
	}

	auth, err := s.registry.GetAuthentication()
	if err != nil {
		return fmt.Errorf("getting registry authentication: %w", err)
	}

	var stdout io.Writer = os.Stdout
	stderr := os.Stderr
	tty := false

	if f, ok := stdout.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		tty = true
	}

	imageConfig, err := image.ConfigFile()
	if err != nil {
		return fmt.Errorf("getting image config file: %w", err)
	}

	slog.InfoContext(ctx, "pushing image to remote registry",
		"source", srcRef,
		"dest", destTag,
		"os", imageConfig.OS,
		"architecture", imageConfig.Architecture)

	startTime := time.Now()
	for {
		progressChan := make(chan v1.Update, 32)

		go func() {
			var lastUpdateTime time.Time
			for update := range progressChan {
				if !tty {
					continue
				}

				if update.Error != nil {
					fmt.Fprintf(stderr, "Error: %v\n", update.Error)
					continue
				}
				if update.Total <= 0 {
					continue
				}
				if time.Since(lastUpdateTime) <= 500*time.Millisecond {
					continue
				}
				lastUpdateTime = time.Now()

				percentage := float64(update.Complete) / float64(update.Total) * 100

				fmt.Fprintf(stdout, "Image push: %.2f%% complete\n", percentage)
			}
		}()

		maxUploadJobs := int(math.Min(16, float64(runtime.NumCPU())))
		options := []remote.Option{
			remote.WithContext(ctx),
			remote.WithAuth(auth),
			remote.WithProgress(progressChan),
			remote.WithJobs(maxUploadJobs),
			remote.WithPlatform(v1.Platform{
				Architecture: imageConfig.Architecture,
				OS:           imageConfig.OS,
				OSFeatures:   imageConfig.OSFeatures,
				OSVersion:    imageConfig.OSVersion,
				Variant:      imageConfig.Variant,
			}),
		}
		if err := remote.Write(destTag, image, options...); err != nil {
			var registryErr *transport.Error
			if errors.As(err, &registryErr) {
				isUnauthorizedErr := false
				if registryErr.StatusCode == http.StatusUnauthorized || registryErr.StatusCode == http.StatusForbidden {
					isUnauthorizedErr = true
				}
				for _, desc := range registryErr.Errors {
					if desc.Code == transport.UnauthorizedErrorCode || desc.Code == transport.DeniedErrorCode {
						isUnauthorizedErr = true
					}
				}
				if isUnauthorizedErr {
					slog.WarnContext(ctx, "unauthorized error pushing image to registry, resetting authentication and retrying", "error", err)

					err = s.registry.ResetAuthentication()
					if err != nil {
						return fmt.Errorf("resetting registry authentication after unauthorized error: %w", err)
					}
					auth, err = s.registry.GetAuthentication()
					if err != nil {
						return fmt.Errorf("getting registry authentication after reset: %w", err)
					}
					continue
				}
			}
			return fmt.Errorf("pushing image to remote registry: %w", err)
		}

		break
	}

	slog.InfoContext(ctx, "image pushed successfully",
		"source", srcRef,
		"destination", destRef,
		"duration", fmt.Sprintf("%f seconds", time.Since(startTime).Seconds()))

	return nil
}
