package image

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"golang.org/x/term"
)

type Service struct{}

func MustNewService() *Service {
	return &Service{}
}

func (s *Service) BuildImage(ctx context.Context, input BuildConfig) error {
	if input.Cmd != nil {
		return s.buildImageViaCmd(ctx, input.Cmd, input.Env, input.Dir)
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

	if err := command.Run(); err != nil {
		return fmt.Errorf("running image build command: %w", err)
	}

	return nil
}

func requestSecretInput(in io.Reader, out io.Writer, prompt string) (string, error) {
	_, err := fmt.Fprintf(out, "%s: ", prompt)
	if err != nil {
		return "", fmt.Errorf("writing prompt: %w", err)
	}

	defer fmt.Fprintf(out, "secret received\n")

	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		secret, err := term.ReadPassword(int(f.Fd()))
		if err != nil {
			return "", fmt.Errorf("reading secret input: %w", err)
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return "", fmt.Errorf("writing newline after secret input: %w", err)
		}

		return strings.TrimSpace(string(secret)), nil
	}

	// When not a terminal, fall back to normal input reading
	reader := bufio.NewReader(in)
	secret, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading secret input: %w", err)
	}

	return strings.TrimSpace(secret), nil
}

func (s *Service) EnsureRegistryAuth(ctx context.Context, input Config) (authn.Authenticator, error) {
	if input.Ghcr != nil {
		// TODO: move to a better place - improve readability of the codebase
		authToken := os.Getenv("CLOUDCTL_GHCR_TOKEN")
		if authToken == "" {
			authToken = os.Getenv("GITHUB_TOKEN")
		}
		if authToken == "" {
			secret, err := requestSecretInput(os.Stdin, os.Stdout, "Please provide Github Personal Access Token (PAT)")
			if err != nil {
				return nil, fmt.Errorf("requesting github token input: %w", err)
			}
			authToken = secret
		}
		if authToken == "" {
			return nil, fmt.Errorf("no github token provided for ghcr authentication")
		}

		return authn.FromConfig(authn.AuthConfig{
			Username: input.Ghcr.Username,
			Password: authToken,
		}), nil
	}

	return nil, fmt.Errorf("no registry authentication strategy configured")
}

func (s *Service) PushImage(ctx context.Context, input Config, auth authn.Authenticator) error {
	var destRef string
	if input.Ghcr != nil {
		destRef = fmt.Sprintf("ghcr.io/%s/%s:%s", input.Ghcr.Owner, input.Ghcr.Repository, input.Ghcr.Tag)
	}

	if destRef == "" {
		return fmt.Errorf("no destination container registry configured")
	}

	srcRef, err := name.NewTag(fmt.Sprintf("%s:%s", input.Repository, input.Tag))
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

	startTime := time.Now()
	maxUploadJobs := int(math.Min(16, float64(runtime.NumCPU())))
	options := []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuth(auth),
		remote.WithProgress(progressChan),
		remote.WithJobs(maxUploadJobs),
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
