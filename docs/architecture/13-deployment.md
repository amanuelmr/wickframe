# 13 · Deployment, Configuration and Upgrades

| | |
|---|---|
| **Status** | Accepted |
| **Last reviewed** | 2026-11-13 |
| **Requirements** | FR-OPS-01…07, NFR-UX-01, NFR-MAINT-03 |
| **Decisions** | [ADR-0003](../adr/0003-go-modular-monolith.md), [ADR-0009](../adr/0009-desktop-shell.md), [ADR-0015](../adr/0015-supply-chain-and-release.md) |

## 1. Distribution channels

All artifacts are distributed openly through GitHub (Releases and the GitHub
Container Registry) and the project website, which links to them. There is no app
store distribution ([vision](../product/vision.md#5-non-goals)).

| Channel | Artifact | Audience | Milestone |
|---|---|---|---|
| GitHub Releases | `wickframe_<ver>_<os>_<arch>.tar.gz/.zip` plus checksums, signatures, SBOMs | Everyone | M1 |
| Container registry | `ghcr.io/amanuelmr/wickframe:<ver>` (multi-arch, distroless, non-root) | Server and homelab | M1 |
| Docker Compose | `deploy/compose/docker-compose.yml` (Wickframe + PostgreSQL), plus a SQLite-only variant | Server and homelab | M1 |
| Desktop installers | Wails v3 packages (Windows, macOS, Linux), attached to GitHub Releases | P1 users | M3 |
| Helm chart | `deploy/helm/wickframe` | Kubernetes users | M3+ |
| Package managers | Homebrew tap, Scoop, AUR (community) | Convenience | Post-M3 |

## 2. Configuration

### 2.1 Sources and precedence

Precedence, highest first: **command-line flags → environment variables
(`WICKFRAME_…`) → config file (`wickframe.yaml`) → built-in defaults**. Loaded
with `knadh/koanf` v2; the CLI uses `spf13/cobra`.

* Environment mapping: `server.listen` ↔ `WICKFRAME_SERVER_LISTEN`. Lists are
  comma-separated.
* `*_FILE` variants (for example `WICKFRAME_SECRET_KEY_FILE`) read secrets from
  files, for Docker and Kubernetes secrets.
* `wickframe config validate` and `wickframe config print --redacted` exist.
* **Unknown keys are an error** (catches typos). Every key has a documented
  default and validation.

### 2.2 Key settings (initial)

```yaml
mode: personal              # personal | server
server:
  listen: "127.0.0.1:7317"  # personal default; server mode default ":7317"
  publicURL: ""             # required in server mode (used for Origin checks, links in notifications)
  trustedProxies: []        # CIDRs allowed to set X-Forwarded-*
  targets: ["all"]          # ADR-0003 roles
  admin:
    listen: ""              # e.g. "127.0.0.1:7318" for /metrics and pprof
database:
  driver: sqlite            # sqlite | postgres
  sqlite:
    path: ""                # default: <dataDir>/wickframe.db
  postgres:
    dsn: ""                 # or WICKFRAME_DATABASE_POSTGRES_DSN_FILE
dataDir: ""                 # default per OS: $XDG_DATA_HOME/wickframe (~/.local/share/wickframe),
                            # ~/Library/Application Support/Wickframe, %LocalAppData%\Wickframe
security:
  secretKeyFile: ""         # required in server mode (ADR §11.5)
auth:
  enabled: false            # forced true in server mode
  oidc: {}                  # issuer, clientID, clientSecretFile, scopes, roleClaim…
log:
  level: info
  format: auto
telemetry:
  otlp:
    endpoint: ""            # empty = tracing off
storage:
  retention: { "1m": "365d", "3m": "365d", "5m": "5y", "15m": "5y", "30m": "5y" }
  timescale: { enabled: auto }
features: {}                # feature flags, all default off
```

The full, generated reference lives in the user documentation (`wickframe config
docs` emits Markdown from the schema).

## 3. Container image

* Base: `gcr.io/distroless/static-debian13:nonroot` (UID 65532). No shell.
* `ENTRYPOINT ["/wickframe"]`, `CMD ["serve"]`. Data directory: `/data` (a volume).
* Exposes 7317. Health check: the Compose `healthcheck` calls
  `/wickframe healthcheck`, a subcommand that requests `/livez` (there is no curl
  in distroless).
* Read-only root filesystem compatible; writes only to `/data` and `/tmp` (tmpfs).
* Labels: OCI annotations (`org.opencontainers.image.*`), including the source,
  version and license.

## 4. Reference Docker Compose (server mode)

```yaml
services:
  wickframe:
    image: ghcr.io/amanuelmr/wickframe:0.1   # pin a digest in production
    restart: unless-stopped
    environment:
      WICKFRAME_MODE: server
      WICKFRAME_SERVER_PUBLICURL: https://charts.example.com
      WICKFRAME_DATABASE_DRIVER: postgres
      WICKFRAME_DATABASE_POSTGRES_DSN_FILE: /run/secrets/pg_dsn
      WICKFRAME_SECURITY_SECRETKEYFILE: /run/secrets/wf_key
    secrets: [pg_dsn, wf_key]
    volumes: [wickframe-data:/data]
    ports: ["127.0.0.1:7317:7317"]          # put a TLS reverse proxy in front
    depends_on:
      db: { condition: service_healthy }
    read_only: true
    tmpfs: [/tmp]
    security_opt: ["no-new-privileges:true"]
    cap_drop: [ALL]
  db:
    image: postgres:18
    restart: unless-stopped
    environment:
      POSTGRES_DB: wickframe
      POSTGRES_USER: wickframe
      POSTGRES_PASSWORD_FILE: /run/secrets/pg_password
    secrets: [pg_password]
    volumes: [pg-data:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U wickframe"]
      interval: 10s
secrets:
  pg_dsn: { file: ./secrets/pg_dsn }
  pg_password: { file: ./secrets/pg_password }
  wf_key: { file: ./secrets/wickframe.key }
volumes:
  wickframe-data: {}
  pg-data: {}
```

This is illustrative. The maintained file lives in `deploy/compose/` and is
exercised by CI.

## 5. Upgrades

* **Supported path:** any version to the next minor version, directly. Skipping
  minors is supported when release notes don't say otherwise; CI tests N−1 → N.
* **Procedure:** back up (`wickframe backup`), replace the binary or image, start.
  Migrations run automatically under a lock. `/readyz` turns ready after
  migrations complete.
* **Rollback:** restore the binary and image of N−1. Because of the
  expand/contract migration discipline ([storage §3](08-storage.md#3-migrations)),
  N−1 can run against a schema migrated by N within the same compatibility window,
  as stated in the release notes. Otherwise, restore the backup.
* The desktop app updates in-app from signed releases (Wails updater, ADR-0009),
  and always backs up the SQLite file first.

## 6. Resource guidance

| Deployment | CPU | RAM | Disk |
|---|---|---|---|
| Personal (desktop or CLI) | any modern | 150–300 MB | 100 MB + candles (~60 MB per 1m instrument-year) |
| Homelab server (≤ 5 users, ≤ 100 instruments) | 1–2 vCPU | 512 MB–1 GB | 5–20 GB |
| Reference server (NFR-PERF-04 load) | 2 vCPU | 4 GB | 50+ GB SSD |
