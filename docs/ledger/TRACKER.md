# Tracker — open work + DoD progress

> **WHAT**: Live snapshot of open work bucketed by area + status. Cron-refreshable; current snapshot is hand-rolled until that automation lands.
> **AUTHORITY**: 🟢 LIVE STATE. Refresh when issue board state materially changes (≥ 5 issue label changes).
> **POINTER**: Per-surface verification state in [TRUST.md](TRUST.md). What's shipped vs design in [../STATUS.md](../STATUS.md).
> **AS-OF**: 2026-05-21

## Counts

| Bucket | Count |
|---|---|
| Total open | ~42 |
| `status/in-progress` | 6 |
| `status/uat` | 19 |
| `status/completed` (awaiting user closure) | 7 |
| Untriaged (no `status/*` label) | ~10 |

## In progress (`status/in-progress`)

| # | Title | Area |
|---|---|---|
| 45 | Google Cast (Chromecast) integration for trailer casting | mobile |
| 54 | iOS App Store: metadata, screenshots, privacy policy | mobile |
| 55 | Google Play Store: metadata, screenshots, content rating | mobile |
| 56 | Expo EAS build configuration for both stores | ci-cd |
| 57 | App Store Connect + Play Console setup | mobile |
| 98 | docs: bring docs/ tree to canonical §11 shape | infra |

## UAT (`status/uat`) — code done, walk pending

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
| 67 | feat: capture all valuable TMDB fields (keywords, tagline, …) | backend/data |
| 77 | fix(mobile): tab icons show '?' glyphs | mobile |
| 79 | fix(mobile): chat send button icon renders incorrectly | mobile |
| 80 | fix(mobile): TrailerPlayer fullscreen — missing progress bar | mobile |
| 82 | feat(mobile): voice input for AI chat — mic button with STT | backend/mobile |
| 83 | feat(mobile): rate button on discover reels — show dialog | mobile |
| 84 | fix(backend+mobile): AI chat quality — country hardcoded US | backend/mobile |
| 85 | fix(mobile): add back navigation to auth/login and auth/signup | mobile |
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

| # | Title | Suggested area |
|---|---|---|
| 81 | feat(mobile): show additional quality scores (IMDB, RT) | mobile/data |
| 86 | feat(data+backend): vertical trailer ingestion — search/store portrait keys | backend/data |
| 88 | feat(mobile): home page — personalised For You feed with AI reasons | backend/mobile |
| 89 | bug(mobile): reels — title/metadata text masked by bottom tab bar | mobile |
| 90 | bug(mobile): reels — provider icons too small / blank | mobile/data |
| 91 | bug(mobile): reels — Watch on Provider always opens App Store | mobile |
| 92 | feat(mobile): reels — add tap affordance to drill into detail | mobile |
| 93 | feat(mobile): floating AI assistant button | mobile |
| 94 | feat(mobile): movie detail — replace static poster with inline trailer | mobile |
| 95 | feat(mobile+backend): clickable awards — drill into winners | backend/mobile |
| 96 | bug(mobile): genre/theme/mood chips navigate to unfiltered discover | mobile |

## Refresh protocol

When ≥ 5 label changes happen since the AS-OF date, regenerate this snapshot:

```bash
gh issue list --repo foundrylab-app/cinova --state open --limit 100 \
  --json number,title,labels \
  | jq -r '.[] | "\(.number)\t\([.labels[].name] | join(","))\t\(.title)"' \
  | sort -n
```

Then hand-bucket into the tables above. Until automation lands, this is operator-maintained.
