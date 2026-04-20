package config

import (
	"context"
	"fmt"
	"os/exec"

	pkgconfig "github.com/InkyQuill/gitlab-mcp-server/pkg/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func newBackendsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backends",
		Short: "List available secret backends and probe their health",
		Long: `Diagnose which secret backends are compiled in, which are reachable,
and what command templates are configured for external backends.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mgr, err := pkgconfig.NewManager("")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Secret backends:")

			// keyring probe
			probeKey := "gitlab-mcp-server-probe"
			probeAcct := "probe"
			setErr := keyring.Set(probeKey, probeAcct, "ok")
			if setErr == nil {
				_ = keyring.Delete(probeKey, probeAcct)
				fmt.Fprintln(out, "  keyring      OK (default)")
			} else {
				fmt.Fprintf(out, "  keyring      UNAVAILABLE: %v\n", setErr)
			}

			// file backend is always compiled in
			fmt.Fprintln(out, "  file         OK (~/.gitlab-mcp-server/secrets.enc when used)")

			// external commands
			if mgr.Config().Backends != nil && len(mgr.Config().Backends.External) > 0 {
				fmt.Fprintln(out, "  external:")
				for scheme, tmpl := range mgr.Config().Backends.External {
					name := firstWord(tmpl)
					path, err := exec.LookPath(name)
					if err != nil {
						fmt.Fprintf(out, "    %-8s template=%q  binary=%q NOT FOUND ON $PATH\n", scheme, tmpl, name)
					} else {
						fmt.Fprintf(out, "    %-8s template=%q  binary=%s\n", scheme, tmpl, path)
					}
				}
			} else {
				fmt.Fprintln(out, "  external     (none configured — add under backends.external in config)")
			}

			_ = context.Background() // reserved for future async probes
			return nil
		},
	}
}

func firstWord(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}
