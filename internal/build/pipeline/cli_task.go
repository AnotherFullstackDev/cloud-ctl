package pipeline

import (
	"fmt"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders"
)

type CliTask struct {
	step                 Step
	placeholderResolvers PlaceholderResolvers
	placeholders         *placeholders.Service
}

func NewCliTask(step Step, placeholderResolvers PlaceholderResolvers, placeholders *placeholders.Service) Task {
	return &CliTask{step, placeholderResolvers, placeholders}
}

func (t *CliTask) GetTaskID() TaskID {
	return t.step.Task
}

func (t *CliTask) GetRequiredSystemPackages() []string {
	return []string{}
}

func (t *CliTask) GetPostInstallCommands() ([][]string, error) {
	return [][]string{}, nil
}

func (t *CliTask) GetRequiredNpmPackages() []string {
	return []string{}
}

func (t *CliTask) GetCmd() ([][]string, error) {
	cmd, ok := t.step.Extra["cmd"]
	if !ok {
		return nil, fmt.Errorf("%w - no 'cmd' specified in extra for cli task", lib.BadUserInputError)
	}
	cmdSlice, err := lib.ConfigEntryToTypedSlice[string](cmd, "cmd")
	if err != nil {
		return nil, fmt.Errorf("converting cmd to a strings slice: %w", err)
	}
	for i, cmdPart := range cmdSlice {
		resolvedCmdPart, err := t.placeholders.ResolvePlaceholders(cmdPart, t.placeholderResolvers)
		if err != nil {
			return nil, fmt.Errorf("resolving placeholders in cmd part '%s': %w", cmdPart, err)
		}
		cmdSlice[i] = resolvedCmdPart
	}

	workdir, ok := t.step.Extra["workdir"]
	if !ok {
		workdir = "."
	}
	workdirStr, ok := workdir.(string)
	if !ok {
		return nil, fmt.Errorf("%w - 'workdir' must be a string", lib.BadUserInputError)
	}
	workdirStr, err = t.placeholders.ResolvePlaceholders(workdirStr, t.placeholderResolvers)
	if err != nil {
		return nil, fmt.Errorf("resolving placeholders: %w", err)
	}

	// Likely not the ideal approach, might worth to investigate a better way to handle this in dagger
	mainCmd := append([]string{"sh", "-lc", strings.Join(append([]string{"cd", workdirStr, "&&"}, cmdSlice...), " ")})

	return [][]string{
		mainCmd,
	}, nil
}
