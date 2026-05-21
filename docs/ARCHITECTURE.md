# Architecture

> **WHAT**: How Cinova is built — components, surfaces, data flow, computation.
> **AUTHORITY**: 📐 Canon. If something in code contradicts this doc, fix the doc OR fix the code — don't let drift compound.
> **POINTER**: Engineering principles live in [PRINCIPLES.md](PRINCIPLES.md) and `~/.claude/CLAUDE.md` (user-global). DoD lives in [DOD.md](DOD.md).

## What Cinova is

Cinova is a mobile-first AI-powered movie and TV discovery app. The product surfaces:

- **CinovaScore** — a Bayesian-weighted rating blended with graph prestige signals, scored 0–100.
- **Natural-language search** — ask "a slow-burn thriller set in Scandinavia" and get results via Axon (Claude) NL-to-Cypher translation.
- **Influence Graph** — a Neo4j knowledge graph mapping director influences sourced from Wikidata, connecting Kubrick to his descendants, Kurosawa to his.
- **AI Enrichment** — themes, moods, and discovery tags extracted by Claude (via Axon) from plot summaries.
- **Streaming Availability** — real-time watch-provider data with deep links to apps and JustWatch fallback.

## Tech stack

| Layer | Technology |
|---|---|
| Backend API | Go 1.23, modular monolith |
| Graph database | Neo4j Community Edition |
| Relational DB | PostgreSQL 16 |
| Cache | Valkey (Redis-compatible) |
| AI gateway | Axon — internal Claude API gateway (`claude-opus-4-6`) |
| Data sources | TMDB API v3 + Wikidata SPARQL + YouTube (vertical trailers) |
| Mobile | React Native 0.81 + Expo SDK 54 (expo-router) |
| Infrastructure | K3s + Flux GitOps |
| Container registry | GHCR (`ghcr.io/foundrylab-app/cinova`) |
| Deploy target | `contabo-mkt` K3s cluster (legacy Sovereign instance) |

## Repository layout

```
cinova/
├── backend/                        # Go backend (API server + ingestion pipeline)
│   ├── cmd/
│   │   ├── api/                    # HTTP API server entrypoint
│   │   └── ingestion/              # Data ingestion pipeline entrypoint
│   ├── internal/
│   │   ├── auth/                   # JWT sessions, anonymous + registered users
│   │   ├── chat/                   # AI chat session state + history
│   │   ├── config/                 # Environment-based configuration
│   │   ├── enrichment/             # Axon AI theme/mood extraction
│   │   ├── graph/                  # Neo4j repositories (movies, shows, people)
│   │   ├── handlers/               # HTTP handlers (movie, person, chat, etc.)
│   │   ├── langflow/               # Langflow integration (workflow engine)
│   │   ├── langfuse/               # Langfuse tracing for LLM calls
│   │   ├── models/                 # Shared domain types
│   │   ├── scoring/                # CinovaScore computation
│   │   ├── search/                 # NL search → Cypher translation
│   │   ├── store/                  # PostgreSQL + Valkey stores
│   │   ├── streaming/              # Watch-provider data + deep links
│   │   ├── tmdb/                   # TMDB API client
│   │   ├── wikidata/               # Wikidata SPARQL client
│   │   ├── workflow/               # Workflow orchestration
│   │   └── youtube/                # YouTube vertical-trailer ingestion
│   ├── migrations/                 # SQL migrations (NNN_*.sql)
│   ├── Dockerfile                  # API server image
│   ├── Dockerfile.ingestion        # Ingestion pipeline image
│   └── go.mod
├── mobile/                         # React Native + Expo mobile app
│   ├── app/                        # Expo Router file-based routes
│   │   ├── (tabs)/                 # Tab navigator routes
│   │   ├── auth/                   # Login + signup
│   │   ├── movie/                  # Movie detail
│   │   └── person/                 # Person detail
│   ├── components/                 # Shared UI components (+ `ui/` primitives)
│   ├── services/                   # API + analytics clients
│   ├── store/                      # Client state (auth, watchlist, etc.)
│   ├── constants/                  # Theme + constants
│   ├── types/                      # TypeScript types
│   ├── assets/                     # Fonts, images
│   ├── store-assets/               # App-store metadata (privacy policy, icons)
│   └── package.json
├── catalog/                        # Static catalog assets (vertical-trailers.html)
├── data-samples/                   # Reference JSON samples
├── deploy/                         # (reserved) — production manifests live in openova-private
└── create-github-issues.sh         # Bootstrap GitHub project issues
```

> Production Kubernetes manifests live in [`openova-io/openova-private`](https://github.com/openova-io/openova-private) at `clusters/contabo-mkt/apps/cinova/`. This repo's `deploy/` directory is reserved for blueprint-style packaging once cinova migrates to a Sovereign Blueprint.

## Runtime topology

> Source: previously the "Architecture" + "CI/CD Pipeline" sections of `README.md` (merged here on 2026-05-21).

```
┌──────────────────────────────────────────────────────────┐
│                   Mobile App (Expo)                      │
│  Home  │  Discover  │  Search  │  Watchlist  │  Profile  │
└─────────────────────────┬────────────────────────────────┘
                          │ HTTPS / JWT
                          ▼
┌──────────────────────────────────────────────────────────┐
│            Go API  (api.cinova.openova.io)               │
│                                                          │
│  /auth/*   /trending   /recommend   /search              │
│  /movie/*  /tv/*       /me/*        /person/*            │
│                     │                                    │
│          ┌──────────┼──────────┐                         │
│          ▼          ▼          ▼                         │
│    PostgreSQL     Neo4j      Valkey                      │
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
│              Go Ingestion (background)                   │
│                                                          │
│  TMDB Bulk Load ──► PostgreSQL (movies, providers)       │
│  TMDB Delta Sync ─► PostgreSQL + Neo4j                   │
│  Wikidata SPARQL ─► Neo4j (influence graph)              │
│  Provider Sync ───► PostgreSQL (every 6h CronJob)        │
│  Theme Extraction ► Neo4j (nightly, via Axon)            │
│  Vertical Trailers ► PostgreSQL (YouTube portrait keys)  │
└──────────────────────────────────────────────────────────┘
```

GitOps:

```
┌──────────────────────────────────────────────────────────┐
│  GitHub Actions                                          │
│  backend/** → build → push GHCR → update SHA manifest    │
│  mobile/**  → tsc + expo export check                    │
│                     ▼                                    │
│  Flux CD → K3s (namespace: cinova) on contabo-mkt        │
│    neo4j, postgres, valkey, api, ingestion CronJobs      │
└──────────────────────────────────────────────────────────┘
```

## CinovaScore — computation

> Source: previously the "CinovaScore" section of `README.md` (merged here on 2026-05-21). Glossary terms for `Bayesian rating`, `graph prestige`, and the score tiers live in [GLOSSARY.md](GLOSSARY.md).

CinovaScore is a 0–100 quality signal that replaces raw star ratings.

```
bayesian = (v * R + m * C) / (v + m)
score    = ((bayesian / 10) * 0.8 + graphPrestige * 0.2) * 100
```

Where:

- `v` = vote count (from TMDB)
- `R` = vote average 0–10 (from TMDB)
- `m` = minimum-votes threshold (1000) — low-vote titles regress toward the mean
- `C` = corpus mean rating (~6.5)
- `graphPrestige` = normalised 0–1 PageRank-like signal from the influence graph

The Bayesian adjustment prevents a film with 12 votes averaging 10.0 from outranking *The Godfather*. The graph-prestige component rewards culturally significant works that have influenced many other directors.

Score tiers and per-user scoring-profile presets (Mainstream / Cinephile / Arthouse / Blockbuster / AwardSeason) are listed in [GLOSSARY.md](GLOSSARY.md).

## API surface

> Source: previously the "API Reference" section of `README.md` (merged here on 2026-05-21). All endpoints require `Authorization: Bearer <jwt>` unless explicitly marked `None`.

### Auth

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/session` | None | Create anonymous session |
| `POST` | `/api/v1/auth/signup` | Anon JWT | Register + migrate anonymous session |
| `POST` | `/api/v1/auth/login` | None | Email/password login |
| `POST` | `/api/v1/auth/refresh` | Refresh token | Rotate refresh token |
| `POST` | `/api/v1/auth/logout` | JWT | Revoke refresh token |

### Discovery

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/trending?type=movie\|tv\|all&country=US` | Trending by country |
| `GET` | `/api/v1/recommend?type=movie\|tv\|all&limit=20` | Personalised recommendations |
| `GET` | `/api/v1/search?q=...` | Natural-language AI search |
| `GET` | `/api/v1/movie/{id}?country=US` | Full movie detail + streaming |
| `GET` | `/api/v1/tv/{id}?country=US` | Full TV-show detail + streaming |
| `GET` | `/api/v1/person/{id}` | Director/actor detail + filmography |

### User

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/me/watchlist?type=all&status=saved` | User watchlist |
| `POST` | `/api/v1/me/rate` | Rate content (1–10) |
| `POST` | `/api/v1/me/save` | Save to watchlist |
| `POST` | `/api/v1/me/dismiss` | Dismiss from recommendations |
| `PATCH` | `/api/v1/me/country` | Update country preference |
| `POST` | `/api/v1/me/push-token` | Register push-notification token |

### Infrastructure

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness (DB + cache check) |

## Caching strategy

| Surface | TTL | Store |
|---|---|---|
| Search results | 1h | Valkey |
| Trending feed | 15m | Valkey |
| Movie/TV detail | request-scope | none (Neo4j/PG queries) |
| Provider data | 6h refresh (CronJob) | PostgreSQL |
