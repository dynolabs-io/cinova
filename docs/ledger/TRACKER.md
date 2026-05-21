# Tracker — open work + DoD progress

> **WHAT**: Live snapshot of open work bucketed by area + status. Cron-refreshable; current snapshot is hand-rolled until that automation lands.
> **AUTHORITY**: 🟢 LIVE STATE. Refresh when issue board state materially changes (≥ 5 issue label changes) OR on any session-ending commit.
> **POINTER**: Per-surface verification state in [TRUST.md](TRUST.md). What's shipped vs design in [../STATUS.md](../STATUS.md).
> **AS-OF**: 2026-05-21 (post-dynolabs-io migration)

## Recent session activity (2026-05-21)

Closed this session — repo move + pipeline retooling:

| # | Title | Outcome |
|---|---|---|
| 98 | docs: bring docs/ tree to canonical §11 shape | ✅ Closed (PR #99) — docs/ scaffold + tree-view README + audit-trailed folds |
| 99 | PR: docs canonical shape | ✅ Merged `02baed7` |
| 100 | infra: transfer cinova to dynolabs-io + adopt vcard iOS TestFlight pipeline | 🟡 In-progress — repo moved + public + pipeline shape live; iterating ios.yml runs to first green TestFlight |
| 101 | PR: retire EAS, adopt vcard macOS-native pipeline | ✅ Merged `e7436bb` |
| 102 | PR: clear strict tsc + eslint errors | ✅ Merged `6bdfca8` |
| 103 | infra: flip dynolabs-io/cinova back to private (2026-07-20) | 🆕 Open — date-anchored reminder |
| 104 | PR: backend.yml GHCR paths foundrylab-app → dynolabs-io | ✅ Merged `ed424be` |

Cross-repo:
- `openova-io/openova-private#163` — Flux manifests cutover GHCR paths ✅ Merged
- `foundrylab-app` org — ✅ DELETED (GHCR migration verified before removal)

iOS pipeline iteration log (live):

| Run | SHA | Result | Diagnosis | Fix shipped |
|---|---|---|---|---|
| 26216262726 | `e7436bb` | cancelled | (manual cancel) | — |
| 26221067473 | `ed424be` | failure | sigh: "Could not find App ID with bundle identifier 'io.dynolabs.cinova'" | Add fastlane produce step |
| 26221253435 | `140ac45` | failure | fastlane produce: "invalid option: --api_key_path" (uses Apple-ID 2FA, not API key) | Replace produce with direct ASC REST API |
| 26221495210 | `8749da6` | failure | POST /v1/apps: HTTP 403 "apps does not allow CREATE" (Apple-side limit) | Drop POST /v1/apps; keep bundleId create; let altool auto-create on first upload |
| 26221703006 | `94aea95` | failure | "Expected .app under Release-iphonesimulator/" — Podfile's DEPLOYMENT_LOCATION=YES sent it to `/tmp/Cinova.dst/` | Find in both locations |
| 26222345040 | `836105e` | failure | fastlane cert: "reached the maximum number of Distribution certificates" — 3 dist certs existed; revoke-oldest only removed 1, leaving 2 | Revoke ALL pre-existing dist certs (CI keychains are ephemeral) |
| 26222547474 | `372d2d4` | failure | install-app step exited 1 with NO diagnostic output — `set -euo pipefail` + `find ... \| grep -v dSYM \| head -1` killed bash before the diagnostic if-block when grep found 0 non-dSYM matches | Drop pipe-to-grep; use `find -not -path "*dSYM*"` and `set -u` (no -e/pipefail) for discovery block |
| 26223284361 | `ecb3383` | failure | install-app: diagnostic confirmed no `Cinova.app` anywhere — SIM build produced only intermediates (`.hmap`, `.dia`, `.pch`, `Cinova.build/`); the Podfile patch's `DEPLOYMENT_LOCATION=YES` redirects BUILT_PRODUCTS_DIR but `xcodebuild build` (no install) doesn't run the deploy phase → app never lands | Override `DEPLOYMENT_LOCATION=NO INSTALL_PATH=` on the SIM-build xcodebuild only; archive still gets YES via Podfile patch. Also strengthened Maestro flow (assertVisible on all 5 tab labels after onboarding) |
| 26224123786 | `0d8f2e4` | failure | `Cinova.app` built at expected location ✓ (DEPLOYMENT_LOCATION fix worked). Install + sim boot ✓. Maestro clearState failed: "Failed to get app binary directory for bundle io.dynolabs.cinova … No such file or directory" — the .app's Info.plist contains `io.openova.cinova`, NOT the bundle ID we expected | **mobile/app.config.ts overrode app.json's bundleIdentifier** (TypeScript config takes precedence in Expo when both present). Update app.config.ts + store-assets/metadata.json from io.openova.cinova → io.dynolabs.cinova |
| 26224865258 | `2038bd3` | failure | App launches ✓, tab bar renders ✓ (5 icons + chat FAB visible in screenshot). Maestro `assertVisible: "Home"` failed after 76s — cinova's tab bar is **icon-only, no text labels**. Backend is scaled-to-0 + Flux suspended (openova-private/faeffea2) so Home content area is blank by design | Relax maestro to: launchApp → 30s wait → takeScreenshot → assertNotVisible error-boundary strings. Don't assert on tab labels (icon-only) or content (backend dead). Native-crash detection in workflow is secondary gate |
| 26225696674 | `5bccd41` | failure | **Maestro PASSED** ✓ (smoke gate works). Archive failed: `Provisioning profile "Dynolabs Cinova App Store" doesn't include the Push Notifications capability` + `... doesn't include the aps-environment entitlement` | Bundle ID was created via `POST /v1/bundleIds` with no capabilities. Add `POST /v1/bundleIdCapabilities` step (capabilityType=PUSH_NOTIFICATIONS) BEFORE fastlane sigh. cinova uses expo-notifications which requires aps-environment |

Cleared in-flight (substrate):
- Apple Developer bundle ID `io.dynolabs.cinova` (T8F2BSD4H7) — created via `POST /v1/bundleIds` in run 26221495210; persistent
- ASC app record for `io.dynolabs.cinova` — **NOT YET CREATED** (Apple API forbids; relying on altool auto-create on first upload)
- "Founders" beta group — skipped because ASC record doesn't exist yet; re-attempted post-upload

## Counts (open issues)

| Bucket | Count |
|---|---|
| Total open | ~43 (#103 added; #98 closed; #100 in-progress) |
| `status/in-progress` | 6 |
| `status/uat` | 18 |
| `status/completed` (awaiting user closure) | 7 |
| Untriaged | ~12 |

## In progress (`status/in-progress`)

| # | Title | Area |
|---|---|---|
| 45 | Google Cast (Chromecast) integration for trailer casting | mobile |
| 54 | iOS App Store: metadata, screenshots, privacy policy | mobile |
| 55 | Google Play Store: metadata, screenshots, content rating | mobile |
| 56 | Expo EAS build configuration for both stores | ci-cd |
| 57 | App Store Connect + Play Console setup | mobile |
| 100 | infra: transfer cinova to dynolabs-io + adopt vcard iOS TestFlight pipeline | infra/ci-cd/mobile |

## UAT (`status/uat`) — code done, walk pending

(unchanged since 2026-05-21 morning — same set as the prior snapshot; #100 hasn't touched any of these)

| # | Title | Area |
|---|---|---|
| 7 | TMDB initial bulk load ingestion job (full mode) | data |
| 46 | Streaming deep links: JustWatch passthrough per provider | mobile |
| 47 | Region/country picker with flag display | mobile |
| 48 | Sign in / Sign up flow with session migration | mobile |
| 49 | Push notifications: new on streaming, leaving soon alerts | mobile |
| 50 | Google AdMob integration: banner + interstitial ads | mobile |
| 51 | Rewarded video ads: watch ad for extra AI searches | mobile |
| 52 | RevenueCat subscription: remove ads tier ($1.99/mo) | mobile |
| 53 | Affiliate deep links: streaming rental/purchase with tracking | mobile |
| 67 | feat: capture all valuable TMDB fields | backend/data |
| 77 | fix(mobile): tab icons show '?' glyphs | mobile |
| 79 | fix(mobile): chat send button icon renders incorrectly | mobile |
| 80 | fix(mobile): TrailerPlayer fullscreen | mobile |
| 82 | feat(mobile): voice input for AI chat — mic button + STT | backend/mobile |
| 83 | feat(mobile): rate button on discover reels — rating dialog | mobile |
| 84 | fix(backend+mobile): AI chat country hardcoded US | backend/mobile |
| 85 | fix(mobile): back navigation in auth screens | mobile |
| 87 | feat(mobile): staggered masonry discover grid | mobile |
| 97 | feat(backend): age bias in discovery — exponential time-decay | backend |

## Awaiting user closure (`status/completed`)

| # | Title |
|---|---|
| 69 | feat: per-user scoring profile presets |
| 71 | feat(mobile): Discover Reels — synopsis + award badge overlay |
| 73 | ci: EAS automated iOS + Android build workflow |
| 74 | infra: K8s CronJob for daily delta ingestion |
| 75 | feat: Wikidata enrichment for TV shows |
| 76 | feat(mobile): make genre/theme/mood chips clickable to filter |
| 78 | fix(mobile): actor + movie detail screens stuck on spinner |

## Untriaged

(unchanged)

| # | Title | Suggested area |
|---|---|---|
| 81 | feat(mobile): show additional quality scores (IMDB, RT) | mobile/data |
| 86 | feat(data+backend): vertical trailer ingestion — portrait YouTube keys | backend/data |
| 88 | feat(mobile): For You feed with AI reasons | backend/mobile |
| 89 | bug(mobile): reels — title masked by bottom tab bar | mobile |
| 90 | bug(mobile): reels — provider icons too small / blank | mobile/data |
| 91 | bug(mobile): reels — Watch on Provider opens App Store | mobile |
| 92 | feat(mobile): reels — tap affordance to detail | mobile |
| 93 | feat(mobile): floating AI assistant button | mobile |
| 94 | feat(mobile): movie detail — inline autoplay trailer | mobile |
| 95 | feat(mobile+backend): clickable awards — drill into winners | backend/mobile |
| 96 | bug(mobile): chips → unfiltered discover | mobile |
| 103 | infra: flip dynolabs-io/cinova back to private (2026-07-20) | infra |

## Refresh protocol

```bash
gh issue list --repo dynolabs-io/cinova --state open --limit 100 \
  --json number,title,labels \
  | jq -r '.[] | "\(.number)\t\([.labels[].name] | join(","))\t\(.title)"' \
  | sort -n
```

Then hand-bucket into the tables above. Refresh on:
- ≥ 5 label changes since AS-OF date
- Any session-ending commit
- Any cross-repo migration affecting this repo's image paths or remote
