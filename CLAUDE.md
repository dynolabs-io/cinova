# cinova — Repo-specific Notes

> This is a product repo (FoundryLab's Cinova app). Generic OpenOva platform working principles live in `~/.claude/CLAUDE.md` (user-global).

## What this is

Cinova is a mobile-first AI-powered movie and TV discovery app. The repo contains a Go backend (modular monolith API + ingestion pipeline against TMDB and Wikidata), a React Native + Expo mobile client, and Flux-managed K8s manifests. Core surfaces are CinovaScore (Bayesian rating + graph prestige), natural-language search (Axon/Claude → Cypher), and a Neo4j influence graph.

## What lives in this repo

| Concern | Path |
|---|---|
| Go API server | `backend/cmd/api/` |
| Go ingestion pipeline | `backend/cmd/ingestion/` |
| Backend internal packages | `backend/internal/{auth,graph,scoring,search,store,tmdb,wikidata,enrichment,streaming}/` |
| Expo mobile app (Expo Router) | `mobile/app/` |
| Mobile shared UI | `mobile/components/`, `mobile/hooks/` |
| K8s manifests (Flux-managed) | `deploy/cinova/` |
| Issue bootstrap script | `create-github-issues.sh` |

## Tech stack

- Go 1.23 (backend)
- React Native 0.81 + Expo SDK 54 + expo-router (mobile)
- Neo4j Community Edition (graph)
- PostgreSQL 16 (relational)
- Valkey (cache)
- Axon (internal Claude API gateway)
- Container registry: `ghcr.io/foundrylab-app/cinova`
- Deploy: K3s + Flux on contabo-mkt (manifests live in `openova-private/clusters/contabo-mkt/apps/cinova/`)

## Development workflow

```bash
# Backend (Go 1.23+, Neo4j, Postgres, Valkey required locally)
cd backend
cp .env.example .env  # fill TMDB_API_KEY, DB URLs, AXON_URL
go run ./cmd/api                              # API server on :8080
go run ./cmd/ingestion --mode=delta --country=US

# Mobile (Node 20+)
cd mobile
npm install
npx expo start
```

## CI/CD

`backend/**` push → build API + ingestion images → push GHCR → bump SHA in `openova-private/clusters/contabo-mkt/apps/cinova/services/api.yaml` → Flux reconciles.
`mobile/**` push → expo export check; EAS builds are manual.

## Known issues

- (empty for now — populate as discovered)

## Sub-agent cap for this project

Default (per user-global) unless project owner overrides here.
