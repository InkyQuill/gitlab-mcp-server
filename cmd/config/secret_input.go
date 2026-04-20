package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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
	fmt.Fprint(f, prompt)
	fd := int(f.Fd())
	bytes, err := term.ReadPassword(fd)
	fmt.Fprintln(f)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

// promptLine reads a line from the given reader (used for backend-choice prompts).
func promptLine(r io.Reader, prompt string, dflt string) (string, error) {
	fmt.Print(prompt)
	sc := bufio.NewScanner(r)
	if sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s == "" {
			return dflt, nil
		}
		return s, nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return dflt, nil
}
