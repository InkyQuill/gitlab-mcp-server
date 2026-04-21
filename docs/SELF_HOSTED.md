# Self-hosted GitLab

The server works with any GitLab.com or self-managed instance that speaks API v4. Most self-hosted setups need nothing beyond pointing at the right host; the notes below cover the handful of cases that do.

## Pointing at your instance

Preferred: add it to the global config as a named server.

```bash
gitlab-mcp-server config add company --host https://gitlab.company.com
gitlab-mcp-server config validate
```

Legacy (env-var fallback, single server, plaintext token):

```bash
GITLAB_HOST=https://gitlab.company.com GITLAB_TOKEN=glpat-… gitlab-mcp-server stdio
```

Docker:

```bash
docker run -i --rm \
  -e GITLAB_TOKEN=glpat-… \
  -e GITLAB_HOST=https://gitlab.company.com \
  gitlab-mcp-server:latest
```

## Tokens

Any of Personal, Project, or Group Access Token works. Required scopes: **`api`** for full functionality, or `read_api` + `read_repository` for read-only. Create them in User / Project / Group → **Settings → Access Tokens**.

## TLS and private CAs

The server uses Go's default TLS stack and trusts the system root store.

- **Self-signed or private-CA certificates:** add the CA to the OS trust store, or point Go at a bundle:

  ```bash
  export SSL_CERT_FILE=/path/to/ca-bundle.pem
  gitlab-mcp-server stdio
  ```

- **Docker:** either bake your CA into a custom image, or mount it:

  ```bash
  docker run -i --rm \
    -v /etc/ssl/certs/company-ca.crt:/etc/ssl/certs/company-ca.crt:ro \
    -e SSL_CERT_FILE=/etc/ssl/certs/company-ca.crt \
    -e GITLAB_TOKEN=glpat-… \
    -e GITLAB_HOST=https://gitlab.company.com \
    gitlab-mcp-server:latest
  ```

There is no flag to skip TLS verification. Fix the trust chain instead.

## Multiple instances side by side

Add as many servers as you need; tool calls can target a specific one via the `server` argument.

```bash
gitlab-mcp-server config add work     --host https://gitlab.company.com --token-ref op://Work/gitlab/token
gitlab-mcp-server config add personal --host https://gitlab.com
gitlab-mcp-server config add mirror   --host https://gitlab.internal --read-only
```

Details: [MULTI_SERVER_SETUP.md](MULTI_SERVER_SETUP.md). For stricter per-call routing, set `GITLAB_MCP_STRICT_RESOLVER=1` (see [CONFIGURATION.md](CONFIGURATION.md)).

## Per-project pinning

In a repository managed on your self-hosted instance:

```bash
cd your-project
gitlab-mcp-server project init
```

This auto-detects from `.git/config` and writes `.gmcprc`. When multiple servers are configured, set `"server": "work"` in `.gmcprc` (or run `project init` after making the matching server the default). Schema details: [PROJECT_CONFIG.md](PROJECT_CONFIG.md).

## Troubleshooting

**`x509: certificate signed by unknown authority`** — install the CA or set `SSL_CERT_FILE` as above.

**`401 Unauthorized` on a self-hosted instance that worked on GitLab.com** — tokens are per-instance; create a new one on the self-hosted side. Confirm the token still has the right scopes.

**Wrong host after `config add`** — Use `config validate`; the command rewrites `userId`/`username` against the current host so a mismatched host fails fast.

**`server "X" not found` in tool calls** — names are case-sensitive and must match the global config exactly. List them with `config list`.

## See also

- [INSTALLATION.md](INSTALLATION.md)
- [CONFIGURATION.md](CONFIGURATION.md)
- [TOKEN_MANAGEMENT.md](TOKEN_MANAGEMENT.md)
- [MULTI_SERVER_SETUP.md](MULTI_SERVER_SETUP.md)
- [PROJECT_CONFIG.md](PROJECT_CONFIG.md)
