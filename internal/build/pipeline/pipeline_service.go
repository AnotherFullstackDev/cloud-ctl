package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"dagger.io/dagger"
	"dagger.io/dagger/dag"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
)

type Config struct {
	NodeVersion string       `mapstructure:"node_version"`
	PnpmVersion string       `mapstructure:"pnpm_version"`
	App         string       `mapstructure:"app"`
	Root        string       `mapstructure:"root"`
	ExtraFiles  []string     `mapstructure:"extra_files"`
	Steps       []Step       `mapstructure:"steps"`
	Platform    lib.Platform `mapstructure:"platform"`
	Cmd         []string     `mapstructure:"cmd"`
}

type Step struct {
	Task             string         `mapstructure:"task"`
	WorkingDirectory *string        `mapstructure:"working_directory"`
	Extra            map[string]any `mapstructure:",remain"`
}

type Task interface {
	GetRequiredSystemPackages() []string
	GetPostInstallCommands() [][]string
	GetCmd() ([][]string, error)
}

type Context struct {
	AppPath string
}

type Service struct {
	config       Config
	repoRoot     string
	monorepo     *PnpmMonorepo
	placeholders *placeholders.Service
}

func NewService(config Config, repoRoot string, monorepo *PnpmMonorepo, placeholders *placeholders.Service) *Service {
	return &Service{config, repoRoot, monorepo, placeholders}
}

func (s *Service) ProcessPipeline(ctx context.Context, outputImage string) error {
	l := slog.With("context", "pipeline_service")

	if s.config.App == "" {
		return fmt.Errorf("%w - no app specified in pipeline config", lib.BadUserInputError)
	}

	allowedPlatforms := map[lib.Platform]struct{}{
		lib.PlatformLinuxAmd64: {},
		lib.PlatformLinuxArm64: {},
	}
	if _, ok := allowedPlatforms[s.config.Platform]; !ok {
		supported := make([]string, 0, len(allowedPlatforms))
		for platform := range allowedPlatforms {
			supported = append(supported, string(platform))
		}
		return fmt.Errorf("%w - unsupported platform '%s' for pipeline builds, Supported are %s", lib.BadUserInputError, s.config.Platform, strings.Join(supported, ", "))
	}

	l.Info("building docker image from pipeline config",
		"app", s.config.App,
		"node_version", s.config.NodeVersion,
		"pnpm_version", s.config.PnpmVersion)

	workspacePackages, err := s.monorepo.GetWorkspacePackages()
	if err != nil {
		return fmt.Errorf("failed to get workspace packages: %w", err)
	}
	l.Debug("retrieved workspace packages", "packages", workspacePackages)

	appPackageIdx := slices.IndexFunc(workspacePackages, func(p WorkspacePackage) bool {
		return p.Manifest.Name == s.config.App
	})
	if appPackageIdx < 0 {
		return fmt.Errorf("%w - app package '%s' not found in monorepo workspace packages", lib.BadUserInputError, s.config.App)
	}

	appPackage := workspacePackages[appPackageIdx]
	l.Info("target workspace package found", "package", appPackage)

	dependencies := s.monorepo.GetPackageDependencies(appPackage, workspacePackages, PackageDependencyTypeDependencies, PackageDependencyTypeDevDependencies)
	l.Debug("app package dependencies", "dependencies", dependencies)

	baseImage := fmt.Sprintf("node:%s-alpine", s.config.NodeVersion)
	workdir := "/app"

	client, err := dagger.Connect(
		ctx,
		dagger.WithLogOutput(os.Stdout),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Dagger: %w", err)
	}
	defer client.Close()

	mandatoryFiles := []string{
		appPackage.MonorepoPath,
		"package.json",
		"pnpm-workspace.yaml",
		"tsconfig.json",
		".npmrc",
		".gitignore",
	}
	includePaths := make([]string, 0, len(mandatoryFiles)+len(dependencies)+len(s.config.ExtraFiles))
	for _, dep := range dependencies {
		includePaths = append(includePaths, dep.MonorepoPath)
	}
	includePaths = append(includePaths, mandatoryFiles...)
	includePaths = append(includePaths, s.config.ExtraFiles...)

	pipelineContext := Context{
		AppPath: appPackage.MonorepoPath,
	}

	var systemPackages []string
	var postInstallCommands [][]string
	var npmPackages []string
	stepTasks := make([]Task, 0, len(s.config.Steps))

	for _, step := range s.config.Steps {
		switch step.Task {
		case "grpc/generate/ts-proto":
			task := NewCompileProtobufToJsTask(step, pipelineContext, s.repoRoot, s.placeholders)
			systemPackages = append(systemPackages, task.GetRequiredSystemPackages()...)
			postInstallCommands = append(postInstallCommands, task.GetPostInstallCommands()...)
			npmPackages = append(npmPackages, task.GetRequiredNpmPackages()...)

			stepTasks = append(stepTasks, task)
		default:
			return fmt.Errorf("%w - unsupported pipeline task '%s'", lib.BadUserInputError, step.Task)
		}
	}

	c := dag.Container(dagger.ContainerOpts{Platform: dagger.Platform(s.config.Platform)}).
		From(baseImage).
		WithWorkdir(workdir).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", "$PNPM_HOME:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithExec(append([]string{"apk", "add", "--no-cache"}, systemPackages...))

	for _, cmd := range postInstallCommands {
		c = c.WithExec(cmd)
	}

	c = c.
		WithExec([]string{"corepack", "enable"}).
		WithExec([]string{"corepack", "prepare", fmt.Sprintf("pnpm@%s", s.config.PnpmVersion), "--activate"})

	c = c.WithExec(append([]string{"pnpm", "add", "-g"}, npmPackages...))

	c = c.
		WithDirectory(workdir, dag.Host().Directory(s.repoRoot), dagger.ContainerWithDirectoryOpts{
			Include: []string{"pnpm-lock.yaml"},
		}).
		WithExec([]string{"pnpm", "fetch"})

	// TODO: optimize this step by first copying only package.json & pnpm-workspace.yaml, installing deps, caching them, and only then copying the rest of the files
	c = c.
		WithDirectory(workdir, dag.Host().Directory(s.repoRoot), dagger.ContainerWithDirectoryOpts{
			Include: includePaths,
			Exclude: []string{
				"**/node_modules/**",
				"**/dist/**",
				"**/build/**",
				"**/out/**",
				"**/.next/**",
				"**/.cache/**",
				"**/.turbo/**",
			},
		}).
		WithExec([]string{"pnpm", "install", "--prefer-offline", "--frozen-lockfile"})

	for _, task := range stepTasks {
		cmds, err := task.GetCmd()
		if err != nil {
			return fmt.Errorf("getting command for pipeline task: %w", err)
		}

		for _, cmd := range cmds {
			c = c.WithExec(cmd)
		}
	}

	err = c.
		ExportImage(ctx, outputImage) // TODO: ensure images compression when exported

	if err != nil {
		return fmt.Errorf("setting up pipeline container: %w", err)
	}

	l.Info("docker image built successfully via pipeline", "image", outputImage)
	l.Info(fmt.Sprintf("run 'docker run --rm -it %s sh' to access the image", outputImage))

	return nil
}
