# Cinova

AI-powered movie and TV discovery. Cinova surfaces what to watch next through a proprietary scoring system, natural language search, and a knowledge graph that maps influences, themes, and moods across the entire film and television catalog.

## What is Cinova?

Cinova is a mobile-first entertainment discovery app that goes beyond simple recommendations. It combines:

- **CinovaScore** — a Bayesian-weighted rating blended with graph prestige signals, scored 0-100
- **Natural Language Search** — ask "a slow-burn thriller set in Scandinavia" and get results via Axon (Claude)-powered NL-to-Cypher translation
- **Influence Graph** — a Neo4j knowledge graph mapping director influences sourced from Wikidata, connecting Kubrick to his descendants, Kurosawa to his
- **AI Enrichment** — themes, moods, and discovery tags extracted by Claude from plot summaries
- **Streaming Availability** — real-time watch provider data with deep links to apps and JustWatch fallback

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend API | Go 1.23, modular monolith |
| Graph Database | Neo4j Community Edition |
| Relational DB | PostgreSQL 16 |
| Cache | Valkey (Redis-compatible) |
| AI | Axon (Claude claude-opus-4-6) via internal API |
| Data Source | TMDB API v3 + Wikidata SPARQL |
| Mobile | React Native + Expo (Expo Router) |
| Infrastructure | K3s + Flux GitOps |
| Container Registry | GHCR (`ghcr.io/foundrylab-app/cinova`) |

## Repository Structure

```
cinova/
├── backend/                  # Go backend (API server + ingestion pipeline)
│   ├── cmd/
│   │   ├── api/              # HTTP API server entrypoint
│   │   └── ingestion/        # Data ingestion pipeline entrypoint
│   ├── internal/
│   │   ├── auth/             # JWT sessions, anonymous + registered users
│   │   ├── config/           # Environment-based configuration
│   │   ├── enrichment/       # Axon AI theme/mood extraction
│   │   ├── graph/            # Neo4j repositories (movies, shows, people)
│   │   ├── models/           # Shared domain types
│   │   ├── scoring/          # CinovaScore computation
│   │   ├── search/           # NL search → Cypher translation
│   │   ├── store/            # PostgreSQL + Valkey stores
│   │   ├── streaming/        # Watch provider data + deep links
│   │   ├── tmdb/             # TMDB API client
│   │   └── wikidata/         # Wikidata SPARQL client
│   ├── Dockerfile            # API server image
│   ├── Dockerfile.ingestion  # Ingestion pipeline image
│   └── go.mod
├── mobile/                   # React Native + Expo mobile app
│   ├── app/                  # Expo Router file-based routes
│   ├── components/           # Shared UI components
│   ├── hooks/                # Custom React hooks
│   └── package.json
├── deploy/                   # Kubernetes manifests (Flux-managed)
│   └── cinova/               # Namespace, deployments, services, ingress
└── create-github-issues.sh   # Bootstrap GitHub project issues
```

## Setup

### Backend

**Prerequisites:** Go 1.23+, a running Neo4j instance, PostgreSQL, and Valkey/Redis.

```bash
cd backend

# Copy and configure environment
cp .env.example .env
# Edit .env with your TMDB_API_KEY, database URLs, Axon URL, etc.

# Run the API server (default :8080)
go run ./cmd/api

# Run a delta ingestion pass (syncs recent TMDB changes)
go run ./cmd/ingestion --mode=delta --country=US

# Run a full bulk ingestion (first-time setup, takes several hours)
go run ./cmd/ingestion --mode=bulk --country=US
```

### Mobile

**Prerequisites:** Node.js 20+, Expo CLI.

```bash
cd mobile

npm install

# Start Expo development server
npx expo start

# Run on iOS simulator
npx expo run:ios

# Run on Android emulator
npx expo run:android
```

## CinovaScore

CinovaScore is a 0–100 quality signal that replaces raw star ratings.

**Formula:**

```
bayesian = (v * R + m * C) / (v + m)
score    = ((bayesian / 10) * 0.8 + graphPrestige * 0.2) * 100
```

Where:
- `v` = vote count (from TMDB)
- `R` = vote average 0–10 (from TMDB)
- `m` = minimum votes threshold (1000) — low-vote titles regress toward the mean
- `C` = corpus mean rating (~6.5)
- `graphPrestige` = normalised 0–1 PageRank-like signal from the influence graph

**Tiers:**

| Score | Tier |
|-------|------|
| 85–100 | Masterpiece |
| 75–84 | Excellent |
| 65–74 | Great |
| 55–64 | Good |
| 45–54 | Average |
| 0–44 | Below Average |

The Bayesian adjustment prevents a film with 12 votes averaging 10.0 from outranking *The Godfather*. The graph prestige component rewards culturally significant works that have influenced many other directors.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                   Mobile App (Expo)                      │
│  Home  │  Discover  │  Search  │  Watchlist  │  Profile  │
└─────────────────────────┬────────────────────────────────┘
                          │ HTTPS / JWT
                          ▼
┌──────────────────────────────────────────────────────────┐
│            Go API  (api.cinova.openova.io)                │
│                                                          │
│  /auth/*   /trending   /recommend   /search              │
│  /movie/*  /tv/*       /me/*        /person/*            │
│                     │                                    │
│          ┌──────────┼──────────┐                         │
│          ▼          ▼          ▼                         │
│    PostgreSQL     Neo4j      Redis                        │
│    (users,       (graph,     (search 1h,                 │
│     content,     scores,     trending 15m)               │
│     watchlist)   themes)                                 │
└──────────────────────────────────────────────────────────┘
           ▲
           │  Axon (Claude)
           │  NL → Cypher translation
           │  Theme/Mood extraction
           ▼
┌──────────────────────────────────────────────────────────┐
│              Go Ingestion (background)                    │
│                                                          │
│  TMDB Bulk Load ──► PostgreSQL (movies, providers)       │
│  TMDB Delta Sync ─► PostgreSQL + Neo4j                   │
│  Wikidata SPARQL ─► Neo4j (influence graph)              │
│  Provider Sync ───► PostgreSQL (every 6h CronJob)        │
│  Theme Extraction ► Neo4j (nightly, via Axon)            │
└──────────────────────────────────────────────────────────┘

                     GitOps
┌──────────────────────────────────────────────────────────┐
│  GitHub Actions                                          │
│  backend/** → build → push GHCR → update SHA manifest   │
│  mobile/**  → expo export check                         │
│                     ▼                                    │
│  Flux CD → K3s (namespace: cinova)                       │
│    neo4j, postgres, redis, api, ingestion CronJobs       │
└──────────────────────────────────────────────────────────┘
```

## CI/CD Pipeline

```
Push to backend/**
  → build-api       (go build + docker build/push to GHCR)
  → build-ingestion (same for ingestion image)
  → deploy          (checkout openova-private, sed SHA into
                     clusters/contabo-mkt/apps/cinova/services/api.yaml,
                     commit + push → Flux reconciles in ~1 min)

Push to mobile/**
  → expo export check (static validation, no credentials needed)
  → EAS builds are triggered manually for App Store / Play Store releases
```

Images: `ghcr.io/foundrylab-app/cinova/api:<sha>` and `ghcr.io/foundrylab-app/cinova/ingestion:<sha>`

Required secrets in repo settings:
- `GITHUB_TOKEN` — auto-provided, used for GHCR push
- `OPENOVA_PRIVATE_PAT` — PAT with write access to `openova-io/openova-private`

## API Reference

All endpoints require `Authorization: Bearer <jwt>` unless noted.

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/session` | None | Create anonymous session |
| `POST` | `/api/v1/auth/signup` | Anon JWT | Register + migrate anonymous session |
| `POST` | `/api/v1/auth/login` | None | Email/password login |
| `POST` | `/api/v1/auth/refresh` | Refresh token | Rotate refresh token |
| `POST` | `/api/v1/auth/logout` | JWT | Revoke refresh token |

### Discovery

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/trending?type=movie\|tv\|all&country=US` | Trending by country |
| `GET` | `/api/v1/recommend?type=movie\|tv\|all&limit=20` | Personalized recommendations |
| `GET` | `/api/v1/search?q=...` | Natural language AI search |
| `GET` | `/api/v1/movie/{id}?country=US` | Full movie detail + streaming |
| `GET` | `/api/v1/tv/{id}?country=US` | Full TV show detail + streaming |
| `GET` | `/api/v1/person/{id}` | Director/actor detail + filmography |

### User

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/me/watchlist?type=all&status=saved` | User watchlist |
| `POST` | `/api/v1/me/rate` | Rate content (1–10) |
| `POST` | `/api/v1/me/save` | Save to watchlist |
| `POST` | `/api/v1/me/dismiss` | Dismiss from recommendations |
| `PATCH` | `/api/v1/me/country` | Update country preference |
| `POST` | `/api/v1/me/push-token` | Register push notification token |

### Infrastructure

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness (DB + cache check) |

## GitHub Issues

To bootstrap all project issues on the `foundrylab-app/cinova` repository:

```bash
chmod +x create-github-issues.sh
./create-github-issues.sh
```

This creates 48 issues across 9 phases with appropriate labels.

## License

Proprietary. All rights reserved. © 2026 FoundryLab.

Unauthorized copying, distribution, or use of this software is strictly prohibited.
