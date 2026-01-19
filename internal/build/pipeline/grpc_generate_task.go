package pipeline

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
	ignore "github.com/sabhiram/go-gitignore"
)

type GrpcGenerateTask struct {
	step                 Step
	placeholderResolvers PlaceholderResolvers
	repoRoot             string
	placeholders         *placeholders.Service
}

type ProtoFile struct {
	Path   string
	Parent string
}

func NewCompileProtobufToJsTask(step Step, placeholderResolvers PlaceholderResolvers, repoRoot string, placeholders *placeholders.Service) Task {
	return &GrpcGenerateTask{step, placeholderResolvers, repoRoot, placeholders}
}

func (t *GrpcGenerateTask) GetTaskID() TaskID {
	return t.step.Task
}

func (t *GrpcGenerateTask) GetRequiredSystemPackages() []string {
	return []string{
		"protobuf",
		"protobuf-dev",
	}
}

func (t *GrpcGenerateTask) GetPostInstallCommands() ([][]string, error) {
	return [][]string{
		// TODO: check it, because it feels like just installation of extra proto files/packages to use during compilation & inside proto files declaration
		{"mkdir", "-p", "/usr/local/include/google"},
		{"ln", "-s", "/usr/include/google/protobuf", "/usr/local/include/google/protobuf"},
	}, nil
}

func (t *GrpcGenerateTask) getProtoFilesByPatterns(includePatterns, excludePatterns []string) ([]ProtoFile, error) {
	l := slog.With("context", "grpc_generate_task", "method", "getProtoFilesByPatterns")

	repoRoot := filepath.Clean(t.repoRoot)
	include := includePatterns
	exclude := excludePatterns

	gitIgnore, err := ignore.CompileIgnoreFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("compile gitignore: %w", err)
	}

	matches := make(map[string]ProtoFile, len(include))

	walkErr := filepath.WalkDir(repoRoot, func(absPath string, d fs.DirEntry, err error) error {
		absolutePath, err := filepath.Abs(absPath)
		if err != nil {
			return fmt.Errorf("getting absolute path: %w", err)
		}
		l.Debug("walking path", "path", absPath, "abs_path", absolutePath)

		if err != nil {
			l.Debug("walking path error", "path", absPath, "err", err)
			return err
		}

		relPath, err := filepath.Rel(repoRoot, absPath)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}
		relPath = filepath.ToSlash(relPath)

		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
			l.Debug("skipping excluded path", "path", relPath, "reason", ".git directory")
			return fs.SkipDir
		}

		//if d.IsDir() {
		//	return nil
		//}

		if relPath != "." && gitIgnore != nil {
			if d.IsDir() {
				if gitIgnore.MatchesPath(relPath) || gitIgnore.MatchesPath(relPath+"/") {
					l.Debug("skipping excluded path", "path", relPath, "reason", "folder under .gitignore")
					return fs.SkipDir
				}
			} else {
				if gitIgnore.MatchesPath(relPath) {
					l.Debug("skipping excluded path", "path", relPath, "reason", "file under .gitignore")
					return nil
				}
			}
		}

		matchesIncludes, err := lib.PathMatchesOneOfPatterns(relPath, include)
		if err != nil {
			return fmt.Errorf("matching include patterns: %w", err)
		}
		if !matchesIncludes {
			return nil
		}

		matchesExcludes, err := lib.PathMatchesOneOfPatterns(relPath, exclude)
		if err != nil {
			return fmt.Errorf("matching exclude patterns: %w", err)
		}
		if matchesExcludes {
			return nil
		}

		matches[relPath] = ProtoFile{
			Path:   relPath,
			Parent: filepath.Dir(relPath),
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("walk monorepo packages: %w", walkErr)
	}

	result := make([]ProtoFile, 0, len(matches))
	for _, workspacePackage := range matches {
		result = append(result, workspacePackage)
	}

	return result, nil
}

func (t *GrpcGenerateTask) GetRequiredNpmPackages() []string {
	return []string{
		"ts-proto", // TODO: make the version configurable
	}
}

// TODO: make proto root directories configurable.
// By default it is the repository root, but some users might want to configure it differently.
func (t *GrpcGenerateTask) GetCmd() ([][]string, error) {
	l := slog.With("context", "grpc_generate_task", "step", t.step.Task)

	includePatterns, ok := t.step.Extra["include"]
	if !ok {
		return nil, fmt.Errorf("%w - 'include' patterns not specified for %s", lib.BadUserInputError, t.step.Task)
	}
	includePatternsSlice, err := lib.ConfigEntryToTypedSlice[string](includePatterns, "include")
	if err != nil {
		return nil, fmt.Errorf("%w - 'include' patterns should be a list of strings for %s: %w", lib.BadUserInputError, t.step.Task, err)
	}
	l.Debug("not validated include patterns", "patterns", includePatternsSlice, "len", len(includePatternsSlice))
	for _, includePattern := range includePatternsSlice {
		if !strings.HasSuffix(includePattern, ".proto") {
			return nil, fmt.Errorf("%w - 'include' patterns should point to .proto files for %s, got: %s", lib.BadUserInputError, t.step.Task, includePattern)
		}
	}
	l.Info("got include patterns", "patterns", includePatternsSlice)

	excludePatterns, ok := t.step.Extra["exclude"]
	if !ok {
		excludePatterns = []interface{}{}
	}
	excludePatternsSlice, err := lib.ConfigEntryToTypedSlice[string](excludePatterns, "exclude")
	if err != nil {
		return nil, fmt.Errorf("%w - 'exclude' patterns should be a list of strings for %s: %w", lib.BadUserInputError, t.step.Task, err)
	}
	l.Info("got exclude patterns", "patterns", excludePatternsSlice)

	out, ok := t.step.Extra["out"]
	if !ok {
		return nil, fmt.Errorf("%w - 'out' option not specified for %s", lib.BadUserInputError, t.step.Task)
	}
	outStr, ok := out.(string)
	if !ok {
		return nil, fmt.Errorf("%w - 'out' option should be a string for %s", lib.BadUserInputError, t.step.Task)
	}
	if outStr == "" {
		return nil, fmt.Errorf("%w - 'out' option should be a non-empty string for %s", lib.BadUserInputError, t.step.Task)
	}
	outStr, err = t.placeholders.ResolvePlaceholders(outStr, t.placeholderResolvers)
	if err != nil {
		return nil, fmt.Errorf("resolve task %s placeholders: %w", t.step.Task, err)
	}
	l.Info("got output location", "out", outStr)

	options, ok := t.step.Extra["opt"]
	if !ok {
		options = []any{}
	}
	optionsSlice, err := lib.ConfigEntryToTypedSlice[string](options, "opt")
	if err != nil {
		return nil, fmt.Errorf("%w - 'opt' option should be a map of string to string for %s: %w", lib.BadUserInputError, t.step.Task, err)
	}
	extraTsProtoOptions := make([]string, 0, len(optionsSlice))
	for _, tsProtoOption := range optionsSlice {
		extraTsProtoOptions = append(extraTsProtoOptions, fmt.Sprintf("--ts_proto_opt=%s", tsProtoOption))
	}

	protoFiles, err := t.getProtoFilesByPatterns(includePatternsSlice, excludePatternsSlice)
	if err != nil {
		return nil, fmt.Errorf("get protobuf files: %w", err)
	}

	protoFilePaths := make([]string, 0, len(protoFiles))
	//protoFileFoldersSet := make(map[string]struct{}, len(protoFiles))
	//protoFileFolders := make([]string, 0, len(protoFiles))
	for _, protoFile := range protoFiles {
		protoFilePaths = append(protoFilePaths, protoFile.Path)

		//if _, ok := protoFileFoldersSet[protoFile.Parent]; ok {
		//	continue
		//}
		//
		//protoFileFoldersSet[protoFile.Parent] = struct{}{}
		//protoFileFolders = append(protoFileFolders, protoFile.Parent)
	}
	l.Info("found proto files", "count", len(protoFilePaths), "files", protoFilePaths)
	//l.Info("proto file folders", "count", len(protoFileFolders), "folders", protoFileFolders)

	// TODO: here will be logic for proto root folders configuration
	// Currently it is the root of the specified monorepo
	protoRootFolders := []string{"."}
	protoPaths := make([]string, 0, len(protoRootFolders))
	for _, protoFolder := range protoRootFolders {
		protoPaths = append(protoPaths, fmt.Sprintf("--proto_path=%s", protoFolder))
	}

	// TODO: make the compilation plugin configurable
	cmd := []string{"protoc"}
	cmd = append(cmd, extraTsProtoOptions...)
	cmd = append(cmd, []string{
		// Likely depending on the compilation plugin commands will be different
		//"--plugin=protoc-gen-ts_proto=./node_modules/.bin/protoc-gen-ts_proto",
		//"--plugin=./node_modules/.bin/protoc-gen-ts_proto",
		fmt.Sprintf("--ts_proto_out=%s", outStr),
		//"--ts_proto_opt=esModuleInterop=true", // TODO: make configurable
		//"--ts_proto_opt=outputServices=grpc-js", // TODO: make configurable
		"--proto_path=/usr/local/include",
	}...)
	cmd = append(cmd, protoPaths...)
	cmd = append(cmd, protoFilePaths...)
	l.Info("running command", "cmd", strings.Join(cmd, " "))

	resultingCmds := [][]string{
		{"mkdir", "-p", outStr},
		cmd,
	}

	return resultingCmds, nil
}
