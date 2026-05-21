# Status — what's built vs design

> **WHAT**: Snapshot of what's actually shipped today vs the target shape in [ARCHITECTURE.md](ARCHITECTURE.md). Refreshed on every code-complete PR. Live work + open backlog live in [ledger/TRACKER.md](ledger/TRACKER.md).
> **AUTHORITY**: 📐 PERMANENT-refreshable. Each section dated; when reality drifts from this doc, fix the doc.
> **AS-OF**: 2026-05-21

## Pillars at a glance

| Pillar | State | Notes |
|---|---|---|
| Backend API | 🟢 Shipped + live | All routes in [ARCHITECTURE.md API surface](ARCHITECTURE.md#api-surface) exist; `/healthz` + `/readyz` live at `api.cinova.openova.io`. |
| Ingestion (TMDB delta) | 🟢 Shipped — running as daily K8s CronJob on contabo-mkt. |
| Ingestion (TMDB bulk) | 🟢 Shipped — operator runs manually for first-time / country addition. |
| Ingestion (Wikidata influence graph) | 🟢 Shipped — Neo4j populated with director influences + TV award/critic data. |
| Ingestion (Axon enrichment) | 🟢 Shipped — nightly theme/mood extraction via Axon. |
| Ingestion (vertical trailers) | 🟢 Shipped — `vertical-trailers-job` CronJob pulls portrait YouTube keys. |
| CinovaScore | 🟢 Shipped — Bayesian + graph-prestige formula computed at ingest time, served from PG. |
| Per-user scoring profile | 🟢 Shipped — Mainstream / Cinephile / Arthouse / Blockbuster / AwardSeason presets. |
| Mobile auth | 🟢 Shipped — anonymous → registered with session migration. |
| Mobile home + discover (reels) | 🟢 Shipped — vertical trailer feed, 21 titles per founder fix-train (commits 5f240fd, 7d865f9, ca307a1, 180f453). |
| Mobile movie/TV detail | 🟡 Shipped with known bugs — see open issues #78, #80, #94. |
| Mobile search (NL) | 🟢 Shipped — Axon NL→Cypher translation. |
| Mobile AI chat | 🟡 Shipped with known quality gaps — see #84 (country hardcoded), #82 (voice input pending). |
| Mobile watchlist | 🟢 Shipped — save / rate / dismiss persisted server-side. |
| Push notifications | 🟡 Backend token registration shipped; client-side notification delivery pending #49. |
| Streaming deep links | 🟡 Shipped with bugs — provider icons + open-in-app regressions (#90, #91, #46). |
| Monetisation (AdMob + RevenueCat) | 🟡 Backend integration shipped; in-app placements pending #50, #51, #52. |
| Affiliate deep links | 🟡 Pending #53. |
| Google Cast / Chromecast | 🔴 In design — #45 `status/in-progress`. |
| App Store / Play Store submission | 🔴 In progress — #54, #55, #57. EAS workflow shipped (#73). |

## Backend modules — coverage vs design

| Package | Purpose | State |
|---|---|---|
| `internal/auth` | JWT sessions, anon + registered | 🟢 |
| `internal/chat` | AI chat session + history | 🟡 (quality gaps in #84) |
| `internal/config` | Env-based config | 🟢 |
| `internal/enrichment` | Axon theme/mood extraction | 🟢 |
| `internal/graph` | Neo4j repositories | 🟢 |
| `internal/handlers` | HTTP handlers (movie, person, chat, scoring, …) | 🟢 |
| `internal/langflow` | Langflow workflow engine | 🟡 (integration shipped, walk-coverage thin) |
| `internal/langfuse` | LLM call tracing | 🟢 |
| `internal/models` | Shared types | 🟢 |
| `internal/scoring` | CinovaScore | 🟢 |
| `internal/search` | NL → Cypher | 🟢 |
| `internal/store` | PG + Valkey | 🟢 |
| `internal/streaming` | Watch providers + deep links | 🟡 (#46, #90, #91) |
| `internal/tmdb` | TMDB client | 🟢 |
| `internal/wikidata` | Wikidata SPARQL | 🟢 |
| `internal/workflow` | Workflow orchestration | 🟢 |
| `internal/youtube` | Vertical-trailer fetch | 🟢 |

## Mobile routes — coverage vs design

| Route | State |
|---|---|
| `(tabs)/index` Home | 🟢 |
| `(tabs)/discover` Reels | 🟢 (post fix-train) |
| `(tabs)/search` | 🟢 |
| `(tabs)/watchlist` | 🟢 |
| `(tabs)/profile` | 🟢 |
| `auth/login`, `auth/signup` | 🟡 (back-nav missing #85) |
| `movie/[id]` | 🟡 (#78, #80, #94) |
| `person/[id]` | 🟡 (#78) |

## Open-issue snapshot (as of 2026-05-21)

| Category | Count |
|---|---|
| Total open | ~42 |
| `status/in-progress` | 6 |
| `status/uat` | 19 |
| `status/completed` (awaiting user verification) | 7 |
| `area/mobile` | 34 |
| `area/backend` | 8 |
| `area/data` | 7 |
| `area/infra` | 2 |
| `area/ci-cd` | 2 |

Live, machine-current snapshot lives in [ledger/TRACKER.md](ledger/TRACKER.md).

## Known gaps vs target architecture

- **`deploy/` directory is empty.** Production manifests live in `openova-private/clusters/contabo-mkt/apps/cinova/`. When cinova migrates to a Sovereign Blueprint (per `openova-io/openova` Blueprint authoring guide), the manifests will move under `deploy/` as a Blueprint artifact.
- **No CHANGELOG.** Per user-global §11 rule 8, GitHub commit history + PRs are the changelog.
- **Maestro flows are smoke-only today.** `.maestro/01-launch.yaml` proves the app boots + tab bar renders. Per-pillar flows (Reels / Discover / Search / Watchlist / Profile) are scheduled to land incrementally — each per-pillar PR adds its own Maestro flow and references it from `.maestro/00-all.yaml`.
- **Bundle ID changed 2026-05-21**: `io.openova.cinova` → `io.dynolabs.cinova` (issue #100). Any existing TestFlight installs under the old ID are invalidated; first TestFlight under the new ID is `ios.yml` run #1 after the new ASC record is created.
