# Cinova

AI-powered movie and TV discovery. Cinova surfaces what to watch next through a proprietary scoring system, natural-language search, and a knowledge graph that maps influences, themes, and moods across the entire film and television catalogue.

| Layer | Tech |
|---|---|
| Backend | Go 1.23 modular monolith — API + ingestion |
| Mobile | React Native 0.81 + Expo SDK 54 (expo-router) |
| Graph | Neo4j Community Edition |
| Relational | PostgreSQL 16 |
| Cache | Valkey |
| AI gateway | Axon (Claude `claude-opus-4-6`) |
| Deploy | K3s + Flux on `contabo-mkt` Sovereign; manifests in `openova-private` |

## Quickstart

```bash
# Backend (Go 1.23+, Neo4j, Postgres, Valkey required locally)
cd backend
cp .env.example .env             # fill TMDB_API_KEY, DB URLs, AXON_URL, JWT_SECRET
go run ./cmd/api                 # API on :8080
go run ./cmd/ingestion --mode=delta --country=US

# Mobile (Node 20+)
cd mobile
npm install
npx expo start
```

Full instructions: [docs/RUNBOOKS.md](docs/RUNBOOKS.md).

## Documentation

### 📐 Canon — read in this order

- [GLOSSARY](docs/GLOSSARY.md) — terminology + banned-terms list
- [STATUS](docs/STATUS.md) — what's built today vs design
- [ARCHITECTURE](docs/ARCHITECTURE.md) — how it works, API surface, CinovaScore formula
- [PRINCIPLES](docs/PRINCIPLES.md) — engineering rules + anti-pattern catalog
- [DOD](docs/DOD.md) — definition of done, five-gate model

### 🔧 Build + operate

- [RUNBOOKS](docs/RUNBOOKS.md) — local setup, CI/CD flow, rollback, ingestion ops
- [SECURITY](docs/SECURITY.md) — identity model, secrets policy, threat model

### 🏛️ Decision records — [docs/adr/](docs/adr/)

- [ADR index](docs/adr/README.md) (no ADRs filed yet)

### 🟢 Live state — [docs/ledger/](docs/ledger/)

- [TRUST](docs/ledger/TRUST.md) — per-surface verification ledger
- [TRACKER](docs/ledger/TRACKER.md) — open work + DoD progress snapshot

### 📚 Operator notes

- [Lessons learned](docs/lessons-learned/README.md) · [Per-incident runbooks](docs/runbooks/README.md) · [Proposals](docs/proposals/README.md) · [Sessions](docs/sessions/README.md) · [Archive](docs/archive/README.md)

## Agent orientation

If you are an agent (Claude Code or otherwise): read [CLAUDE.md](CLAUDE.md) for repo-specific orientation, then `~/.claude/CLAUDE.md` for the user-global engineering principles.

## License

Proprietary. All rights reserved. © 2026 FoundryLab.

Unauthorised copying, distribution, or use of this software is strictly prohibited.
