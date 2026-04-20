package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ExternalCmdBackend resolves secrets by shelling out to a user-configured command.
// Schemes are allow-listed at construction time via a templates map:
//
//	op:   "op read %s"
//	pass: "pass show %s"
//	age:  "age -d -i /home/u/.config/age/key.txt %s"
//
// The %s is replaced with the opaque portion of the ref. Commands are executed
// via exec.Command (not `sh -c`), so shell metacharacters in the ref cannot
// influence command parsing. Store is not implemented — users populate their
// external tool directly.
type ExternalCmdBackend struct {
	templates map[string]string
}

// NewExternalCmdBackend constructs a backend from a scheme → template map. The
// map is copied, so later mutation by the caller does not affect the backend.
func NewExternalCmdBackend(templates map[string]string) *ExternalCmdBackend {
	cp := make(map[string]string, len(templates))
	for k, v := range templates {
		cp[k] = v
	}
	return &ExternalCmdBackend{templates: cp}
}

// Schemes returns the list of schemes this backend handles.
func (x *ExternalCmdBackend) Schemes() []string {
	out := make([]string, 0, len(x.templates))
	for k := range x.templates {
		out = append(out, k)
	}
	return out
}

// Scheme is NOT meaningful for this multi-scheme backend. It returns "external"
// as a sentinel — callers should use RegisterAll to install one shim per scheme.
func (x *ExternalCmdBackend) Scheme() string { return "external" }

// RegisterAll registers a per-scheme shim in the given registry for every
// scheme this backend handles. This keeps the one-scheme-per-backend rule in
// BackendRegistry intact.
func (x *ExternalCmdBackend) RegisterAll(r *BackendRegistry) error {
	for scheme := range x.templates {
		if err := r.Register(&externalSchemeShim{scheme: scheme, parent: x}); err != nil {
			return err
		}
	}
	return nil
}

// Resolve runs the templated command and returns its last non-empty stdout line.
func (x *ExternalCmdBackend) Resolve(ctx context.Context, ref string) (string, error) {
	scheme, opaque, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	tmpl, ok := x.templates[scheme]
	if !ok {
		return "", fmt.Errorf("external-cmd backend: no command template for scheme %q", scheme)
	}
	name, args, err := buildExec(tmpl, opaque)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("external-cmd backend: %s: %w", name, err)
	}
	secret := lastNonEmptyLine(string(out))
	if secret == "" {
		return "", fmt.Errorf("external-cmd backend: empty output from %s", name)
	}
	return secret, nil
}

// Store is unsupported: external tools are populated by the user out-of-band.
func (x *ExternalCmdBackend) Store(_ context.Context, _, _ string) (string, error) {
	return "", errors.New("external-cmd backend: store is not supported — populate your external tool directly and pass --token-ref")
}

// Delete is unsupported for the same reason.
func (x *ExternalCmdBackend) Delete(_ context.Context, _ string) error {
	return errors.New("external-cmd backend: delete is not supported — manage entries in your external tool directly")
}

// buildExec splits the template on whitespace, substitutes %s with the opaque
// ref, and returns (name, args). Metacharacters in the opaque are preserved as
// a single argv element — no shell is involved.
func buildExec(tmpl, opaque string) (string, []string, error) {
	fields := strings.Fields(tmpl)
	if len(fields) == 0 {
		return "", nil, errors.New("external-cmd backend: empty command template")
	}
	replaced := make([]string, 0, len(fields))
	subs := 0
	for _, f := range fields {
		if strings.Contains(f, "%s") {
			replaced = append(replaced, strings.ReplaceAll(f, "%s", opaque))
			subs++
			continue
		}
		replaced = append(replaced, f)
	}
	if subs == 0 {
		return "", nil, errors.New("external-cmd backend: template missing %s placeholder")
	}
	return replaced[0], replaced[1:], nil
}

func lastNonEmptyLine(s string) string {
	sc := bufio.NewScanner(strings.NewReader(s))
	last := ""
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		if line != "" {
			last = line
		}
	}
	return last
}

// externalSchemeShim adapts ExternalCmdBackend to the single-scheme Register API.
type externalSchemeShim struct {
	scheme string
	parent *ExternalCmdBackend
}

func (s *externalSchemeShim) Scheme() string { return s.scheme }
func (s *externalSchemeShim) Resolve(ctx context.Context, ref string) (string, error) {
	return s.parent.Resolve(ctx, ref)
}
func (s *externalSchemeShim) Store(ctx context.Context, name, secret string) (string, error) {
	return s.parent.Store(ctx, name, secret)
}
func (s *externalSchemeShim) Delete(ctx context.Context, ref string) error {
	return s.parent.Delete(ctx, ref)
}
