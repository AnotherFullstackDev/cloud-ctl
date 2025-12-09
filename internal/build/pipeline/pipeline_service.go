package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"dagger.io/dagger"
	"dagger.io/dagger/dag"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
)

type Config struct {
	NodeVersion  string       `mapstructure:"node_version"`
	PnpmVersion  string       `mapstructure:"pnpm_version"`
	App          string       `mapstructure:"app"`
	Root         string       `mapstructure:"root"`
	ExtraFiles   []string     `mapstructure:"extra_files"`
	ExcludeFiles []string     `mapstructure:"exclude_files"`
	Steps        []Step       `mapstructure:"steps"`
	RuntimeSteps []Step       `mapstructure:"runtime_steps"`
	Platform     lib.Platform `mapstructure:"platform"`
	Cmd          []string     `mapstructure:"cmd"`
}

type Step struct {
	Task             TaskID         `mapstructure:"task"`
	WorkingDirectory *string        `mapstructure:"working_directory"`
	Extra            map[string]any `mapstructure:",remain"`
}

type processStepsResult struct {
	SystemPackages  []string
	PostInstallCmds [][]string
	NpmPackages     []string
	Tasks           []Task
}

type Task interface {
	GetTaskID() TaskID
	GetRequiredSystemPackages() []string
	GetPostInstallCommands() ([][]string, error)
	GetRequiredNpmPackages() []string
	GetCmd() ([][]string, error)
}

type PlaceholderResolvers map[string]placeholders.PlaceholderResolver

type Service struct {
	config       Config
	repoRoot     string
	monorepo     *PnpmMonorepo
	placeholders *placeholders.Service
}

type TaskID string

const (
	TaskIDGrpcGenerateTsProto TaskID = "grpc/generate/ts-proto"
	TaskIDSetupPnpm           TaskID = "setup/pnpm"
	TaskIDSetupBun            TaskID = "setup/bun"
	TaskIDCli                 TaskID = "cli"
)

func NewService(config Config, repoRoot string, monorepo *PnpmMonorepo, placeholders *placeholders.Service) *Service {
	return &Service{config, repoRoot, monorepo, placeholders}
}

func (s *Service) ProcessPipeline(ctx context.Context, outputImage string) error {
	l := slog.With("context", "pipeline_service")

	if s.config.App == "" {
		return fmt.Errorf("%w - no app specified in pipeline config", lib.BadUserInputError)
	}

	platform := s.config.Platform
	if platform == "" {
		platform = lib.PlatformLinuxAmd64
	}
	allowedPlatforms := map[lib.Platform]struct{}{
		lib.PlatformLinuxAmd64: {},
		lib.PlatformLinuxArm64: {},
	}
	if _, ok := allowedPlatforms[platform]; !ok {
		supported := make([]string, 0, len(allowedPlatforms))
		for platform := range allowedPlatforms {
			supported = append(supported, string(platform))
		}
		return fmt.Errorf("%w - unsupported platform '%s' for pipeline builds, Supported are %s", lib.BadUserInputError, platform, strings.Join(supported, ", "))
	}

	l.Info("building docker image from pipeline config",
		"app", s.config.App,
		"node_version", s.config.NodeVersion,
		"pnpm_version", s.config.PnpmVersion,
		"platform", platform,
		"cmd", s.config.Cmd)

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

	pipelinePlaceholderResolvers := PlaceholderResolvers{
		"app.dir": func() (string, error) {
			return appPackage.Path, nil
		},
		"app.package": func() (string, error) {
			return appPackage.Manifest.Name, nil
		},
	}

	cmd := s.config.Cmd
	if cmd == nil || len(cmd) == 0 {
		return fmt.Errorf("%w - no 'cmd' specified for pipeline build", lib.BadUserInputError)
	}
	for i, cmdPart := range cmd {
		cmdPartResolved, err := s.placeholders.ResolvePlaceholders(cmdPart, pipelinePlaceholderResolvers)
		if err != nil {
			return fmt.Errorf("failed to resolve placeholders for command '%s': %w", cmdPart, err)
		}
		cmd[i] = cmdPartResolved
	}
	l.Info("resolved cmd", "cmd", cmd)

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

	filesForPackageInstallation := []string{
		appPackage.ManifestPath,
		"package.json",
		"pnpm-lock.yaml",
		"pnpm-workspace.yaml",
		".npmrc",
		".gitignore",
	}
	for _, dep := range dependencies {
		filesForPackageInstallation = append(filesForPackageInstallation, dep.ManifestPath)
	}
	for _, f := range filesForPackageInstallation {
		stat, err := os.Stat(filepath.Join(s.repoRoot, f))
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat package installation path: %w", err)
		}
		if stat != nil && stat.IsDir() {
			return fmt.Errorf("package installation path '%s' is a directory, expected a file", f)
		}
	}
	l.Info("package installation files", "files", filesForPackageInstallation)

	mandatoryFiles := []string{
		appPackage.Path,
		"package.json",
		"pnpm-workspace.yaml",
		"pnpm-lock.yaml",
		"tsconfig.json",
		".npmrc",
		".gitignore",
	}
	includePaths := make([]string, 0, len(mandatoryFiles)+len(dependencies)+len(s.config.ExtraFiles))
	for _, dep := range dependencies {
		includePaths = append(includePaths, dep.Path)
	}
	includePaths = append(includePaths, mandatoryFiles...)
	includePaths = append(includePaths, s.config.ExtraFiles...)
	l.Info("production build files", "paths", includePaths)

	stepResults, err := s.processSteps(s.config.Steps, pipelinePlaceholderResolvers)
	if err != nil {
		return fmt.Errorf("processing pipeline steps: %w", err)
	}

	pnpmCacheVolume := client.CacheVolume(fmt.Sprintf("pnpm-cache-%s", s.config.PnpmVersion))

	builder := dag.Container(dagger.ContainerOpts{Platform: dagger.Platform(platform)}).
		From(baseImage).
		WithWorkdir(workdir).
		WithEnvVariable("PNPM_HOME", "/pnpm").
		WithEnvVariable("PATH", "$PNPM_HOME:$PATH", dagger.ContainerWithEnvVariableOpts{Expand: true}).
		WithMountedCache("/pnpm/store", pnpmCacheVolume).
		WithExec(append([]string{"apk", "add", "--no-cache"}, stepResults.SystemPackages...))

	for _, cmd := range stepResults.PostInstallCmds {
		builder = builder.WithExec(cmd)
	}

	builder = builder.
		WithExec([]string{"corepack", "enable"}).
		WithExec([]string{"corepack", "prepare", fmt.Sprintf("pnpm@%s", s.config.PnpmVersion), "--activate"})

	builder = builder.WithExec(append([]string{"pnpm", "add", "-g"}, stepResults.NpmPackages...))

	builder = builder.
		WithDirectory(workdir, dag.Host().Directory(s.repoRoot), dagger.ContainerWithDirectoryOpts{
			Include: []string{"pnpm-lock.yaml"},
		}).
		WithExec([]string{"pnpm", "fetch"})

	hostRepoRootDir := dag.Host().Directory(s.repoRoot)
	builder = builder.
		WithDirectory(workdir, hostRepoRootDir, dagger.ContainerWithDirectoryOpts{
			Include: filesForPackageInstallation,
		}).
		WithExec([]string{"pnpm", "install", "--prefer-offline", "--frozen-lockfile"}).
		WithDirectory(workdir, hostRepoRootDir, dagger.ContainerWithDirectoryOpts{
			//Include: []string{"**"}, // include all files for build
			Exclude: append([]string{
				"**/node_modules",
				"**/dist",
				"**/build",
				"**/out",
				"**/.next",
				"**/.cache",
				"**/.turbo",
			}, s.config.ExcludeFiles...),
			Gitignore: true,
		})

	for _, task := range stepResults.Tasks {
		cmds, err := task.GetCmd()
		if err != nil {
			return fmt.Errorf("getting command for pipeline task: %w", err)
		}

		for _, cmd := range cmds {
			builder = builder.WithExec(cmd)
		}
	}

	nodeModulePaths := []string{"node_modules", filepath.Join(appPackage.Path, "node_modules")}
	for _, pkg := range append(dependencies, appPackage) {
		nodeModulePaths = append(nodeModulePaths, filepath.Join(pkg.Path, "node_modules"))
	}
	l.Debug("node_modules paths to clean up", "paths", nodeModulePaths)

	deps := client.Container(dagger.ContainerOpts{Platform: dagger.Platform(platform)}).
		From(baseImage).
		WithWorkdir(workdir).
		WithExec([]string{"apk", "add", "--no-cache", "curl"}).
		WithExec([]string{"curl", "-fsSL", "https://gobinaries.com/tj/node-prune", "-o", "/tmp/install-node-prune.sh"}).
		WithExec([]string{"sh", "/tmp/install-node-prune.sh"}).
		// IMPORTANT! - when caching volume mounted, it increases the image size by about 100MB with no significant package installation speed benefits
		WithExec([]string{"corepack", "enable"}).
		WithExec([]string{"corepack", "prepare", fmt.Sprintf("pnpm@%s", s.config.PnpmVersion), "--activate"}).
		WithEnvVariable("CI", "true").
		WithDirectory(workdir, builder.Directory(workdir), dagger.ContainerWithDirectoryOpts{
			Include: append([]string{}, filesForPackageInstallation...),
		}).
		WithExec([]string{"pnpm", "install", "--prefer-offline", "--frozen-lockfile", "--prod"}).
		WithExec([]string{"pnpm", "prune", "--prod", "--no-optional"}).
		// gobinaries usually installs into /usr/local/bin; if not, find it with `which node-prune`
		WithExec([]string{"/usr/local/bin/node-prune", "/app/node_modules"})

	allowedRuntimeStageTasks := []TaskID{
		TaskIDSetupPnpm,
		TaskIDSetupBun,
	}
	runtimeStepsResult, err := s.processSteps(s.config.RuntimeSteps, pipelinePlaceholderResolvers, allowedRuntimeStageTasks...)
	if err != nil {
		return fmt.Errorf("processing runtime steps: %w", err)
	}

	runtimePathsToInclude := includePaths
	runtime := client.Container(dagger.ContainerOpts{Platform: dagger.Platform(platform)}).
		From(baseImage).
		WithWorkdir(workdir)

	if len(runtimeStepsResult.SystemPackages) > 0 {
		runtime = runtime.WithExec(append([]string{"apk", "add", "--no-cache"}, runtimeStepsResult.SystemPackages...))

		for _, cmd := range runtimeStepsResult.PostInstallCmds {
			runtime = runtime.WithExec(cmd)
		}
	}

	if len(runtimeStepsResult.NpmPackages) > 0 {
		return fmt.Errorf("%w - installing npm packages is not supported in runtime phase", lib.BadUserInputError)
	}

	// The key parts for runtime image construction:
	// 1. Copy the pruned node_modules from the deps stage - it must utilize the layer caching so it is not uploaded every time the image is rebuilt
	// 2. Copy other node_modules for the packages in the monorepo. The goal is the same - utilize layer caching for node_modules
	// 3. Copy the rest of source code and build artifacts without overriding node_modules
	runtime = runtime.
		WithDirectory(filepath.Join(workdir, "node_modules"), deps.Directory(filepath.Join(workdir, "node_modules"))).
		WithDirectory(workdir, deps.Directory(workdir), dagger.ContainerWithDirectoryOpts{
			Include: slices.Collect(func(yield func(path string) bool) {
				for _, pkg := range append(dependencies, appPackage) {
					yield(filepath.Join(pkg.Path, "node_modules"))
				}
			}),
		}).
		WithDirectory(workdir, builder.Directory(workdir), dagger.ContainerWithDirectoryOpts{
			Include: runtimePathsToInclude,
			Exclude: []string{"node_modules"},
		})
	l.Info("runtime paths to include", "paths", runtimePathsToInclude)

	for _, task := range runtimeStepsResult.Tasks {
		cmds, err := task.GetCmd()
		if err != nil {
			return fmt.Errorf("getting command for runtime pipeline task: %w", err)
		}

		for _, cmd := range cmds {
			runtime = runtime.WithExec(cmd)
		}
	}

	err = runtime.
		WithEntrypoint([]string{cmd[0]}).
		WithDefaultArgs(cmd[1:]).
		ExportImage(ctx, outputImage) // TODO: ensure images compression when exported

	if err != nil {
		return fmt.Errorf("setting up pipeline container: %w", err)
	}

	l.Info("docker image built successfully via pipeline", "image", outputImage)
	l.Info(fmt.Sprintf("run 'docker run --rm -it %s sh' to access the image", outputImage))

	return nil
}

func (s *Service) processSteps(steps []Step, placeholderResolvers PlaceholderResolvers, allowedSteps ...TaskID) (processStepsResult, error) {
	l := slog.With("context", "pipeline_service", "method", "processSteps")
	l.Debug("processing steps", "steps_count", len(steps), "allowed_steps_count", len(allowedSteps), "steps", steps)

	result := processStepsResult{
		SystemPackages:  make([]string, 0, len(steps)),
		PostInstallCmds: make([][]string, 0, len(steps)),
		NpmPackages:     make([]string, 0, len(steps)),
		Tasks:           make([]Task, 0, len(steps)),
	}

	allowedStepsMap := make(map[TaskID]struct{}, len(allowedSteps))
	for _, allowedStep := range allowedSteps {
		allowedStepsMap[allowedStep] = struct{}{}
	}

	for _, step := range steps {
		var task Task

		if len(allowedStepsMap) > 0 {
			if _, ok := allowedStepsMap[step.Task]; !ok {
				l.Debug("task is not allowed in this context", "task", step.Task)
				return result, fmt.Errorf("%w - pipeline step '%s' is not allowed in this context", lib.BadUserInputError, step.Task)
			}
		}

		switch step.Task {
		case TaskIDGrpcGenerateTsProto:
			task = NewCompileProtobufToJsTask(step, placeholderResolvers, s.repoRoot, s.placeholders)
		case TaskIDSetupPnpm:
			task = NewSetupPnpmTask(s.config, step)
		case TaskIDCli:
			task = NewCliTask(step, placeholderResolvers, s.placeholders)
		case TaskIDSetupBun:
			task = NewSetupBunTask(step)
		default:
			return result, fmt.Errorf("%w - unsupported pipeline task '%s'", lib.BadUserInputError, step.Task)
		}

		result.SystemPackages = append(result.SystemPackages, task.GetRequiredSystemPackages()...)

		postInstallCommands, err := task.GetPostInstallCommands()
		if err != nil {
			return result, fmt.Errorf("getting post install commands: %w", err)
		}
		result.PostInstallCmds = append(result.PostInstallCmds, postInstallCommands...)
		result.NpmPackages = append(result.NpmPackages, task.GetRequiredNpmPackages()...)
		result.Tasks = append(result.Tasks, task)

		l.Debug("task processed",
			"task", step.Task,
			"system_packages", task.GetRequiredSystemPackages(),
			"npm_packages", task.GetRequiredNpmPackages(),
			"post_install_cmds", postInstallCommands)
	}

	return result, nil
}
