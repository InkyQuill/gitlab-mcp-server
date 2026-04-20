package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// promptSecret reads a secret from the controlling TTY without echoing.
// If stdin is not a TTY, it returns an error instructing the caller to use
// --token-ref instead.
func promptSecret(prompt string) (string, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", errors.New("no TTY available — use --token-ref <ref> in non-interactive contexts")
	}
	defer f.Close()
	_, _ = fmt.Fprint(f, prompt)
	fd := int(f.Fd())
	bytes, err := term.ReadPassword(fd)
	fmt.Fprintln(f)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}
