package gitlab

import (
	"context"

	gl "gitlab.com/gitlab-org/api/client-go"
	log "github.com/sirupsen/logrus"
)

// ClientResolver resolves which GitLab client to use for a given context
// It supports:
// 1. Project-specific token (from .gmcprc)
// 2. Host-based matching
// 3. Default fallback
type ClientResolver struct {
	pool          *ClientPool
	defaultServer string
	logger        *log.Logger
}

// NewClientResolver creates a new client resolver
func NewClientResolver(pool *ClientPool, defaultServer string, logger *log.Logger) *ClientResolver {
	return &ClientResolver{
		pool:          pool,
		defaultServer: defaultServer,
		logger:        logger,
	}
}

// Resolve determines which client to use based on the current context
// Resolution order:
// 1. Read .gmcprc to get tokenName
// 2. If tokenName exists, use that client
// 3. If gitlabHost in .gmcprc, find matching client by host
// 4. Fall back to defaultServer
func (cr *ClientResolver) Resolve(ctx context.Context) (*gl.Client, string, error) {
	// Try to read project config
	config, configPath, err := FindProjectConfig()
	if err != nil || config == nil {
		// No config found, use default
		cr.logger.Debugf("No project config found, using default client: %v", err)
		return cr.pool.GetDefaultClient()
	}

	cr.logger.Debugf("Found project config at %s: %+v", configPath, config)

	// Priority 1: Use tokenName from config
	if config.TokenName != "" {
		client, err := cr.pool.GetClient(config.TokenName)
		if err != nil {
			cr.logger.Warnf("Token '%s' specified in config but not found in pool, falling back to default", config.TokenName)
		} else {
			cr.logger.Debugf("Using client '%s' from project config", config.TokenName)
			return client, config.TokenName, nil
		}
	}

	// Priority 2: Match by gitlabHost
	if config.GitLabHost != "" && config.GitLabHost != "https://gitlab.com" {
		// Try to find a client that matches this host
		// Client names are either hostnames or "default"
		clientNames := cr.pool.ListClients()
		for _, name := range clientNames {
			// Check if this client's host matches
			if client, err := cr.pool.GetClient(name); err == nil {
				// We need to check if the client was created with this host
				// For now, we'll use a simple name-based matching
				// NOTE: Future improvement - store host metadata in ClientPool for better matching
				if name == config.GitLabHost {
					cr.logger.Debugf("Using client '%s' matching host %s", name, config.GitLabHost)
					return client, name, nil
				}
			}
		}

		cr.logger.Warnf("No client found matching host %s, falling back to default", config.GitLabHost)
	}

	// Priority 3: Use default server
	if cr.defaultServer != "" {
		client, err := cr.pool.GetClient(cr.defaultServer)
		if err != nil {
			cr.logger.Warnf("Default client '%s' not found, using first available", cr.defaultServer)
			return cr.pool.GetDefaultClient()
		}
		cr.logger.Debugf("Using default client '%s'", cr.defaultServer)
		return client, cr.defaultServer, nil
	}

	// Priority 4: Last resort - get any available client
	return cr.pool.GetDefaultClient()
}

// GetClientFn returns a GetClientFn function that uses the resolver
// This can be passed to tool initialization
func (cr *ClientResolver) GetClientFn() GetClientFn {
	return func(ctx context.Context) (*gl.Client, error) {
		client, name, err := cr.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		cr.logger.Debugf("Resolved to client '%s' for this request", name)
		return client, nil
	}
}

// ResolveForProject resolves the client for a specific project ID
// This is useful when you know the project ID but don't have .gmcprc context
func (cr *ClientResolver) ResolveForProject(ctx context.Context, projectID string) (*gl.Client, string, error) {
	// First try to resolve normally (may use .gmcprc)
	client, name, err := cr.Resolve(ctx)
	if err != nil {
		return nil, "", err
	}

	cr.logger.Debugf("Resolved to client '%s' for project %s", name, projectID)
	return client, name, nil
}
