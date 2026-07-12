# gateway — Eternal Orbit Labs API

Go HTTP API for auth, sessions, and user management (Gin, PostgreSQL, Redis).

## Getting Started

### Dependency setup

- **Go** 1.25+
- **PostgreSQL** 16 *(not required in local-memory mode)*
- **Redis** 7 *(not required in local-memory mode)*
- **Docker** *(not required in local-memory mode)*

Copy environment config:

```bash
cp .env.example .env
# Edit .env — JWT_SECRET, OAuth keys, etc.
```

### Option A — Local memory mode (no Docker required)

Set `DATABASES_MEMORY=LOCAL` in your `.env` (or export it) and run:

```bash
DATABASES_MEMORY=LOCAL make run
# or
DATABASES_MEMORY=LOCAL go run ./cmd/server
```

This starts an **embedded PostgreSQL** and an **in-memory Redis** inside the Go process.
No Docker, no external services — everything runs locally. Migrations are applied
automatically against the embedded database. Data is ephemeral and resets on restart.

### Option B — Docker databases (production-like)

Run postgres and redis:

```bash
docker compose up -d postgres redis
```

Run the gateway locally:

```bash
make run
# or
go run ./cmd/server
```

API listens on [http://localhost:8080](http://localhost:8080) by default.

Full stack (postgres + redis + gateway):

```bash
docker compose up -d
```

Dev with hot reload:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up gateway
```

Other commands:

```bash
make build    # compile binary to bin/server
make tidy     # go mod tidy
```

Migrations run automatically on server startup.

---

## Branching conventions

| Branch | Purpose |
| --- | --- |
| `main` | Stable code for release |
| `release/[version-number]` | Base branch for a development release |
| `feature/[feature-name]` | Feature branch from `release/[version-number]` |
| `fix/[feature-name]` | Hotfix after `release` has been merged to `main` |

> **Note:** Use **kebab-case** for `feature-name` (e.g. `feature/oauth-callback`, not `feature/oauth/callback`) to avoid multi-level path slashes in branch names.

---

## Commit message conventions

Every commit should start with one of these tags:

| Tag | When to use |
| --- | --- |
| `init` | Initializing a feature with base setup |
| `wip` | Work in progress when switching branches mid-task |
| `feature` | New features or modules |
| `improvements` | Updates to existing features |
| `revamp` | Substantial rework of existing logic or API shape |
| `fix` | Bug fixes |
| `ui` | Response or error payload changes that affect clients (rare in gateway) |
| `test` | Adding or updating tests |
| `config` | Config, migrations, Docker, env, or tooling changes |

**Format:**

```text
<tag>: <short description>
```

**Examples:**

```text
feature: add personal API key endpoints
fix: validate refresh token before session lookup
config: bump Dockerfile to Go 1.25
test: cover auth middleware with expired JWT
```

> **Note:** Keep these commit tags in your message and spell them exactly as listed.
