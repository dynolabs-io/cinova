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

> Source: previously the "CI/CD Pipeline" section of `README.md` (merged here on 2026-05-21); rewired on 2026-05-21 to retire EAS/Expo-Go and adopt the vcard-sibling macOS-native TestFlight pipeline (issue #100).

Three GitHub Actions workflows live in `.github/workflows/`:

| Workflow | Trigger | What it does |
|---|---|---|
| `backend.yml` | push to `main` under `backend/**` | Builds `api` + `ingestion` images, pushes to GHCR, bumps SHA in `openova-private` manifests |
| `ci.yml` | push to `main` or PR touching `mobile/**` | Runs `tsc --noEmit` + `npm run lint` on ubuntu-latest |
| `ios.yml` | push to `main` touching `mobile/**` or `.maestro/**`, or manual dispatch | macos-latest: `npx expo prebuild` → Podfile patch → CocoaPods → fastlane cert + sigh → sim build → Maestro E2E gate → archive (manual signing) → IPA → altool upload to TestFlight → assign to "Founders" beta group |
| `asc-assign-build.yml` | manual dispatch | Re-assigns a specific CFBundleVersion to "Founders" beta group (rescue path when the in-line assignment was skipped) |

Retired workflows (deleted in PR for #100): `mobile.yml` (replaced by `ci.yml`), `eas-build.yml` (EAS no longer in the build path — `ios.yml` builds natively on macOS runners; EAS retained only as a manual local fallback via `eas build` for developer-machine builds).

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

Images published: `ghcr.io/dynolabs-io/cinova/api:<sha>` and `ghcr.io/dynolabs-io/cinova/ingestion:<sha>`.

Required repo secrets:

| Secret | Used by | Source |
|---|---|---|
| `GITHUB_TOKEN` | `backend.yml` | Auto-provided |
| `OPENOVA_PRIVATE_PAT` | `backend.yml` | PAT with write access to `openova-io/openova-private` |
| `APPLE_TEAM_ID` | `ios.yml` | Dynolabs Apple Developer Team ID (`77GHJHUGD4`) |
| `APP_STORE_CONNECT_ISSUER_ID` | `ios.yml`, `asc-assign-build.yml` | ASC API issuer UUID |
| `APP_STORE_CONNECT_KEY_ID` | `ios.yml`, `asc-assign-build.yml` | ASC API key ID |
| `APP_STORE_CONNECT_PRIVATE_KEY` | `ios.yml`, `asc-assign-build.yml` | base64 of `AuthKey_*.p8` |
| `IOS_DIST_CERT_P12` | (reserved — `ios.yml` regenerates via fastlane cert instead) | base64 of distribution `.p12` |
| `IOS_DIST_P12_PASSWORD` | (paired with `IOS_DIST_CERT_P12`) | Password for the `.p12` |
| `IOS_DIST_PROVISION` | (reserved — `ios.yml` regenerates via fastlane sigh) | base64 of `Cinova_AppStore.mobileprovision` |

Bootstrap on a fresh clone via `./scripts/bootstrap-secrets.sh` — prompts for each value once and pushes via `gh secret set`. The 6 Apple secrets are re-used verbatim from `dynolabs-io/vcard` (`gh secret list -R dynolabs-io/vcard`) because they belong to the org's single Apple Developer Team. The 7th (`IOS_DIST_PROVISION`) is per-app and must be generated fresh for `io.dynolabs.cinova` in the Apple Developer portal.

### iOS TestFlight build flow

Single workflow `ios.yml` triggers on every push to `main` touching `mobile/**` or `.maestro/**`. End-to-end on a `macos-latest` GitHub-hosted runner (~25–45 min wall time):

1. **Prebuild** — `npx expo prebuild --platform ios --no-install --clean` generates the native Xcode project under `mobile/ios/`.
2. **CFBundleVersion bump** — `PlistBuddy` sets `CFBundleVersion=${{ github.run_number }}` so each upload is unique (altool silently dedupes otherwise).
3. **Podfile patch** — disables signing for Pod static-lib targets + forces `SKIP_INSTALL=NO` on the app target (Xcode 26 trap: `SKIP_INSTALL=YES` default leaves `Products/Applications/` empty).
4. **Certificate + provisioning profile** — `fastlane cert` creates a fresh distribution cert; `fastlane sigh` regenerates the App Store profile bound to that cert. ASC API key is used directly; the per-app `IOS_DIST_PROVISION` secret is reserved as a fallback.
5. **Simulator build + Maestro E2E gate** — `xcodebuild` builds for iOS Simulator, app is installed, Maestro flows under `.maestro/` run with crash-log monitoring. If any flow fails, archive + upload are skipped.
6. **Archive** — `xcodebuild archive` with `CODE_SIGN_STYLE=Manual` and the just-fetched profile.
7. **IPA wrap** — `Xcode 26` broke `-exportArchive`; the workflow wraps the archive's `.app` into a `Payload/<App>.app.ipa` ZIP directly.
8. **Upload** — `xcrun altool --upload-app` to App Store Connect.
9. **Beta group assignment** — POST `/v1/betaGroups/{Founders}/relationships/builds` to make the build visible to internal testers (default is `hasAccessToAllBuilds=False`).

Android EAS builds remain manual (`eas build --platform android`); the existing `mobile/eas.json` is retained for that purpose only.

## Manual rollback (when GitHub Actions can't deploy)

If you must roll back to a specific SHA without going through CI:

```bash
# 1. Find a known-good short SHA from the GHCR registry or commit log
KNOWN_GOOD=abc1234

# 2. Edit the manifest pin in openova-private
cd ~/repos/openova-private
sed -i "s|image: ghcr.io/dynolabs-io/cinova/api:.*|image: ghcr.io/dynolabs-io/cinova/api:${KNOWN_GOOD}|" \
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

Creates 48 issues across 9 phases with appropriate labels on `dynolabs-io/cinova`. The script is idempotent at the level of "issue with same title already exists" — re-run safely.

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
