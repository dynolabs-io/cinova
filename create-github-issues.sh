#!/usr/bin/env bash
# create-github-issues.sh — Create all Cinova project issues on GitHub
# Prerequisites: gh auth login (with foundrylab-app org access)
# Usage: ./create-github-issues.sh

set -euo pipefail

REPO="foundrylab-app/cinova"

echo "==> Creating labels for $REPO"

# Area labels
gh label create "area/backend"  --color "0075ca" --description "Go backend API and services"         --repo "$REPO" --force
gh label create "area/mobile"   --color "e4e669" --description "React Native / Expo mobile app"     --repo "$REPO" --force
gh label create "area/infra"    --color "d93f0b" --description "Kubernetes and infrastructure"      --repo "$REPO" --force
gh label create "area/ci-cd"    --color "f9d0c4" --description "GitHub Actions and deployment"      --repo "$REPO" --force
gh label create "area/data"     --color "bfd4f2" --description "Data ingestion and enrichment jobs" --repo "$REPO" --force

# Status labels (Kanban columns)
gh label create "status/in-progress" --color "fbca04" --description "Actively being worked on"         --repo "$REPO" --force
gh label create "status/uat"         --color "c2e0c6" --description "Code done, running tests"         --repo "$REPO" --force
gh label create "status/completed"   --color "0e8a16" --description "All tests passed, awaiting review" --repo "$REPO" --force
gh label create "status/parked"      --color "cccccc" --description "Blocked or deprioritized"          --repo "$REPO" --force

echo "==> Labels created"
echo ""
echo "==> Creating issues..."

# ─── PHASE 1 — Infrastructure & Backend Foundation ────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "Deploy Neo4j Community to cinova K8s namespace" \
  --label "area/infra" \
  --body "## Goal
Deploy Neo4j Community Edition to the \`cinova\` namespace on the K3s cluster.

## Tasks
- [ ] PersistentVolumeClaim for Neo4j data (10Gi)
- [ ] Deployment manifest with neo4j:5-community image
- [ ] Service manifest (bolt: 7687, http: 7474)
- [ ] ConfigMap for neo4j.conf (heap, page cache sizing for 8GB node)
- [ ] Add to cinova kustomization.yaml
- [ ] Verify bolt connection from backend pod

## Acceptance Criteria
- Neo4j pod running and healthy
- Backend can connect via \`bolt://neo4j:7687\`
- Data persists across pod restarts"

gh issue create \
  --repo "$REPO" \
  --title "PostgreSQL schema: users, anonymous_sessions, refresh_tokens" \
  --label "area/backend" \
  --body "## Goal
Define and apply the PostgreSQL schema for the Cinova auth layer.

## Tables
\`\`\`sql
-- users: registered accounts
-- anonymous_sessions: pre-auth device sessions
-- refresh_tokens: JWT refresh token store with rotation
\`\`\`

## Tasks
- [ ] Write migration files (golang-migrate format)
- [ ] users table: id, email, password_hash, created_at, updated_at
- [ ] anonymous_sessions table: id, device_fingerprint, country, created_at, last_seen_at
- [ ] refresh_tokens table: id, user_id/session_id, token_hash, expires_at, revoked
- [ ] Indexes on email, token_hash, expires_at
- [ ] Integration test: run migrations on test DB

## Acceptance Criteria
- Migrations run cleanly up and down
- All FK constraints and indexes in place"

gh issue create \
  --repo "$REPO" \
  --title "Go backend: config, models, router skeleton" \
  --label "area/backend" \
  --body "## Goal
Bootstrap the Go backend project structure with config loading, domain models, and an HTTP router.

## Tasks
- [ ] Config struct with viper/env loading (DB URL, Neo4j URL, JWT secret, TMDB key, port)
- [ ] Domain models: Movie, TVShow, Person, Genre, WatchProvider, User, Session
- [ ] chi or stdlib router with versioned prefix \`/api/v1\`
- [ ] Middleware: logger, recoverer, request ID, CORS
- [ ] Health endpoints: GET /healthz (liveness), GET /readyz (readiness)
- [ ] Dockerfile with multi-stage build (builder → api target)
- [ ] Dockerfile.ingestion for ingestion binary

## Acceptance Criteria
- \`go build ./...\` succeeds
- GET /healthz returns 200
- Docker image builds successfully"

gh issue create \
  --repo "$REPO" \
  --title "Anonymous session auth: create session, issue JWT" \
  --label "area/backend" \
  --body "## Goal
Allow devices to use the app without registering by issuing anonymous session JWTs.

## Tasks
- [ ] POST /api/v1/auth/session — create anonymous session, return access + refresh JWTs
- [ ] JWT claims: session_id, type=anonymous, country, exp
- [ ] Middleware to validate JWT and attach session to request context
- [ ] Rate limit: 5 session creates per IP per hour
- [ ] Store session in PostgreSQL anonymous_sessions table

## Acceptance Criteria
- Fresh install gets a valid JWT with no account required
- JWT validates correctly on subsequent requests
- Invalid/expired JWT returns 401"

gh issue create \
  --repo "$REPO" \
  --title "Authenticated auth: signup, login, refresh, session merge" \
  --label "area/backend" \
  --body "## Goal
Full auth flow for registered users, including merging their anonymous session history on signup.

## Tasks
- [ ] POST /api/v1/auth/signup — email + password, merge anonymous session
- [ ] POST /api/v1/auth/login — return access + refresh JWTs
- [ ] POST /api/v1/auth/refresh — rotate refresh token
- [ ] POST /api/v1/auth/logout — revoke refresh token
- [ ] Password hashing with bcrypt (cost 12)
- [ ] Session merge: transfer ratings/watchlist from anon session to new user
- [ ] Email validation (format only, no verification email in MVP)

## Acceptance Criteria
- Signup with existing anon session preserves watchlist/ratings
- Refresh token rotation works; old token is revoked
- Logout invalidates refresh token"

gh issue create \
  --repo "$REPO" \
  --title "TMDB API client: movie details, TV details, trending, bulk IDs" \
  --label "area/data" \
  --body "## Goal
Build a Go TMDB API v3 client used by both the API server and ingestion job.

## Tasks
- [ ] HTTP client with rate limiting (40 req/s TMDB limit) and retry with backoff
- [ ] GET /movie/{id} — full movie details with credits, watch providers, similar
- [ ] GET /tv/{id} — full TV show details
- [ ] GET /trending/all/day — daily trending list
- [ ] GET /movie/changes — bulk changed IDs since timestamp (for delta sync)
- [ ] GET /configuration/countries — watch provider country list
- [ ] Struct types matching TMDB response shapes
- [ ] Unit tests with recorded fixtures

## Acceptance Criteria
- Client respects TMDB rate limits without 429 errors
- All methods return typed structs
- Unit tests pass"

gh issue create \
  --repo "$REPO" \
  --title "TMDB initial bulk load ingestion job (full mode)" \
  --label "area/data" \
  --body "## Goal
One-time ingestion job that loads the full TMDB catalog into PostgreSQL and Neo4j.

## Tasks
- [ ] Download TMDB daily export file (movie_ids, tv_series_ids)
- [ ] Fan-out worker pool (configurable concurrency, default 10)
- [ ] Fetch full details for each ID via TMDB client
- [ ] Upsert movies/shows into PostgreSQL
- [ ] Create nodes and relationships in Neo4j (Movie, Person, Genre)
- [ ] Progress logging with ETA estimate
- [ ] Idempotent: safe to re-run (upsert, not insert)
- [ ] Kubernetes Job manifest for one-time execution

## Acceptance Criteria
- Full catalog loaded (~900K movies, ~200K TV shows) without crashing
- Job is idempotent — re-running does not duplicate data
- Completion logged with total count and duration"

gh issue create \
  --repo "$REPO" \
  --title "Watch providers sync CronJob (every 6h)" \
  --label "area/data" \
  --body "## Goal
Keep watch provider availability (streaming, rental, purchase) fresh via a scheduled sync.

## Tasks
- [ ] CronJob manifest (schedule: \`0 */6 * * *\`)
- [ ] Fetch /watch/providers/movie and /watch/providers/tv for all supported countries
- [ ] Upsert provider records and movie-provider-country mappings
- [ ] Log provider count and countries processed
- [ ] Alert (log ERROR) if TMDB returns unexpected error

## Acceptance Criteria
- CronJob runs on schedule without errors
- Provider data updated in DB within 6h of TMDB changes"

gh issue create \
  --repo "$REPO" \
  --title "CI/CD: GitHub Actions → GHCR → openova-private SHA deploy" \
  --label "area/ci-cd" \
  --body "## Goal
Automated build and deploy pipeline for the Go backend.

## Tasks
- [ ] backend.yml workflow: triggers on backend/** changes
- [ ] build-api job: go build, docker build, push to GHCR
- [ ] build-ingestion job: same for ingestion image
- [ ] deploy job: checkout openova-private, sed SHA into api.yaml, commit + push
- [ ] OPENOVA_PRIVATE_PAT secret configured in repo settings
- [ ] Verify Flux reconciles new image within ~1 min

## Acceptance Criteria
- Push to backend/ triggers full pipeline
- New SHA appears in clusters/contabo-mkt/apps/cinova/services/api.yaml
- Pod rolls out successfully"

gh issue create \
  --repo "$REPO" \
  --title "Traefik ingress: api.cinova.openova.io with TLS" \
  --label "area/infra" \
  --body "## Goal
Expose the Cinova API publicly at api.cinova.openova.io with Let's Encrypt TLS.

## Tasks
- [ ] Ingress manifest using Traefik IngressRoute or standard Ingress with Traefik class
- [ ] TLS via cert-manager ClusterIssuer (letsencrypt-prod)
- [ ] DNS record: api.cinova.openova.io → VPS IP (via Dynadot API)
- [ ] Route: / → cinova-api service port 8080
- [ ] Verify TLS cert issued and valid
- [ ] Test GET https://api.cinova.openova.io/healthz returns 200

## Acceptance Criteria
- https://api.cinova.openova.io/healthz returns HTTP 200
- TLS cert valid (not self-signed)
- HTTP redirects to HTTPS"

# ─── PHASE 2 — Graph & AI Enrichment ─────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "Neo4j graph schema: Movie, Person, Genre, Theme, Mood nodes" \
  --label "area/backend" \
  --body "## Goal
Define the Neo4j graph schema used for recommendations and AI-enriched metadata.

## Node Types
- \`Movie\` — tmdb_id, title, year, cinova_score
- \`TVShow\` — tmdb_id, name, year, cinova_score
- \`Person\` — tmdb_id, name, role (director/actor/writer)
- \`Genre\` — name
- \`Theme\` — name (extracted by Claude)
- \`Mood\` — name (extracted by Claude)
- \`WatchProvider\` — provider_id, name

## Relationship Types
\`DIRECTED\`, \`ACTED_IN\`, \`WROTE\`, \`HAS_GENRE\`, \`HAS_THEME\`, \`HAS_MOOD\`, \`INFLUENCED_BY\`, \`AVAILABLE_ON\`

## Tasks
- [ ] Cypher schema constraints and indexes (CREATE CONSTRAINT, CREATE INDEX)
- [ ] Go functions to create/upsert each node type
- [ ] Go functions to create/upsert each relationship type
- [ ] Schema migration runner (apply idempotent Cypher on startup)

## Acceptance Criteria
- Schema applied cleanly on empty and populated DB
- All node/relationship CRUD functions tested"

gh issue create \
  --repo "$REPO" \
  --title "Wikidata SPARQL ingestion: director influence graph" \
  --label "area/data" \
  --body "## Goal
Enrich the Neo4j graph with director influence relationships sourced from Wikidata.

## Tasks
- [ ] SPARQL query to get film directors and their influences from Wikidata
- [ ] Match Wikidata entries to TMDB Person nodes by name/ID
- [ ] Create \`INFLUENCED_BY\` relationships in Neo4j
- [ ] Schedule as a weekly CronJob (low frequency, data changes rarely)
- [ ] Handle Wikidata rate limits and timeouts gracefully

## Acceptance Criteria
- At least 1000 INFLUENCED_BY relationships ingested for major directors
- Job completes without error
- Graph traversal query returns influence chain for a known director"

gh issue create \
  --repo "$REPO" \
  --title "Claude theme/mood extraction batch job via Axon" \
  --label "area/backend" \
  --body "## Goal
Use Claude (via the Axon agent gateway) to extract themes and moods for movies/shows lacking this data.

## Tasks
- [ ] Identify movies with no Theme/Mood nodes in Neo4j
- [ ] Batch prompt construction: title + overview + genre → themes/moods JSON
- [ ] Call Axon API at api.openova.io/axon with prompt
- [ ] Parse response and create Theme/Mood nodes + relationships in Neo4j
- [ ] Process in batches of 50 with rate limiting
- [ ] Kubernetes Job manifest for on-demand runs; CronJob for nightly delta

## Acceptance Criteria
- 95%+ of top 10K movies have at least 3 themes and 2 moods
- Extraction results are consistent (idempotent re-runs)"

gh issue create \
  --repo "$REPO" \
  --title "CinovaScore algorithm: Bayesian + graph prestige" \
  --label "area/backend" \
  --body "## Goal
Compute the CinovaScore — Cinova's quality signal combining Bayesian rating and graph prestige.

## Algorithm
\`\`\`
CinovaScore = 0.6 * BayesianRating + 0.4 * GraphPrestige
BayesianRating = (v/(v+m)) * R + (m/(v+m)) * C
  v = vote count, m = minimum votes threshold (500)
  R = movie average rating, C = global mean rating
GraphPrestige = PageRank-derived score from director influence graph
\`\`\`

## Tasks
- [ ] Implement Bayesian rating calculation
- [ ] Run Neo4j PageRank on director influence graph
- [ ] Combine into CinovaScore and store on Movie/TVShow node
- [ ] Batch update job for all existing content
- [ ] Incremental update: recalculate on new vote data from TMDB sync
- [ ] Expose score in API responses

## Acceptance Criteria
- All movies have a CinovaScore between 0–100
- Score correlates reasonably with known high-quality films"

gh issue create \
  --repo "$REPO" \
  --title "Natural language search: NL → Cypher via Claude" \
  --label "area/backend" \
  --body "## Goal
Enable users to search with natural language queries (e.g. 'mind-bending sci-fi from the 90s') via Claude-generated Cypher.

## Tasks
- [ ] POST /api/v1/search endpoint accepting \`q\` string parameter
- [ ] Prompt engineering: system prompt with Neo4j schema context
- [ ] Call Claude via Axon with user query → receive Cypher query
- [ ] Validate and sanitize generated Cypher (read-only, no WRITE clauses)
- [ ] Execute Cypher on Neo4j, map results to Movie/TVShow structs
- [ ] Cache results in Redis (key: hash of query, TTL: 1h)
- [ ] Fallback: if Cypher invalid, run full-text search on PostgreSQL

## Acceptance Criteria
- 'mind-bending sci-fi from the 90s' returns relevant results
- Malicious Cypher injection attempts are blocked
- Response time < 3s (cached < 100ms)"

gh issue create \
  --repo "$REPO" \
  --title "Graph recommendation engine: taste profile traversal" \
  --label "area/backend" \
  --body "## Goal
Generate personalized movie/TV recommendations by traversing the Neo4j graph from a user's taste profile.

## Algorithm
1. Build user taste vector from ratings, saves, dismisses
2. Find seed nodes (high-rated movies/directors/themes)
3. Traverse: DIRECTED_BY → INFLUENCED_BY → other movies
4. Score candidates by graph distance + CinovaScore + recency
5. Filter already-seen and dismissed content

## Tasks
- [ ] Aggregate user interaction history into taste profile
- [ ] Cypher query for graph traversal (depth 2–3 hops)
- [ ] Scoring and ranking logic in Go
- [ ] Cache per-user recommendations (TTL: 15 min)
- [ ] Anonymous session recommendations (based on session interactions only)
- [ ] GET /api/v1/recommend endpoint

## Acceptance Criteria
- Recommendations improve after 5+ interactions
- Response time < 500ms (graph query + scoring)
- No dismissed content appears in results"

gh issue create \
  --repo "$REPO" \
  --title "Redis cache: search results (1h TTL), trending (15m TTL)" \
  --label "area/backend" \
  --body "## Goal
Add Redis caching to reduce Neo4j load and improve response latency.

## Tasks
- [ ] Deploy Redis to cinova namespace (K8s Deployment + Service)
- [ ] Go Redis client (go-redis/v9)
- [ ] Cache middleware for GET endpoints
- [ ] Cache key strategy: path + query params hash
- [ ] TTLs: search=1h, trending=15m, recommendations=15m, movie detail=6h
- [ ] Cache invalidation on ingestion sync completion
- [ ] Redis health check in /readyz

## Acceptance Criteria
- Repeated identical search returns in < 50ms
- Cache hit rate > 80% for trending endpoint under normal load"

# ─── PHASE 3 — Backend API Endpoints ─────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "GET /api/v1/trending: trending movies by country" \
  --label "area/backend" \
  --body "## Goal
Return trending movies and TV shows filtered by the user's country.

## Spec
\`\`\`
GET /api/v1/trending?type=movie|tv|all&country=US
Authorization: Bearer <jwt>

Response: { items: MovieCard[], updatedAt: string }
\`\`\`

## Tasks
- [ ] Query PostgreSQL for today's trending (from TMDB daily sync)
- [ ] Filter watch providers by country
- [ ] Include CinovaScore, top streaming provider, poster path
- [ ] Cache: 15 min TTL per country+type combination
- [ ] Default country from JWT claims if not provided

## Acceptance Criteria
- Returns 20 results with streaming availability for user's country
- Response time < 200ms (cached)"

gh issue create \
  --repo "$REPO" \
  --title "GET /api/v1/recommend: personalized recommendations" \
  --label "area/backend" \
  --body "## Goal
Return personalized movie/TV recommendations based on the user's taste profile.

## Spec
\`\`\`
GET /api/v1/recommend?type=movie|tv|all&limit=20&offset=0
Authorization: Bearer <jwt>

Response: { items: MovieCard[], source: 'personalized'|'popular' }
\`\`\`

## Tasks
- [ ] Use graph recommendation engine (see separate issue)
- [ ] Fall back to trending/popular for users with < 3 interactions
- [ ] Pagination support (limit/offset)
- [ ] Include reason snippet ('Because you liked Blade Runner')
- [ ] Filter out content already in watchlist

## Acceptance Criteria
- New users get popular content fallback
- Users with history get personalized results
- Reason field populated for personalized results"

gh issue create \
  --repo "$REPO" \
  --title "GET /api/v1/movie/{id}: full movie detail with streaming" \
  --label "area/backend" \
  --body "## Goal
Full movie detail endpoint with all metadata, cast, and streaming availability.

## Spec
\`\`\`
GET /api/v1/movie/{id}?country=US
Authorization: Bearer <jwt>

Response: MovieDetail (full)
\`\`\`

## Fields
title, overview, release_date, runtime, genres, themes, moods, cast (top 10),
director, CinovaScore, TMDB rating, streaming providers (by country),
trailer_key (YouTube), similar movies (10), user's rating/save status

## Tasks
- [ ] Join PostgreSQL movie + Neo4j themes/moods + Redis cache
- [ ] User interaction state (has user rated/saved this?)
- [ ] Cache: 6h TTL (detail rarely changes)
- [ ] 404 for unknown IDs

## Acceptance Criteria
- Returns all fields for a known movie ID
- Streaming providers accurate for specified country
- User's rating/save state reflected correctly"

gh issue create \
  --repo "$REPO" \
  --title "GET /api/v1/tv/{id}: full TV show detail" \
  --label "area/backend" \
  --body "## Goal
Full TV show detail endpoint mirroring the movie detail endpoint.

## Spec
\`\`\`
GET /api/v1/tv/{id}?country=US
Authorization: Bearer <jwt>
\`\`\`

## Fields
name, overview, first_air_date, number_of_seasons, number_of_episodes, status,
genres, themes, moods, creators, cast (top 10), CinovaScore, TMDB rating,
streaming providers (by country), trailer_key, similar shows (10)

## Tasks
- [ ] Implement analogous to movie detail
- [ ] Season/episode count from TMDB sync
- [ ] Cache: 6h TTL

## Acceptance Criteria
- All fields populated for a known TV show ID
- Streaming providers accurate"

gh issue create \
  --repo "$REPO" \
  --title "POST /api/v1/me/rate, save, dismiss: interaction endpoints" \
  --label "area/backend" \
  --body "## Goal
Record user interactions (rating, save to watchlist, dismiss) used to power recommendations.

## Spec
\`\`\`
POST /api/v1/me/rate    { content_id, content_type, rating: 1-10 }
POST /api/v1/me/save    { content_id, content_type }
POST /api/v1/me/dismiss { content_id, content_type }
Authorization: Bearer <jwt>
\`\`\`

## Tasks
- [ ] PostgreSQL interactions table: user_id/session_id, content_id, content_type, action, value, created_at
- [ ] Upsert on re-rate (one rating per user per content)
- [ ] Invalidate recommendation cache on new interaction
- [ ] Works for both anonymous and authenticated users
- [ ] Rate limit: 100 interactions/min per user

## Acceptance Criteria
- Interaction recorded in DB
- Re-rating updates existing record
- Recommendation cache invalidated after interaction"

gh issue create \
  --repo "$REPO" \
  --title "GET /api/v1/me/watchlist: user/anonymous watchlist" \
  --label "area/backend" \
  --body "## Goal
Return the user's saved watchlist with pagination and filtering.

## Spec
\`\`\`
GET /api/v1/me/watchlist?type=movie|tv|all&status=saved|watched&limit=20&offset=0
Authorization: Bearer <jwt>
\`\`\`

## Tasks
- [ ] Query interactions table for saves
- [ ] Join with movie/TV metadata for card display
- [ ] Filter by type and watch status
- [ ] Pagination (limit/offset)
- [ ] Works for anonymous sessions (persisted in JWT session)

## Acceptance Criteria
- Returns saved items with full card data
- Pagination works correctly
- Anonymous and authenticated users both get correct results"

gh issue create \
  --repo "$REPO" \
  --title "Streaming provider deep links and JustWatch passthrough" \
  --label "area/backend" \
  --body "## Goal
Generate actionable deep links to streaming apps from watch provider data.

## Tasks
- [ ] Provider deep link map: Netflix, Prime Video, Disney+, Apple TV+, Hulu, Max, etc.
- [ ] Deep link format: \`netflix://title/{id}\` (mobile), \`https://netflix.com/watch/{id}\` (web fallback)
- [ ] JustWatch URL construction: \`https://www.justwatch.com/us/movie/{slug}\`
- [ ] Include rental/purchase links where available (Amazon, Apple, Vudu)
- [ ] Country-specific provider availability

## Acceptance Criteria
- Deep link opens correct content in streaming app on device
- JustWatch fallback always available
- Rental/purchase prices shown where provider supports"

gh issue create \
  --repo "$REPO" \
  --title "Country/region detection and filtering" \
  --label "area/backend" \
  --body "## Goal
Detect and respect user's country for streaming availability and content filtering.

## Tasks
- [ ] GeoIP lookup on first session creation (ip-api.com or MaxMind GeoLite2)
- [ ] Store country in JWT claims (anonymous and authenticated)
- [ ] Allow user override via PATCH /api/v1/me/country
- [ ] All content endpoints filter streaming providers by country
- [ ] Supported countries list from TMDB configuration

## Acceptance Criteria
- Country auto-detected on session creation
- User can override country
- Streaming providers shown are available in user's country only"

# ─── PHASE 4 — Mobile Foundation ─────────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "React Native + Expo project setup with expo-router" \
  --label "area/mobile" \
  --body "## Goal
Bootstrap the Cinova mobile app with Expo, expo-router, and all core dependencies.

## Key Dependencies
- expo ~52 (latest stable)
- expo-router (file-based routing)
- react-native-reanimated (animations)
- react-native-gesture-handler
- expo-image (optimized image loading)
- expo-av (video/audio)
- zustand (state management)
- axios (API client)
- @shopify/flash-list (performant lists)

## Tasks
- [ ] \`npx create-expo-app cinova --template blank-typescript\`
- [ ] Configure expo-router with tab layout
- [ ] Install and configure all dependencies
- [ ] app.json: name, slug, bundle IDs (app.cinova.io)
- [ ] TypeScript strict mode
- [ ] Absolute imports with tsconfig paths

## Acceptance Criteria
- App boots on iOS simulator and Android emulator
- expo-router navigation works between tabs
- No TypeScript errors"

gh issue create \
  --repo "$REPO" \
  --title "Dark theme design system: colors, typography, spacing" \
  --label "area/mobile" \
  --body "## Goal
Establish the Cinova design system — dark, cinematic, premium feel.

## Color Palette
\`\`\`
Background:  #0A0A0F (near-black)
Surface:     #12121A
Card:        #1A1A26
Border:      #2A2A3A
Primary:     #6C63FF (violet)
Accent:      #FF6B6B (coral)
Gold:        #FFD700 (CinovaScore)
Text:        #F0F0F5
Muted:       #8888AA
\`\`\`

## Tasks
- [ ] Theme constants file (colors, spacing, radii, shadows)
- [ ] Typography scale: display, heading, body, caption (Inter font)
- [ ] Base components: Text, View wrappers with theme support
- [ ] Gradient utility (linear gradient from react-native-linear-gradient)
- [ ] Storybook or visual demo screen for design review

## Acceptance Criteria
- All screens use theme constants (no hardcoded hex values)
- Dark background throughout — no light mode"

gh issue create \
  --repo "$REPO" \
  --title "Anonymous session initialization on first launch" \
  --label "area/mobile" \
  --body "## Goal
On first app launch, create an anonymous session so users can immediately browse without registering.

## Tasks
- [ ] Call POST /api/v1/auth/session on first launch
- [ ] Store access + refresh JWTs in expo-secure-store
- [ ] Handle network failure gracefully (retry with backoff, offline mode)
- [ ] Session persists across app restarts
- [ ] Generate consistent device fingerprint (expo-device + random UUID)

## Acceptance Criteria
- First launch establishes session within 2s
- App works offline after initial session creation (cached data)
- Session survives app restart"

gh issue create \
  --repo "$REPO" \
  --title "Zustand store: user, session, country, watchlist" \
  --label "area/mobile" \
  --body "## Goal
Global state management for auth, user preferences, and watchlist.

## Slices
- \`sessionSlice\`: JWT tokens, session ID, isAnonymous flag
- \`userSlice\`: user profile, country, preferences
- \`watchlistSlice\`: saved items (optimistic updates)
- \`uiSlice\`: loading states, error messages

## Tasks
- [ ] Zustand store with persist middleware (AsyncStorage)
- [ ] Auth slice with token refresh logic
- [ ] Watchlist with optimistic add/remove
- [ ] Country selector with stored override
- [ ] Hydration on app start from secure storage

## Acceptance Criteria
- State persists across app restarts
- Optimistic watchlist updates feel instant
- Token refresh is transparent to the user"

gh issue create \
  --repo "$REPO" \
  --title "API client with JWT auth interceptor and refresh logic" \
  --label "area/mobile" \
  --body "## Goal
Axios-based API client that automatically attaches JWTs and handles token refresh.

## Tasks
- [ ] Axios instance configured for api.cinova.openova.io
- [ ] Request interceptor: attach Authorization header
- [ ] Response interceptor: on 401, call refresh endpoint, retry original request
- [ ] Refresh token queue: concurrent requests wait for single refresh
- [ ] On refresh failure: clear tokens, redirect to session init
- [ ] TypeScript types for all API responses
- [ ] Environment config: DEV points to localhost:8080, PROD to api.cinova.openova.io

## Acceptance Criteria
- Expired access token refreshed transparently
- Concurrent requests don't trigger multiple refresh calls
- Network errors surfaced as typed errors"

gh issue create \
  --repo "$REPO" \
  --title "Bottom tab navigation: Home, Discover, Search, Watchlist, Profile" \
  --label "area/mobile" \
  --body "## Goal
Main tab navigation structure with cinematic dark styling.

## Tabs
1. Home — trending + recommendations carousels
2. Discover — full-screen vertical Reels-style browse
3. Search — NL AI search
4. Watchlist — saved content grid
5. Profile — account + settings

## Tasks
- [ ] expo-router tabs layout with custom tab bar
- [ ] Custom tab bar component: icon + label, violet active indicator
- [ ] Haptic feedback on tab press (expo-haptics)
- [ ] Tab icons from @expo/vector-icons (Feather or custom SVG)
- [ ] Deep link support: cinova://movie/{id}, cinova://tv/{id}

## Acceptance Criteria
- All 5 tabs navigate correctly
- Active tab highlighted with violet indicator
- Deep links open correct screen"

# ─── PHASE 5 — Mobile Core Screens ───────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "Home screen: hero carousel + horizontal content carousels" \
  --label "area/mobile" \
  --body "## Goal
The main discovery screen with a cinematic hero section and horizontal content rows.

## Layout
1. HeroCarousel (full-width, auto-scrolling featured content)
2. 'Trending in [Country]' — horizontal MovieCard scroll
3. 'Recommended for You' — horizontal MovieCard scroll
4. 'Top Rated on Netflix' — provider-filtered row
5. 'Hidden Gems' (CinovaScore > 85, vote count < 10K)
6. Genre rows (Action, Drama, Comedy...)

## Tasks
- [ ] FlatList-based horizontal carousels with FlashList
- [ ] Pull-to-refresh
- [ ] Skeleton loading state
- [ ] Infinite scroll for recommendation rows
- [ ] Section headers with 'See All' navigation

## Acceptance Criteria
- Home loads in < 1.5s on 4G
- Smooth 60fps scroll on both platforms
- Skeleton shown while data loads"

gh issue create \
  --repo "$REPO" \
  --title "Discover screen: full-screen vertical Reels/TikTok style" \
  --label "area/mobile" \
  --body "## Goal
Immersive full-screen vertical scroll through movies/shows — swipe up to next, down to previous.

## UX
- Full-screen movie poster/backdrop with gradient overlay
- Bottom: title, year, genres, CinovaScore
- Right action column: Rate, Save, More Info, Share
- Auto-advance trailer preview on 2s hover (muted)
- Swipe up: next movie | Swipe down: previous | Tap: full detail

## Tasks
- [ ] PagerView or FlatList pagingEnabled vertical scroll
- [ ] react-native-reanimated for smooth transitions
- [ ] Preload next 3 items for instant swipe
- [ ] Action column with haptic feedback on tap
- [ ] Muted video preview with expo-av
- [ ] Dismiss gesture (horizontal swipe left)

## Acceptance Criteria
- 60fps swipe between full-screen cards
- Video preview plays within 2s of settling on card
- Actions (rate, save) respond instantly with haptic"

gh issue create \
  --repo "$REPO" \
  --title "Search screen: NL AI search with debounce and results" \
  --label "area/mobile" \
  --body "## Goal
Natural language search powered by Claude via the backend NL→Cypher endpoint.

## UX
- Search bar with AI sparkle icon
- Placeholder: 'Try: psychological thriller set in Tokyo'
- Debounced input (500ms) triggers API call
- Results grid (2-column) with MovieCard components
- Suggested queries below empty search bar
- Search history (local, last 10)

## Tasks
- [ ] Debounced search input with loading indicator
- [ ] POST /api/v1/search integration
- [ ] 2-column result grid with FlashList
- [ ] Suggested query chips (pulled from backend or hardcoded MVP)
- [ ] Local search history with expo-async-storage
- [ ] Empty state: 'No results — try different words'
- [ ] Error state: 'Search unavailable — try again'

## Acceptance Criteria
- Results appear within 3s of typing pause
- Cached queries return in < 100ms
- Suggested queries are tappable and populate the search bar"

gh issue create \
  --repo "$REPO" \
  --title "Movie detail screen: cinematic hero + full info + cast + similar" \
  --label "area/mobile" \
  --body "## Goal
Full movie detail page with cinematic backdrop hero and complete metadata.

## Layout
1. Full-width backdrop with gradient to black
2. Floating: title, year, runtime, CinovaScore ring
3. Streaming providers row (logos, tappable deep links)
4. Play Trailer button
5. Themes + Moods pills
6. Overview (expandable)
7. Cast horizontal scroll (headshots + names)
8. Rate / Save / Dismiss action bar
9. Similar Movies horizontal row

## Tasks
- [ ] Animated scroll header (backdrop fades as you scroll down)
- [ ] expo-av trailer playback (inline, tap for fullscreen)
- [ ] StreamingBadge components with deep links
- [ ] Cast photos with person detail navigation
- [ ] Share sheet via expo-sharing
- [ ] Swipe-back gesture (iOS native feel)

## Acceptance Criteria
- Backdrop image loads < 1s (CDN + expo-image cache)
- Trailer plays within 3s of tap
- All streaming links open correct app/URL"

gh issue create \
  --repo "$REPO" \
  --title "Watchlist screen: grid layout with filter tabs" \
  --label "area/mobile" \
  --body "## Goal
User's saved content in a browsable grid with filter tabs.

## Layout
- Filter tabs: All | Movies | TV Shows | Watched
- 3-column poster grid (FlashList)
- Long press → quick action menu (Mark Watched, Remove)
- Pull to refresh
- Empty state with 'Start saving movies' CTA

## Tasks
- [ ] FlashList grid (numColumns: 3)
- [ ] Filter tab component (animated underline indicator)
- [ ] Long press context menu with Reanimated
- [ ] Optimistic remove (instant UI update, API call in background)
- [ ] Empty state illustration

## Acceptance Criteria
- Grid renders 100+ items at 60fps
- Filter tabs switch instantly
- Long press menu appears within 200ms"

gh issue create \
  --repo "$REPO" \
  --title "Profile screen: anonymous CTA vs logged-in stats" \
  --label "area/mobile" \
  --body "## Goal
Profile screen adapts based on auth state — prompts anonymous users to register, shows stats for logged-in users.

## Anonymous State
- 'Create account to sync your watchlist'
- Sign Up / Sign In CTAs
- Ratings count teaser

## Logged-In State
- Avatar (initials fallback), username, email
- Stats: Movies rated, TV rated, Watchlist count
- Country selector
- Sign Out

## Settings Section (both states)
- Notifications toggle
- Remove ads (RevenueCat)
- App version
- Privacy Policy / Terms links

## Tasks
- [ ] Conditional rendering based on isAnonymous from store
- [ ] Stats fetched from GET /api/v1/me/stats
- [ ] Country picker modal
- [ ] Sign out clears store + secure storage + starts new anon session

## Acceptance Criteria
- Anonymous and logged-in states render correctly
- Country change takes effect immediately on other screens
- Sign out is clean (no residual state)"

gh issue create \
  --repo "$REPO" \
  --title "Person detail screen: filmography + recommendations" \
  --label "area/mobile" \
  --body "## Goal
Director/actor detail page showing filmography and related recommendations.

## Layout
- Profile photo + name + known for (Director / Actor / Writer)
- Biography (expandable)
- Filmography grid (sorted by CinovaScore desc)
- 'Others influenced by [Name]' row (from Neo4j graph)

## Tasks
- [ ] GET /api/v1/person/{id} endpoint (if not yet built, add to backend backlog)
- [ ] Filmography FlashList grid (2-column)
- [ ] Graph-powered 'influenced by' section
- [ ] Navigate to movie detail on tap

## Acceptance Criteria
- Person page loads for directors and actors
- Filmography sorted by CinovaScore descending
- Influence graph section populated for major directors"

# ─── PHASE 6 — Mobile Components ─────────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "MovieCard component: sm/md/lg sizes, CinovaScore badge, streaming logo" \
  --label "area/mobile" \
  --body "## Goal
Reusable MovieCard component used across all screens.

## Sizes
- \`sm\` (80×120): poster only, used in compact rows
- \`md\` (120×180): poster + title + CinovaScore badge
- \`lg\` (160×240): poster + title + year + top streaming provider logo

## Props
\`\`\`typescript
type MovieCardProps = {
  item: MovieCard
  size: 'sm' | 'md' | 'lg'
  onPress: () => void
  onLongPress?: () => void
}
\`\`\`

## Tasks
- [ ] Three size variants with consistent border radius and shadow
- [ ] CinovaScore badge (gold, circular, top-right corner)
- [ ] Streaming logo (provider logo bottom-left)
- [ ] Skeleton loading variant
- [ ] Accessible: role='button', accessible label with title + score

## Acceptance Criteria
- All three sizes render correctly on iOS and Android
- Skeleton transitions smoothly to loaded state
- Tapping navigates to correct detail screen"

gh issue create \
  --repo "$REPO" \
  --title "HeroCarousel component: auto-scroll, gradient overlay, CTAs" \
  --label "area/mobile" \
  --body "## Goal
Full-width featured content carousel for the top of the Home screen.

## Design
- Full-width, height = 55% of screen
- Background: movie backdrop image
- Gradient: transparent → black (bottom 40%)
- Overlay: title, genres, CinovaScore, 'Watch Now' + 'Save' CTAs
- Pagination dots at bottom
- Auto-scrolls every 5s, pauses on user interaction

## Tasks
- [ ] Reanimated-powered horizontal pager
- [ ] expo-image for backdrop with fade-in
- [ ] Auto-scroll timer (pause on press-in, resume on press-out)
- [ ] Pagination dots with animated active indicator
- [ ] CTA buttons: Watch Now (→ deep link), Save (→ watchlist)

## Acceptance Criteria
- Auto-scrolls smoothly every 5s
- Manual swipe overrides auto-scroll
- Backdrop loads from CDN < 1s"

gh issue create \
  --repo "$REPO" \
  --title "CinovaScore component: circular progress ring, color-coded" \
  --label "area/mobile" \
  --body "## Goal
Reusable CinovaScore display with circular SVG ring.

## Design
- Circular SVG ring (react-native-svg)
- Color: green (80+), yellow (60-79), red (<60)
- Center: numeric score
- Sizes: xs (24px), sm (40px), md (64px), lg (96px)

## Tasks
- [ ] SVG ring with animated fill on mount (Reanimated)
- [ ] Color mapping from score value
- [ ] Four size variants
- [ ] Label below (optional): 'CinovaScore'
- [ ] Accessibility: accessible label 'CinovaScore: 87 out of 100'

## Acceptance Criteria
- Ring animates on mount
- Color correct for all score ranges
- Renders correctly on both platforms"

gh issue create \
  --repo "$REPO" \
  --title "StreamingBadge component: provider logo + deep link" \
  --label "area/mobile" \
  --body "## Goal
Tappable streaming provider badge that opens the content in the streaming app.

## Design
- Provider logo (Netflix N, Prime smile, Disney+ D, etc.)
- Sizes: xs (icon only), sm (icon + name)
- Border: subtle 1px border, rounded
- Available / Rent / Buy status indicator

## Tasks
- [ ] Provider logo image map (local assets)
- [ ] Tap handler: Linking.openURL with deep link → web fallback
- [ ] Status label: 'Stream', 'Rent from \$X.XX', 'Buy from \$X.XX'
- [ ] Horizontal scrollable row for multiple providers
- [ ] Loading skeleton

## Acceptance Criteria
- Tap opens correct streaming app on device
- Falls back to web URL if app not installed
- Renders correctly for all major providers"

gh issue create \
  --repo "$REPO" \
  --title "ReelItem component: full-screen reel with action column" \
  --label "area/mobile" \
  --body "## Goal
Full-screen card component for the Discover (Reels) screen.

## Design
- Full screen (width × height)
- Background: backdrop or poster image
- Bottom gradient overlay
- Left: title, year, genres, overview snippet, CinovaScore
- Right action column (vertical): Rate ⭐, Save 🔖, Info ℹ️, Share 📤
- Top-right: streaming provider logo

## Tasks
- [ ] Full-screen layout with Dimensions.get('window')
- [ ] Right action column with icon buttons + count labels
- [ ] expo-av muted video preview (trailer) with autoplay on settle
- [ ] Pause video when not active (Reanimated shared value)
- [ ] Haptic feedback on action button tap

## Acceptance Criteria
- Video pauses when card is not in view
- Action buttons provide immediate haptic response
- Correct content for each card position"

# ─── PHASE 7 — Advanced Features ─────────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "Trailer playback: inline + fullscreen with expo-av" \
  --label "area/mobile" \
  --body "## Goal
Trailer playback experience — inline preview and tappable fullscreen player.

## Tasks
- [ ] expo-av Video component for inline trailer (muted, looping)
- [ ] Tap to expand: fullscreen modal player with sound
- [ ] Fullscreen: orientation lock to landscape
- [ ] Custom controls: play/pause, seek bar, fullscreen toggle, close
- [ ] YouTube key from movie detail → construct embed URL or use expo-web-browser
- [ ] Fallback: if no trailer, show backdrop with 'No trailer available'
- [ ] Cache trailer URI to avoid re-fetch

## Acceptance Criteria
- Trailer plays inline on Discover and Detail screens
- Fullscreen player opens in landscape
- Playback continues seamlessly from inline to fullscreen position"

gh issue create \
  --repo "$REPO" \
  --title "Google Cast (Chromecast) integration for trailer casting" \
  --label "area/mobile" \
  --body "## Goal
Allow users to cast trailers to Chromecast-enabled TVs.

## Tasks
- [ ] react-native-google-cast integration
- [ ] Cast button in trailer player (appears when Chromecast detected)
- [ ] Session management: connect, cast, disconnect
- [ ] Mini controller bar when casting is active
- [ ] Graceful degradation: cast button hidden if no Chromecast available

## Acceptance Criteria
- Cast button appears when Chromecast is on local network
- Trailer casts and plays on TV
- Mini controller shows current cast status"

gh issue create \
  --repo "$REPO" \
  --title "Streaming deep links: JustWatch passthrough per provider" \
  --label "area/mobile" \
  --body "## Goal
Open content directly in the streaming app when tapping a provider badge.

## Deep Link Map (examples)
\`\`\`
Netflix:   netflix://www.netflix.com/watch/{netflix_id}
Prime:     aiv://aiv/resume?_encoding=UTF8&asin={asin}
Disney+:   disneyplus://
Apple TV+: https://tv.apple.com (web only)
JustWatch: https://www.justwatch.com/{country}/movie/{slug}
\`\`\`

## Tasks
- [ ] Provider → deep link scheme map
- [ ] Linking.canOpenURL check before attempting native deep link
- [ ] Fallback chain: native deep link → web URL → JustWatch
- [ ] Track which provider was tapped (for analytics)
- [ ] Test on both iOS and Android for each major provider

## Acceptance Criteria
- Native deep link opens correct content in app (not just home screen)
- JustWatch fallback always works
- No crash if provider app not installed"

gh issue create \
  --repo "$REPO" \
  --title "Region/country picker with flag display" \
  --label "area/mobile" \
  --body "## Goal
Allow users to manually select their country for streaming availability.

## UI
- Bottom sheet modal (react-native-bottom-sheet)
- Search bar to filter countries
- Flag emoji + country name list (FlatList)
- Current selection highlighted
- 'Auto-detect' option at top

## Tasks
- [ ] Country list from TMDB supported countries (fetched + cached)
- [ ] Flag rendering (emoji flags from ISO code)
- [ ] Search filter (debounced)
- [ ] Store selection in user store + sync to backend via PATCH /api/v1/me/country
- [ ] Trigger available immediately: re-fetch streaming data for current screen

## Acceptance Criteria
- Country picker opens as bottom sheet
- Selection updates streaming availability on current screen within 1s
- Selection persists across app restarts"

gh issue create \
  --repo "$REPO" \
  --title "Sign in / Sign up flow with session migration" \
  --label "area/mobile" \
  --body "## Goal
Auth screens for registering and signing in, with seamless migration of anonymous session data.

## Screens
1. AuthGate — shown when user taps 'Create Account' from Profile
2. SignUp — email + password + confirm password
3. SignIn — email + password + 'Forgot password' (MVP: omit reset)
4. Success — 'Welcome to Cinova, {name}!' with confetti (expo-confetti)

## Tasks
- [ ] Form validation (email format, password length ≥ 8)
- [ ] Call POST /api/v1/auth/signup with anon session JWT header
- [ ] Backend migrates anon session data to new user
- [ ] Store new authenticated JWTs in secure storage
- [ ] Update user store to isAnonymous: false
- [ ] Dismiss modal and refresh screens with user data

## Acceptance Criteria
- Watchlist and ratings carry over after signup
- Form shows inline validation errors
- Success screen displayed before returning to app"

gh issue create \
  --repo "$REPO" \
  --title "Push notifications: new on streaming, leaving soon alerts" \
  --label "area/mobile" \
  --body "## Goal
Push notifications for streaming availability changes on watchlisted content.

## Notification Types
1. 'Now on Netflix: {movie} is now streaming in your country'
2. 'Leaving soon: {movie} leaves Netflix in 7 days'
3. 'New recommendation: Based on your taste, try {movie}'

## Tasks
- [ ] expo-notifications setup with permission request flow
- [ ] Push token registration stored in backend (POST /api/v1/me/push-token)
- [ ] Backend: scheduled job checks watchlist vs provider changes
- [ ] Send via Expo Push Notification Service (EPNS)
- [ ] Notification deep links: tap → open movie detail
- [ ] Notification preferences in Profile settings

## Acceptance Criteria
- Permission prompt shown on appropriate trigger (not on first launch)
- Notification taps navigate to correct movie detail
- Users can disable notifications per type"

# ─── PHASE 8 — Monetization ───────────────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "Google AdMob integration: banner + interstitial ads" \
  --label "area/mobile" \
  --body "## Goal
Ad integration for the free tier using Google AdMob.

## Ad Placements
- Banner: bottom of Search and Watchlist screens
- Interstitial: shown every 5th movie detail view
- No ads on Home or Discover (UX priority screens)

## Tasks
- [ ] react-native-google-mobile-ads integration
- [ ] AdMob app ID configured in app.json (via expo-build-properties)
- [ ] BannerAd component (test ID in dev, real ID in prod)
- [ ] InterstitialAd with frequency capping (every 5 views)
- [ ] Ad-free check: if RevenueCat subscriber, skip all ads
- [ ] GDPR consent form via UMP SDK (required for EU users)
- [ ] Test with AdMob test device IDs

## Acceptance Criteria
- Banner ads render without layout shift
- Interstitial shows at correct frequency
- Subscribers see zero ads"

gh issue create \
  --repo "$REPO" \
  --title "Rewarded video ads: watch ad for extra AI searches" \
  --label "area/mobile" \
  --body "## Goal
Rewarded video ads gate additional AI search quota for free users.

## UX Flow
1. Free user has 10 AI searches/day
2. On limit: 'Watch a short ad for 5 more searches'
3. User watches 30s ad → +5 search credits granted
4. Credits reset at midnight UTC

## Tasks
- [ ] Search credit counter in user store (persisted)
- [ ] RewardedAd from react-native-google-mobile-ads
- [ ] Credit grant on ad completion (backend validation optional for MVP)
- [ ] 'Watch Ad' modal with countdown and skip option (post-5s)
- [ ] Subscribers bypass quota entirely

## Acceptance Criteria
- Free user sees limit message at 10 searches
- Completing rewarded ad grants exactly 5 credits
- Credits display updates immediately"

gh issue create \
  --repo "$REPO" \
  --title "RevenueCat subscription: remove ads tier (\$1.99/mo)" \
  --label "area/mobile" \
  --body "## Goal
Subscription purchase flow for the 'Cinova Plus' ad-free tier.

## Product
- Monthly: \$1.99/month
- Annual: \$14.99/year (~38% discount)
- Benefits: Remove all ads, unlimited AI search

## Tasks
- [ ] RevenueCat SDK (react-native-purchases)
- [ ] Paywall screen: benefits list + pricing + CTA
- [ ] Purchase flow with native StoreKit/Play Billing
- [ ] Entitlement check: 'cinova_plus' → hide all ads
- [ ] Restore purchases button
- [ ] Subscription status in Profile screen
- [ ] Webhook: RevenueCat → backend to track subscriber status

## Acceptance Criteria
- Subscription purchase completes end-to-end in sandbox
- Ads disappear immediately after purchase
- Restore purchases works"

gh issue create \
  --repo "$REPO" \
  --title "Affiliate deep links: streaming rental/purchase with tracking" \
  --label "area/mobile" \
  --body "## Goal
Generate tracked affiliate links for streaming rentals and purchases (Amazon, Apple, Vudu).

## Programs
- Amazon Associates: \`amazon.com/dp/{asin}?tag={affiliate-id}\`
- Apple Affiliate: \`https://geo.itunes.apple.com/...\`
- Vudu: partner program application required

## Tasks
- [ ] Affiliate ID configuration per program
- [ ] Link construction for each provider with affiliate parameter
- [ ] Click tracking: log to analytics (Umami event)
- [ ] A/B test: affiliate link vs JustWatch passthrough
- [ ] Revenue reporting dashboard (Umami custom events)

## Acceptance Criteria
- Affiliate links contain correct tracking parameters
- Click events logged in Umami
- Links open correct purchase page"

# ─── PHASE 9 — App Store ─────────────────────────────────────────────────────

gh issue create \
  --repo "$REPO" \
  --title "iOS App Store: metadata, screenshots, privacy policy" \
  --label "area/mobile" \
  --body "## Goal
Prepare all App Store Connect materials for iOS submission.

## Required Assets
- App name: Cinova
- Subtitle: AI Movie & TV Discovery
- Description (up to 4000 chars)
- Keywords (100 chars): movie, tv, streaming, discover, watchlist, ai, recommendation
- 6.7\" screenshots (iPhone 15 Pro Max) — 6 required
- 12.9\" iPad screenshots (optional)
- App icon: 1024×1024 PNG (no alpha)
- Privacy policy URL: https://cinova.openova.io/privacy

## Tasks
- [ ] Write App Store description
- [ ] Generate screenshots from simulator (key screens)
- [ ] Privacy policy page on website
- [ ] App category: Entertainment
- [ ] Age rating: 4+
- [ ] App Store Connect app record created

## Acceptance Criteria
- All required fields complete in App Store Connect
- Screenshots approved (no device frames, correct dimensions)
- Privacy policy URL live"

gh issue create \
  --repo "$REPO" \
  --title "Google Play Store: metadata, screenshots, content rating" \
  --label "area/mobile" \
  --body "## Goal
Prepare all Google Play Console materials for Android submission.

## Required Assets
- App name: Cinova (max 30 chars)
- Short description (80 chars)
- Full description (4000 chars)
- Feature graphic: 1024×500 PNG
- Phone screenshots: at least 2, up to 8 (16:9 or 9:16)
- Hi-res icon: 512×512 PNG
- Privacy policy URL

## Tasks
- [ ] Write Play Store description (can mirror App Store with tweaks)
- [ ] Generate Android screenshots
- [ ] Complete content rating questionnaire (IARC)
- [ ] Data safety form (what data is collected, shared)
- [ ] Play Console app record created
- [ ] Target API level 34 (Android 14) compliance

## Acceptance Criteria
- All required fields complete in Play Console
- Content rating obtained
- Data safety section complete"

gh issue create \
  --repo "$REPO" \
  --title "Expo EAS build configuration for both stores" \
  --label "area/ci-cd" \
  --body "## Goal
Configure Expo Application Services (EAS) for production builds for iOS and Android.

## Tasks
- [ ] eas.json with production build profiles
- [ ] iOS: provisioning profile + distribution certificate in EAS secrets
- [ ] Android: keystore in EAS secrets
- [ ] Bundle IDs: \`app.cinova.io\` (iOS) / \`app.cinova.io\` (Android)
- [ ] OTA updates enabled (expo-updates) for JS-only fixes
- [ ] EAS Submit configured for both stores
- [ ] GitHub Actions: manual trigger for \`eas build --platform all\`

## Acceptance Criteria
- \`eas build --platform ios --profile production\` succeeds
- \`eas build --platform android --profile production\` succeeds
- IPA and AAB artifacts downloadable from EAS dashboard"

gh issue create \
  --repo "$REPO" \
  --title "App Store Connect + Play Console setup" \
  --label "area/mobile" \
  --body "## Goal
One-time setup of developer accounts and console configurations.

## Tasks
- [ ] Apple Developer Program enrollment (\$99/year) — verify if already active
- [ ] Google Play Developer account (\$25 one-time) — verify if already active
- [ ] App Store Connect: create Cinova app record
- [ ] Google Play Console: create Cinova app record
- [ ] TestFlight internal testing group setup
- [ ] Google Play internal testing track setup
- [ ] Team members added to both consoles

## Acceptance Criteria
- Both app records created and accessible
- Internal testing tracks ready for first build upload
- Team access configured"

echo ""
echo "==> All issues created successfully for $REPO"
echo ""
echo "View issues at: https://github.com/$REPO/issues"
