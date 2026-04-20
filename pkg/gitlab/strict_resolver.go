package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// ErrHostMismatch is returned when the API response URL host differs from the
// host recorded in the server's config.
var ErrHostMismatch = errors.New("configured server host does not match API host")

// StrictResolver picks a GitLab client based strictly on .gmcprc. No fallbacks,
// no host-based matching, no default-server cascade. On first use per session
// per server it verifies the API host matches the configured host and caches
// the pass/fail.
type StrictResolver struct {
	pool         *ClientPool
	serverHosts  map[string]string
	logger       *log.Logger
	mu           sync.Mutex
	verifiedOK   map[string]bool
	verifiedFail map[string]error
}

// NewStrictResolver returns a resolver. serverHosts maps each configured
// server name to its expected host URL (from global config).
func NewStrictResolver(pool *ClientPool, serverHosts map[string]string, logger *log.Logger) *StrictResolver {
	return &StrictResolver{
		pool:         pool,
		serverHosts:  serverHosts,
		logger:       logger,
		verifiedOK:   map[string]bool{},
		verifiedFail: map[string]error{},
	}
}

// Resolve returns (client, serverName, error). It NEVER falls back.
func (r *StrictResolver) Resolve(ctx context.Context) (*gl.Client, string, error) {
	cfg, _, err := FindProjectConfig()
	if err != nil {
		return nil, "", fmt.Errorf("strict resolver: failed to read .gmcprc: %w", err)
	}
	if cfg == nil {
		return nil, "", errors.New("strict resolver: no project configured — run 'gitlab-mcp-server project init' or pass --server")
	}
	if cfg.Server == "" {
		return nil, "", errors.New("strict resolver: .gmcprc is missing required 'server' field — re-run 'gitlab-mcp-server project init'")
	}

	client, err := r.pool.GetClient(cfg.Server)
	if err != nil {
		configured := make([]string, 0, len(r.serverHosts))
		for n := range r.serverHosts {
			configured = append(configured, n)
		}
		return nil, "", fmt.Errorf("strict resolver: server %q not configured; configured servers: %s",
			cfg.Server, strings.Join(configured, ", "))
	}

	if err := r.verifyHost(ctx, cfg.Server, client); err != nil {
		return nil, "", err
	}
	return client, cfg.Server, nil
}

// GetClientFn adapts StrictResolver to the GetClientFn signature used by
// existing tool handlers.
func (r *StrictResolver) GetClientFn() GetClientFn {
	return func(ctx context.Context) (*gl.Client, error) {
		c, _, err := r.Resolve(ctx)
		return c, err
	}
}

func (r *StrictResolver) verifyHost(ctx context.Context, name string, client *gl.Client) error {
	r.mu.Lock()
	if r.verifiedOK[name] {
		r.mu.Unlock()
		return nil
	}
	if prev, ok := r.verifiedFail[name]; ok {
		r.mu.Unlock()
		return prev
	}
	r.mu.Unlock()

	wantHost := r.serverHosts[name]
	if wantHost == "" {
		return fmt.Errorf("strict resolver: no host recorded for server %q", name)
	}

	_, resp, err := client.Users.CurrentUser(gl.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("strict resolver: host verification call failed: %w", err)
	}
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return errors.New("strict resolver: host verification did not receive a response URL")
	}
	wantURL, err := url.Parse(wantHost)
	if err != nil {
		return fmt.Errorf("strict resolver: configured host %q is not a valid URL: %w", wantHost, err)
	}
	gotHost := resp.Request.URL.Host
	if wantURL.Host != gotHost {
		mismatch := fmt.Errorf("%w: configured %q but API responded from %q",
			ErrHostMismatch, wantURL.Host, gotHost)
		r.mu.Lock()
		r.verifiedFail[name] = mismatch
		r.mu.Unlock()
		return mismatch
	}
	r.mu.Lock()
	r.verifiedOK[name] = true
	r.mu.Unlock()
	return nil
}
