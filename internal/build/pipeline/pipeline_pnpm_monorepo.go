package pipeline

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	ignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v3"
)

type PnpmMonorepo struct {
	repoRoot string
}

type PackageJson struct {
	Name             string            `json:"name"`
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

type WorkspacePackage struct {
	Path         string
	Manifest     PackageJson
	ManifestPath string
}

type WorkspaceManifest struct {
	Packages []string `yaml:"packages"`
}

func NewPnpmMonorepo(repoRoot string) *PnpmMonorepo {
	return &PnpmMonorepo{repoRoot}
}

func (p *PnpmMonorepo) GetWorkspacePackages() ([]WorkspacePackage, error) {
	repoRoot := filepath.Clean(p.repoRoot)

	workspace, err := p.getWorkspaceManifest()
	if err != nil {
		return nil, fmt.Errorf("get workspace manifest: %w", err)
	}

	include, exclude := p.splitWorkspacePackagesPatterns(workspace.Packages)

	gitIgnore, err := ignore.CompileIgnoreFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("compile gitignore: %w", err)
	}

	matches := make(map[string]WorkspacePackage, len(include))

	walkErr := filepath.WalkDir(repoRoot, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, absPath)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}
		relPath = filepath.ToSlash(relPath)
		//if relPath == "." {
		//	relPath = ""
		//}

		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
			return fs.SkipDir
		}

		if relPath != "." && gitIgnore != nil && (gitIgnore.MatchesPath(relPath) || gitIgnore.MatchesPath(relPath+"/")) {
			return fs.SkipDir
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

		packageManifestPath := filepath.Join(absPath, "package.json")
		st, err := os.Stat(packageManifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("stat package manifest: %w", err)
		}
		if st.IsDir() {
			return nil
		}

		content, err := os.ReadFile(packageManifestPath)
		if err != nil {
			return fmt.Errorf("read package manifest: %w", err)
		}

		var pkgManifest PackageJson
		if err := json.Unmarshal(content, &pkgManifest); err != nil {
			return fmt.Errorf("unmarshal package manifest: %w", err)
		}

		matches[relPath] = WorkspacePackage{
			Path:         relPath,
			Manifest:     pkgManifest,
			ManifestPath: filepath.Join(relPath, "package.json"),
		}

		return nil
	})

	if walkErr != nil {
		return nil, fmt.Errorf("walk monorepo packages: %w", walkErr)
	}

	packages := make([]WorkspacePackage, 0, len(matches))
	for _, workspacePackage := range matches {
		packages = append(packages, workspacePackage)
	}
	slices.SortFunc(packages, func(a, b WorkspacePackage) int {
		return strings.Compare(a.Path, b.Path)
	})

	return packages, nil
}

type PackageDependencyType string

const (
	PackageDependencyTypeDependencies     PackageDependencyType = "dependencies"
	PackageDependencyTypeDevDependencies  PackageDependencyType = "devDependencies"
	PackageDependencyTypePeerDependencies PackageDependencyType = "peerDependencies"
)

func (p *PnpmMonorepo) GetPackageDependencies(pkg WorkspacePackage, workspacePackages []WorkspacePackage, dependencyTypes ...PackageDependencyType) []WorkspacePackage {
	dependencies := make(map[string]WorkspacePackage, len(workspacePackages))
	return p.getPackageDependencies(dependencies, pkg, workspacePackages, dependencyTypes...)
}

func (p *PnpmMonorepo) getPackageDependencies(dependencies map[string]WorkspacePackage, pkg WorkspacePackage, workspacePackages []WorkspacePackage, dependencyTypes ...PackageDependencyType) []WorkspacePackage {
	totalDependencies := make(map[string]string, len(workspacePackages))
	for _, dependencyType := range dependencyTypes {
		switch dependencyType {
		case PackageDependencyTypeDependencies:
			maps.Copy(totalDependencies, pkg.Manifest.Dependencies)
		case PackageDependencyTypeDevDependencies:
			maps.Copy(totalDependencies, pkg.Manifest.DevDependencies)
		case PackageDependencyTypePeerDependencies:
			maps.Copy(totalDependencies, pkg.Manifest.PeerDependencies)
		}
	}

	for _, workspacePkg := range workspacePackages {
		if _, ok := totalDependencies[workspacePkg.Manifest.Name]; ok {
			workspacePackages = slices.DeleteFunc(workspacePackages, func(p WorkspacePackage) bool {
				_, processed := dependencies[p.Manifest.Name]
				return processed
			})

			dependencies[workspacePkg.Manifest.Name] = workspacePkg

			relatedDependencies := p.getPackageDependencies(dependencies, workspacePkg, workspacePackages, dependencyTypes...)
			for _, relatedDep := range relatedDependencies {
				dependencies[relatedDep.Manifest.Name] = relatedDep
			}
		}
	}

	result := make([]WorkspacePackage, 0, len(dependencies))
	for _, dep := range dependencies {
		result = append(result, dep)
	}
	slices.SortFunc(result, func(a, b WorkspacePackage) int {
		return strings.Compare(a.Path, b.Path)
	})

	return result
}

func (p *PnpmMonorepo) getWorkspaceManifest() (WorkspaceManifest, error) {
	l := slog.With("context", "pnpm-monorepo-service", "method", "getWorkspaceManifest")

	var manifest WorkspaceManifest

	repoRoot := filepath.Clean(p.repoRoot)
	manifestPath := filepath.Join(repoRoot, "pnpm-workspace.yaml")
	l.Debug("loading manifest", "path", manifestPath)

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest, err
	}

	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return manifest, err
	}

	if len(manifest.Packages) == 0 {
		return manifest, fmt.Errorf("%w - no packages found in %s", lib.BadUserInputError, manifestPath)
	}

	return manifest, nil
}

func (p *PnpmMonorepo) splitWorkspacePackagesPatterns(patterns []string) (includePatterns, excludePatterns []string) {
	for _, pattern := range patterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}

		negative := strings.HasPrefix(p, "!")
		if negative {
			p = strings.TrimSpace(strings.TrimPrefix(p, "!"))
		}

		p = strings.TrimSpace(strings.TrimPrefix(p, "./"))
		p = filepath.ToSlash(p)

		if negative {
			excludePatterns = append(excludePatterns, p)
		} else {
			includePatterns = append(includePatterns, p)
		}
	}

	return includePatterns, excludePatterns
}
