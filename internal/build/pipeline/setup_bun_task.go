package pipeline

import (
	"fmt"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

type SetupBunTask struct {
	step Step
}

func NewSetupBunTask(step Step) Task {
	return &SetupBunTask{step}
}

func (t *SetupBunTask) GetTaskID() TaskID {
	return t.step.Task
}

func (t *SetupBunTask) GetRequiredSystemPackages() []string {
	return []string{"curl", "unzip", "bash"}
}

func (t *SetupBunTask) GetPostInstallCommands() ([][]string, error) {
	version, err := t.getBunVersion()
	if err != nil {
		return nil, fmt.Errorf("error getting bun version: %w", err)
	}

	return [][]string{
		{"curl", "-fsSL", "https://bun.sh/install", "-o", "/tmp/bun-install.sh"},
		{"bash", "/tmp/bun-install.sh", version},
		// TODO: fix installation path to be not root specific
		{"ln", "-sf", "/root/.bun/bin/bun", "/usr/local/bin/bun"},
		{"ln", "-sf", "/usr/local/bin/bun", "/usr/local/bin/bunx"},
	}, nil
}

func (t *SetupBunTask) GetRequiredNpmPackages() []string {
	return []string{}
}

func (t *SetupBunTask) GetCmd() ([][]string, error) {
	return [][]string{}, nil
}

func (t *SetupBunTask) getBunVersion() (string, error) {
	version, ok := t.step.Extra["version"]
	if !ok {
		return "", fmt.Errorf("%w - 'version' is required for bun setup step", lib.BadUserInputError)
	}
	versionStr, ok := version.(string)
	if !ok {
		return "", fmt.Errorf("%w - bun 'version' must be a string", lib.BadUserInputError)
	}

	return versionStr, nil
}
