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
| 26226740000 | `869f929` | failure | **Archive PASSED** ✓ + IPA wrapped ✓. altool upload failed: `ERROR: Cannot determine the Apple ID from Bundle ID 'io.dynolabs.cinova' and platform 'IOS' (12)` — no ASC app record exists for the bundle ID, and Apple's ASC REST API forbids `CREATE` on the apps resource (HTTP 403). | **PIPELINE COMPLETE; blocks on Apple-UI click**. Filed as #105 — founder created ASC app "Cinova - Movies & TV" (id 6772122414) at 2026-05-22 ~09:00 UTC. |

## 2026-05-22 — Build 14 reaches TestFlight (post-ASC-create)

After founder created the ASC record per #105:

| Run | SHA | Result | Diagnosis | Fix shipped |
|---|---|---|---|---|
| 26279869138 | `91ef2e6` | partial success | **Build uploaded ✓ (build 13)**, altool returned "UPLOAD SUCCEEDED with no errors". Assign-to-Founders step failed: HTTP 422 "Build is not in an internally testable state" because processingState was still PROCESSING when assign fired immediately after upload | Two-phase wait: poll for build resource appearance (6 min), then poll for processingState=VALID (45 min) before assign |
| 26281419755 | (rerun) | cancelled | asc-assign-build.yml had timeout-minutes=5; in-script polling needs ~45 min | Bump workflow-level timeout to 55 min |
| 26284227101 | manual assign for v14 | partial success | processingState=VALID ✓ but HTTP 422 "not internally testable" still — root cause: Apple requires explicit declaration of `usesNonExemptEncryption` (build was uploaded with that attribute=None) | `PATCH /v1/builds/{id}` with `{usesNonExemptEncryption: false}` before assign |
| 26284313968 | post-export-compliance retry | **success** ✓ | processingState=VALID + usesNonExemptEncryption=False + POST to betaGroups → HTTP 204 | Folded both fixes into ios.yml + asc-assign-build.yml permanently |

## 2026-05-23 — Build visibility to testers (open work, this session)

Filed as **#106**.

Build 14 is in ASC, processingState=VALID, attached to Founders group. But:
- `/v1/betaTesters/{id}/builds` returns `[]` for both `hatyil@gmail.com` and `emrahbaysal@gmail.com`
- Founder reports Cinova doesn't appear in iPhone TestFlight (signed in as hatyil)
- Apple invitation emails to gmail.com unreliable (emrahbaysal got nothing; hatyil's invite "revoked or invalid" on forward)

| Attempt | Mechanism | Result |
|---|---|---|
| 1 | `POST /v1/betaTesters` (email-invite flow) | HTTP 201 but state=None for both |
| 2 | `POST /v1/betaTesterInvitations` (explicit resend) | HTTP 201, no email delivery |
| 3 | Delete 9 duplicate tester records + single clean re-create | state=INVITED but builds still empty |
| 4 | `PATCH Founders.hasAccessToAllBuilds=true` | Apple rejects: "attribute can not be included in UPDATE operation" |
| 5 | External "Public Beta" group + publicLink `testflight.apple.com/join/mXU72Eaz` | Created ✓ but pre-Beta-Review HTML shows "this beta isn't accepting any new testers right now" |
| 6 | Beta App Review submission (`POST /v1/betaAppReviewSubmissions`) | HTTP 422 "Missing required data" even after beta description + build localization populated |
| 7 | `POST /v1/builds/{id}/relationships/individualTesters` (direct attach) | ✅ HTTP 409 (already attached); verify confirms `tester sees 1 builds: ['14']` for both — earlier `0 builds` was eventual-consistency lag |
| 8 | `PATCH /v1/buildBetaDetails/{id}` autoNotifyEnabled toggle false→true | ✅ HTTP 200 — Apple's TestFlight push dispatched |
| 9 | Probe `POST /v1/builds/.../notifyBetaTesters` + variants | ❌ HTTP 404 (endpoint does not exist; Apple uses internal cron, not API trigger) |

Build 14 final state (2026-05-23 07:13 UTC):
- `internalBuildState: IN_BETA_TESTING` — Apple is actively distributing
- `autoNotifyEnabled: True`
- 2 individualTesters attached, state=INVITED
- Per-tester `/builds` query: `1 builds visible` (build 14) for each
- Walk-screenshot DoD: open Cinova on iPhone TestFlight signed in as hatyil@gmail.com, screenshot → #106

Autonomous walk-evidence posted (2026-05-23 11:01 UTC):
- `docs/walks/2026-05-21-cinova-build14-launch.png` committed (`272b1a7`) — real iOS Simulator screenshot of io.dynolabs.cinova running build 14 from CI run 26224865258
- Posted to #106 as comment 4525116734 with embedded image
- Same `Cinova.app` bytes that uploaded to TestFlight as build 14
- Real-device iPhone TestFlight screenshot is the next walk artefact (closes #106 fully)

Cleared substrate (persistent):
- Apple Developer bundle ID `io.dynolabs.cinova` (T8F2BSD4H7), PUSH_NOTIFICATIONS capability
- ASC app record "Cinova - Movies & TV" (id 6772122414), Founders + Public Beta groups, 2 testers, build 14 VALID + assigned to Founders, beta description + build-14 "whatsNew" localization populated

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

## 2026-05-23 — Multi-surface walk evidence captured for #106

| Run | SHA | Outcome |
|---|---|---|
| 26331168746 | 38f7fae | cancelled (superseded) |
| 26331165699 | 38f7fae | failure (point-coords brittle in 02-tab-walk.yaml) |
| 26331392307 | 331aecb | success — Maestro ran but takeScreenshot output not uploaded (artifact step missed walk-*.png at workspace root) |
| 26331927878 | ff6730e | failure — transient `Could not resolve host: github.com` during CocoaPods libavif fetch |
| 26331965196 | ff6730e | **SUCCESS** — full 6-frame walk evidence captured |

`.maestro/02-tab-walk.yaml` walks Home → Reels → Discover → Watchlist → Profile → Chat FAB and `takeScreenshot`s each surface.

`ios.yml` patched (`ff6730e`) to copy walk-*.png from `$GITHUB_WORKSPACE` to `$RUNNER_TEMP/maestro-screenshots/` before artifact upload — without that, Maestro's takeScreenshot output lands at workspace root and gets discarded.

Screenshots committed to `docs/walks/walk-{1..6}-*.png` (`caa7e5b`) and posted to #106 comment 4525406913.

Build 18 (latest) also uploaded + VALID + IN_BETA_TESTING during this session via `ios.yml` run 26331392307 — `asc-assign-build.yml` run for build 18 currently in-flight to attach the existing testers.

## 2026-05-23 — Issue board cleanup (founder directive)

11 stale/shipped/obsolete issues closed in one pass:

| # | Title | Reason |
|---|---|---|
| 7  | TMDB bulk load                                    | Shipped — `backend/cmd/ingestion/` mode=bulk; backend parked #107 |
| 56 | Expo EAS build config                             | EAS retired in #101 |
| 69 | Per-user scoring profile presets                  | Shipped — `backend/internal/scoring/` |
| 71 | Discover Reels synopsis + award badge             | Shipped — walk evidence in #106 |
| 73 | EAS automated iOS+Android workflow                | EAS retired in #101 |
| 74 | K8s CronJob daily delta                           | Shipped — openova-private cronjobs |
| 75 | Wikidata enrichment for TV                        | Shipped — `backend/internal/wikidata/` |
| 76 | Genre/theme chips clickable                       | Shipped — #96 tracks filter behavior bug |
| 77 | Tab icons '?' glyphs                              | Shipped — fix `e4bc827`, walk evidence #106 |
| 78 | Actor+movie detail spinner                        | Shipped — content gated on #107 backend re-enable |
| 105| ASC app record (Apple-UI blocker)                 | Resolved — founder created record 2026-05-22 |

Filed: #110 (rotate OPENOVA_PRIVATE_PAT — backend.yml deploy step fails Bad credentials post-rotation)

Remaining open work: 28 issues (#7 closed dropped count from 29). All have valid scope; none are rubbish. Major buckets:
- 3 infra (#103, #107, #110) — date-anchored / parked / blocked-ext
- 1 testflight UAT (#106)
- 6 mobile in-progress (Cast, app-store metadata, ASC+Play setup, etc.)
- 17 mobile UAT awaiting walks (most gated on #107 backend re-enable)

## 2026-05-24 — BACKLOG-STANDARDS audit

Driven by [dynolabs-io/workflow#BACKLOG-STANDARDS.md](https://github.com/dynolabs-io/workflow/blob/main/docs/BACKLOG-STANDARDS.md). Triage decision tree applied per-issue. Best-effort first pass:

| Action | Count | Issues |
|---|---|---|
| A (fix in place) | 4 | #45 (Chromecast → parked) · #89, #90, #91 (reels bugs — schema + severity) |
| B (close + re-file) | 3 | #54 → #113 · #55 → #112 · #57 → #111 |
| C (close as not-planned) | 1 | #86 (shipped) |

### State before → after

| Bucket | Before | After |
|---|---|---|
| `status/in-progress` | 4 | **0** (all stale claims demoted/re-filed) |
| `status/uat` | 18 | 18 |
| `status/parked` | 1 (#107) | **2** (added #45) |
| Unclaimed | 12 | 14 (added 3 new conformant #111/112/113; closed 1 obsolete #86) |
| Total open | 35 | **30** |

### Labels canon now applied

- Created repo labels: `severity/p0`, `severity/p1`, `severity/p2`, `severity/p3`, `blocker` (none existed before).
- Applied `severity/p2` to reels-bug cluster #89/90/91/96.
- `status/blocked-ext` was created earlier this session for #105/#110 (both now closed).

### Remaining work for next audit pass

Unclaimed backlog still has 11 March-21-era issues without schema conformance (#81, #88, #92, #93, #94, #95, #96). All have intent-clear bodies but use `## Bug` / `## Goal` / `## Tasks` rather than the canonical `## Problem` / `## Acceptance criteria` / `## Out of scope` / `## Repos touched`. Each needs Action A reformat (~5 min/issue). Defer to next focused audit session.
