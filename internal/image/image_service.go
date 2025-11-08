package image

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/image/registry"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/term"
)

type Service struct {
	config   Config
	registry registry.Registry
}

func MustNewService(config Config, registry registry.Registry) *Service {
	return &Service{
		config:   config,
		registry: registry,
	}
}

func (s *Service) BuildImage(ctx context.Context) error {
	if s.config.Build.Cmd != nil {
		return s.buildImageViaCmd(ctx, s.config.Build.Cmd, s.config.Build.Env, s.config.Build.Dir)
	}

	return fmt.Errorf("no image build strategy configured")
}

func (s *Service) buildImageViaCmd(ctx context.Context, cmd []string, env map[string]string, dir string) error {
	if len(cmd) <= 0 {
		return fmt.Errorf("no command provided for image build")
	}

	args := cmd
	if len(args) == 1 {
		args = []string{"sh", "-c", cmd[0]}
	}

	environment := os.Environ()
	for k, v := range env {
		environment = append(environment, fmt.Sprintf("%s=%s", k, v))
	}

	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Env = environment
	command.Dir = dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	slog.Info("running image build command", "args", command.Args)

	if err := command.Run(); err != nil {
		return fmt.Errorf("running image build command: %w", err)
	}

	return nil
}

func (s *Service) PushImage(ctx context.Context) error {
	auth, err := s.registry.GetAuthentication()
	if err != nil {
		return fmt.Errorf("getting registry authentication: %w", err)
	}

	destRef := s.registry.GetImageRef()
	if destRef == "" {
		return fmt.Errorf("container registry returned empty image reference")
	}

	srcRef, err := name.NewTag(fmt.Sprintf("%s:%s", s.config.Repository, s.config.Tag))
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

	var stdout io.Writer = os.Stdout
	stderr := os.Stderr
	tty := false
	progressChan := make(chan v1.Update, 32)

	if f, ok := stdout.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		tty = true
	}

	go func() {
		var lastUpdateTime time.Time
		for update := range progressChan {
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

			if tty {
				fmt.Fprintf(stdout, "Image push: %.2f%% complete\n", percentage)
			}
		}
	}()

	imageConfig, err := image.ConfigFile()
	if err != nil {
		return fmt.Errorf("getting image config file: %w", err)
	}

	slog.Info("pushing image to remote registry",
		"source", srcRef,
		"os", imageConfig.OS,
		"architecture", imageConfig.Architecture,
		"os_features", imageConfig.OSFeatures,
		"os_version", imageConfig.OSVersion,
		"variant", imageConfig.Variant)

	startTime := time.Now()
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
		return fmt.Errorf("pushing image to remote registry: %w", err)
	}

	slog.Info("image pushed successfully",
		"source", srcRef,
		"destination", destRef,
		"duration", fmt.Sprintf("%f seconds", time.Since(startTime).Seconds()))

	return nil
}
