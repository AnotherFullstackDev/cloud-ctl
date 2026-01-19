package pipeline

import "fmt"

type SetupPnpmTask struct {
	config Config
	step   Step
}

func NewSetupPnpmTask(config Config, step Step) Task {
	return &SetupPnpmTask{config, step}
}

func (t *SetupPnpmTask) GetTaskID() TaskID {
	return t.step.Task
}

func (t *SetupPnpmTask) GetRequiredSystemPackages() []string {
	return []string{}
}

func (t *SetupPnpmTask) GetPostInstallCommands() ([][]string, error) {
	return [][]string{}, nil
}

func (t *SetupPnpmTask) GetRequiredNpmPackages() []string {
	return []string{}
}

func (t *SetupPnpmTask) GetCmd() ([][]string, error) {
	return [][]string{
		{"corepack", "enable"},
		{"corepack", "prepare", fmt.Sprintf("pnpm@%s", t.config.PnpmVersion), "--activate"},
	}, nil
}
