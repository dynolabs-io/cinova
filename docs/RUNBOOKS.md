# Runbooks

> **WHAT**: Operator how-tos — local setup, CI/CD flow, deploy + rollback, ingestion ops, bootstrap.
> **AUTHORITY**: 📐 Canon. Per-incident playbooks live in [`runbooks/`](runbooks/); generic ops live here.
> **POINTER**: Security policies (secrets, identity) live in [SECURITY.md](SECURITY.md). DoD criteria live in [DOD.md](DOD.md).

## Local development

> Source: previously the "Setup" section of `README.md` (merged here on 2026-05-21).

### Backend

**Prerequisites:** Go 1.23+, a running Neo4j instance, PostgreSQL 16, and Valkey/Redis.

```bash
cd backend

# Copy and configure environment
cp .env.example .env
# Edit .env with your TMDB_API_KEY, DB URLs, AXON_URL, JWT_SECRET, etc.

# Run the API server (default :8080)
go run ./cmd/api

# Run a delta ingestion pass (syncs recent TMDB changes)
go run ./cmd/ingestion --mode=delta --country=US

# Run a full bulk ingestion (first-time setup, takes several hours)
go run ./cmd/ingestion --mode=bulk --country=US
```

Required env vars are documented in `backend/.env.example`. Secrets policy is in [SECURITY.md](SECURITY.md).

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

## CI/CD pipeline

> Source: previously the "CI/CD Pipeline" section of `README.md` (merged here on 2026-05-21).

Three GitHub Actions workflows live in `.github/workflows/`:

| Workflow | Trigger | What it does |
|---|---|---|
| `backend.yml` | push to `main` under `backend/**` | Builds `api` + `ingestion` images, pushes to GHCR, bumps SHA in `openova-private` manifests |
| `mobile.yml` | push to `main` under `mobile/**` | Runs `tsc --noEmit` + Expo export check (static validation, no credentials needed) |
| `eas-build.yml` | push to `main` under `mobile/**`, version tags `v*`, or manual dispatch | EAS production / preview builds for iOS + Android via `expo-github-action` |

### Backend deploy flow

```
Push to backend/**
  → build-api       (go build + docker build/push to GHCR)
  → build-ingestion (same for ingestion image)
  → deploy          (checkout openova-private, sed SHA into:
                       clusters/contabo-mkt/apps/cinova/services/api.yaml
                       clusters/contabo-mkt/apps/cinova/services/ingestion-worker.yaml
                       clusters/contabo-mkt/apps/cinova/cronjobs/ingestion.yaml
                       clusters/contabo-mkt/apps/cinova/cronjobs/ingestion-full.yaml
                       clusters/contabo-mkt/apps/cinova/cronjobs/ingestion-enrich.yaml
                       clusters/contabo-mkt/apps/cinova/cronjobs/vertical-trailers-job.yaml
                     commit + push → Flux reconciles in ~1 min)
```

Images published: `ghcr.io/foundrylab-app/cinova/api:<sha>` and `ghcr.io/foundrylab-app/cinova/ingestion:<sha>`.

Required repo secrets:

- `GITHUB_TOKEN` — auto-provided, used for GHCR push.
- `OPENOVA_PRIVATE_PAT` — PAT with write access to `openova-io/openova-private`.
- `EXPO_TOKEN` — Expo account token for EAS builds.

### Mobile build flow

EAS builds run on:

- Every push to `main` under `mobile/**` → `preview` profile, Android (auto)
- Version tags `v*` → `production` profile, all platforms
- Manual `workflow_dispatch` → operator chooses profile + platform

Profiles + credentials are configured in `mobile/eas.json` and EAS dashboard secrets (Apple Developer / Google Play). App Store / Play Store submissions remain manual.

## Manual rollback (when GitHub Actions can't deploy)

If you must roll back to a specific SHA without going through CI:

```bash
# 1. Find a known-good short SHA from the GHCR registry or commit log
KNOWN_GOOD=abc1234

# 2. Edit the manifest pin in openova-private
cd ~/repos/openova-private
sed -i "s|image: ghcr.io/foundrylab-app/cinova/api:.*|image: ghcr.io/foundrylab-app/cinova/api:${KNOWN_GOOD}|" \
  clusters/contabo-mkt/apps/cinova/services/api.yaml

# 3. Commit + push — Flux reconciles in ~1 min
git add clusters/contabo-mkt/apps/cinova/services/api.yaml
git commit -m "rollback: cinova api to ${KNOWN_GOOD}"
git push
```

Do the equivalent edit on the ingestion CronJob manifests if the bad SHA also affects ingestion.

## Ingestion operations

| Mode | Command | When |
|---|---|---|
| Delta | `go run ./cmd/ingestion --mode=delta --country=<ISO>` | Daily CronJob — syncs recent TMDB changes |
| Bulk | `go run ./cmd/ingestion --mode=bulk --country=<ISO>` | First-time setup; takes hours |
| Enrichment | `--mode=enrich` | Nightly CronJob — themes/moods via Axon |
| Vertical-trailer fetch | `--mode=vertical-trailers` | Periodic — pulls portrait YouTube keys |

The Kubernetes CronJob manifests for each mode live in `openova-private/clusters/contabo-mkt/apps/cinova/cronjobs/`.

## GitHub issue bootstrap

> Source: previously the "GitHub Issues" section of `README.md` (merged here on 2026-05-21).

```bash
chmod +x create-github-issues.sh
./create-github-issues.sh
```

Creates 48 issues across 9 phases with appropriate labels on `foundrylab-app/cinova`. The script is idempotent at the level of "issue with same title already exists" — re-run safely.

Existing label taxonomy:

| Label | Used by |
|---|---|
| `area/backend` | API + ingestion code |
| `area/mobile` | Expo app |
| `area/data` | ingestion / TMDB / Wikidata / YouTube |
| `area/infra` | K8s manifests, registry, secrets |
| `area/ci-cd` | GitHub Actions workflows |
| `enhancement` / `bug` | type |
| `status/in-progress` | active session |
| `status/uat` | code done, walk pending |
| `status/completed` | walk passed, awaiting user verification |
| `status/parked` | blocked or deprioritised |

Per user-global `~/.claude/CLAUDE.md` §6: GitHub does NOT auto-mutex status labels — remove the old one before adding the new one.
