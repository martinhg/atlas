# AtlasOS

**Engineering Intelligence Platform** — Map your entire software ecosystem and know what breaks before you break it.

AtlasOS gives engineering teams a living map of their repositories, dependencies, services, and teams. It answers the questions that slow organizations down:

- **"What depends on this?"** — Instant dependency graph across all repos
- **"What breaks if we update X?"** — Impact analysis with blast radius
- **"Who owns this?"** — Automatic ownership detection
- **"What should we fix first?"** — Risk-prioritized vulnerabilities with organizational context

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+
- pnpm 11+ (`npm i -g pnpm`)
- Docker

### Setup

```bash
# Start PostgreSQL
make dev-up

# Run the API server
make run-server

# In another terminal, run the frontend
cd web && pnpm install && pnpm dev
```

The API runs at `http://localhost:8080` and the frontend at `http://localhost:5173`.

## Architecture

```
atlas/
├── cmd/
│   ├── atlas-server/     # API server (BSL 1.1)
│   └── atlas/            # CLI (Apache 2.0)
├── internal/             # Backend modules
│   ├── auth/             # Authentication & authorization
│   ├── org/              # Organization management
│   ├── catalog/          # Service catalog
│   ├── ingest/           # Data ingestion pipeline
│   │   └── parsers/      # Dependency parsers (Apache 2.0)
│   ├── impact/           # Impact analysis engine
│   └── search/           # Full-text search
├── web/                  # React frontend (BSL 1.1)
├── migrations/           # SQL migrations
└── deploy/               # Docker & infrastructure
```

## CLI

The AtlasOS CLI is open source (Apache 2.0) and can be used standalone:

```bash
# Install
go install github.com/nesbite/atlas/cmd/atlas@latest

# Scan current directory
atlas scan
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go |
| Frontend | React + Vite + TypeScript |
| Database | PostgreSQL 16 |
| Graph | Apache AGE (PostgreSQL extension) |
| Search | PostgreSQL full-text search |
| Jobs | River (PostgreSQL-backed) |

## License

AtlasOS uses a dual-license model:

- **CLI and parsers** — [Apache License 2.0](cmd/atlas/LICENSE)
- **Server and web app** — [Business Source License 1.1](LICENSE)

See [LICENSE](LICENSE) for details.

## Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) before submitting a PR.

## Security

To report a vulnerability, please see our [Security Policy](SECURITY.md).
