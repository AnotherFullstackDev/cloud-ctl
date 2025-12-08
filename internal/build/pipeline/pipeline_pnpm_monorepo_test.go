package pipeline

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindPnpmWorkspacePackages_BasicIncludeExcludeAndGitignore(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	t.Run("basic include/exclude with gitignore", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		// pnpm workspace patterns include ** and exclude one sub-tree.
		writeFile(t, root, "pnpm-workspace.yaml", `
packages:
  - "apps/*"
  - "packages/**"
  - "!packages/**/fixtures/**"
`)

		// .gitignore: ignore dist, node_modules, and a specific folder.
		writeFile(t, root, ".gitignore", `
node_modules/
dist/
ignored-app/
`)

		dirSpec := DirectorySpec{
			"apps/api": {
				{Name: "package.json", Content: `{ "name": "api" }`},
			},
			// No package.json here; should be ignored.
			"apps/web": {},
			"ignored-app/x": {
				{Name: "package.json", Content: `{ "name": "ignored" }`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{ "name": "lib-b" }`},
			},
			"packages/lib-b/dist": {
				{Name: "junk.txt", Content: `x`},
			},
			"packages/tooling/fixtures/demo": {
				{Name: "package.json", Content: `{ "name": "demo" }`},
			},
			"node_modules/sneaky": {
				{Name: "package.json", Content: `{ "name": "sneaky" }`},
			},
			"node_modules/@org/pkg": {
				{Name: "package.json", Content: `{ "name": "@org/pkg" }`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		got, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		var gotNames []string
		for _, pkg := range got {
			gotNames = append(gotNames, pkg.Path)
		}
		sort.Strings(gotNames)

		want := []string{
			"apps/api",
			"packages/lib-a",
			"packages/lib-b",
		}
		r.Equal(want, gotNames)
	})

	t.Run("ignores everything in .git", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		writeFile(t, root, "pnpm-workspace.yaml", `
packages:
  - "**"
`)

		// .git directory with an apparent package.json should never be returned.
		DirectorySpec{
			".git": {
				{Name: "config", Content: `some git config`},
			},
			".git/sneaky": {
				{Name: "package.json", Content: `{ "name": "nope" }`},
			},
		}.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		got, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		var gotNames []string
		for _, pkg := range got {
			gotNames = append(gotNames, pkg.Path)
		}

		r.NotContains(gotNames, ".git/sneaky")
	})

	t.Run("only root package", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		writeFile(t, root, "pnpm-workspace.yaml", `
packages:
  - "."
`)
		writeFile(t, root, "package.json", `{ "name": "root" }`)

		workspace := NewPnpmMonorepo(root)
		got, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		var gotNames []string
		for _, pkg := range got {
			gotNames = append(gotNames, pkg.Path)
		}

		r.Equal([]string{"."}, gotNames)
	})

	t.Run("no packages matched", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		writeFile(t, root, "pnpm-workspace.yaml", `
packages:
  - "nonexistent/**"
`)

		workspace := NewPnpmMonorepo(root)
		got, err := workspace.GetWorkspacePackages()
		r.NoError(err)
		r.Empty(got)
	})
}

func TestPnpmMonorepo_GetPackageDependencies(t *testing.T) {
	t.Parallel()
	r := require.New(t)

	t.Run("no dependencies", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			// Package A has no dependencies.
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "dependencies": {"lib-a": "workspace:*"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgA := packages[0] // lib-a
		deps := workspace.GetPackageDependencies(pkgA, packages, PackageDependencyTypeDependencies, PackageDependencyTypeDevDependencies, PackageDependencyTypePeerDependencies)
		r.Empty(deps)
	})

	t.Run("dependencies declared via workspace:*", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "dependencies": {"lib-a": "workspace:*"}}`},
			},
			"packages/lib-c": {
				{Name: "package.json", Content: `{"name": "lib-c", "devDependencies": {"lib-b": "workspace:*"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgC := packages[2] // lib-c
		deps := workspace.GetPackageDependencies(pkgC, packages, PackageDependencyTypeDependencies, PackageDependencyTypeDevDependencies, PackageDependencyTypePeerDependencies)

		var depNames []string
		for _, dep := range deps {
			depNames = append(depNames, dep.Manifest.Name)
		}
		sort.Strings(depNames)

		want := []string{"lib-a", "lib-b"}
		r.Equal(want, depNames)
	})

	t.Run("dependencies declared via version numbers", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a", "version": "1.0.0" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "dependencies": {"lib-a": "^1.0.0"}}`},
			},
			"packages/lib-c": {
				{Name: "package.json", Content: `{"name": "lib-c", "devDependencies": {"lib-b": "~1.0.0"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgC := packages[2] // lib-c
		deps := workspace.GetPackageDependencies(pkgC, packages, PackageDependencyTypeDependencies, PackageDependencyTypeDevDependencies, PackageDependencyTypePeerDependencies)

		var depNames []string
		for _, dep := range deps {
			depNames = append(depNames, dep.Manifest.Name)
		}
		sort.Strings(depNames)

		want := []string{"lib-a", "lib-b"}
		r.Equal(want, depNames)
	})

	t.Run("dependencies declared but not present in workspace", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "dependencies": {"non-workspace-pkg": "^1.0.0"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgB := packages[1] // lib-b
		deps := workspace.GetPackageDependencies(pkgB, packages, PackageDependencyTypeDependencies, PackageDependencyTypeDevDependencies, PackageDependencyTypePeerDependencies)
		r.Empty(deps)
	})

	t.Run("peer dependencies of subpackages are always included", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{ "name": "lib-a" }`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "peerDependencies": {"lib-a": "workspace:*"}}`},
			},
			"packages/lib-c": {
				{Name: "package.json", Content: `{"name": "lib-c", "dependencies": {"lib-b": "workspace:*", "lib-a": "workspace:*"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgC := packages[2] // lib-c
		deps := workspace.GetPackageDependencies(pkgC, packages, PackageDependencyTypeDependencies)

		var depNames []string
		for _, dep := range deps {
			depNames = append(depNames, dep.Manifest.Name)
		}
		sort.Strings(depNames)

		want := []string{"lib-a", "lib-b"}
		r.Equal(want, depNames)
	})

	t.Run("packages with circular dependencies", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()

		dirSpec := DirectorySpec{
			".": {
				{Name: "pnpm-workspace.yaml", Content: `
packages:
  - "packages/*"
`},
			},
			"packages/lib-a": {
				{Name: "package.json", Content: `{"name": "lib-a", "dependencies": {"lib-b": "workspace:*"}}`},
			},
			"packages/lib-b": {
				{Name: "package.json", Content: `{"name": "lib-b", "dependencies": {"lib-c": "workspace:*"}}`},
			},
			"packages/lib-c": {
				{Name: "package.json", Content: `{"name": "lib-c", "dependencies": {"lib-a": "workspace:*"}}`},
			},
		}
		dirSpec.Build(t, root)

		workspace := NewPnpmMonorepo(root)
		packages, err := workspace.GetWorkspacePackages()
		r.NoError(err)

		pkgA := packages[0] // lib-a
		deps := workspace.GetPackageDependencies(pkgA, packages, PackageDependencyTypeDependencies)

		var depNames []string
		for _, dep := range deps {
			depNames = append(depNames, dep.Manifest.Name)
		}
		sort.Strings(depNames)

		want := []string{"lib-a", "lib-b", "lib-c"}
		r.Equal(want, depNames)
	})
}

// --- helpers ---

func mkdirAll(t *testing.T, root string, rel string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755)
	require.NoError(t, err)
}

func writeFile(t *testing.T, root string, rel string, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	err := os.MkdirAll(filepath.Dir(abs), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(abs, []byte(content), 0o644)
	require.NoError(t, err)
}

type FileSpec struct {
	Name    string
	Content string
}

type DirectorySpec map[string][]*FileSpec

func (d DirectorySpec) Build(t *testing.T, root string) {
	t.Helper()
	for directoryPath, files := range d {
		mkdirAll(t, root, directoryPath)
		for _, file := range files {
			if file != nil {
				writeFile(t, root, directoryPath+"/"+file.Name, file.Content)
			}
		}
	}
}
