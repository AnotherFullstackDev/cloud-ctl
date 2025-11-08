package lib

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/term"
)

func RequestSecretInput(in io.Reader, out io.Writer, prompt string) (string, error) {
	_, err := fmt.Fprintf(out, "%s: ", prompt)
	if err != nil {
		return "", fmt.Errorf("writing prompt: %w", err)
	}

	defer slog.Debug("secret received")

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

	slog.Debug("Not a terminal, falling back to normal input reading")

	// When not a terminal, fall back to normal input reading
	reader := bufio.NewReader(in)
	secret, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading secret input: %w", err)
	}

	return strings.TrimSpace(secret), nil
}
