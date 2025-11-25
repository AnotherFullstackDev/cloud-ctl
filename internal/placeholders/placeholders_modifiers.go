package placeholders

import (
	"fmt"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
)

func upperModifier(input string, args []string) (string, error) {
	return strings.ToUpper(input), nil
}

func lowerModifier(input string, args []string) (string, error) {
	return strings.ToLower(input), nil
}

func trimModifier(input string, args []string) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("trim modifier expects at most one argument, got %d. %w", len(args), lib.BadUserInputError)
	}
	if len(args) == 0 {
		return strings.TrimSpace(input), nil
	}
	return strings.Trim(input, args[0]), nil
}

func replaceModifier(input string, args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("replace modifier expects exactly two arguments, got %d. %w", len(args), lib.BadUserInputError)
	}
	return strings.Replace(input, args[0], args[1], 1), nil
}

func replaceAllModifier(input string, args []string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("replace_all modifier expects exactly two arguments, got %d. %w", len(args), lib.BadUserInputError)
	}
	return strings.ReplaceAll(input, args[0], args[1]), nil
}
