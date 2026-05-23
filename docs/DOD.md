# Definition of Done

> **WHAT**: What "done" means for a Cinova change. Stricter than user-global default — operator must walk the user-facing surface on a real device and attach a screenshot to the issue.
> **AUTHORITY**: 📐 Canon. Per user-global `~/.claude/CLAUDE.md` §2, a project may set a stricter DoD than the default; never weaker. This document is that stricter DoD.
> **POINTER**: Architecture in [ARCHITECTURE.md](ARCHITECTURE.md). Per-PR engineering rules in [PRINCIPLES.md](PRINCIPLES.md). Verification ledger in [ledger/TRUST.md](ledger/TRUST.md).

## The cinova DoD — five gates, all required

A change is done when **all five gates** pass. "PR merged + tests green" is **not** done.

### 1. Code-complete gate

- Tests pass: `cd backend && go test ./...` is green.
- TypeScript clean: `cd mobile && npx tsc --noEmit --skipLibCheck` reports zero errors.
- Linters clean: `cd backend && go vet ./...` is green.
- No new defensive null-guards, `?? 'default'`, `enabled: false` defaults, or "for now / MVP" comments (see [PRINCIPLES.md](PRINCIPLES.md) anti-pattern catalog).
- Commit messages are conventional (`feat:`, `fix:`, `chore:`, `docs:`, `revert:`, `infra:`, `ci:`).

### 2. CI gate

- Required workflows are green on the PR's HEAD SHA:
  - `backend.yml` (if backend touched) — `build-api`, `build-ingestion`, `deploy` all succeeded.
  - `mobile.yml` (if mobile touched) — `tsc` step succeeded.
  - `eas-build.yml` — runs async to TestFlight / Play Console; not strictly required for merge but tracked separately.
- **No admin-merge through red CI.** "Pre-existing failure" is not a bypass — fix the check first.
- **No `--no-verify`, `--no-gpg-sign`, or hook-skip flags.**

### 3. Deploy gate (backend changes only)

After merge:

- GHCR tag `ghcr.io/dynolabs-io/cinova/api:<short-sha>` exists.
- `openova-private/clusters/contabo-mkt/apps/cinova/services/api.yaml` is bumped to that SHA on `main`.
- Flux reconcile log shows the deployment rolled out cleanly: `kubectl -n cinova rollout status deploy/api` returns `successfully rolled out`.
- `curl https://api.cinova.openova.io/healthz` returns 200.
- `curl https://api.cinova.openova.io/readyz` returns 200 (DB + cache check).

### 4. Operator-walk gate (the canonical proof)

Per user-global `~/.claude/CLAUDE.md` §2 + §3, "PR merged" is not progress — "operator walks the surface on a fresh provision + screenshot attached to the issue" is.

For each cinova surface touched, walk it on a real device or simulator and attach the screenshot:

| Surface | Walk |
|---|---|
| Home tab | Open app → trending feed renders within 2s → tap a poster → movie detail loads with CinovaScore badge + tier label |
| Discover (reels) | Swipe through ≥ 5 vertical trailers → each autoplays → tap "Save" persists across app restart |
| Search | Type a NL query ("slow-burn thriller set in Scandinavia") → AI search returns ≥ 3 hits → tap a hit lands on detail |
| Watchlist | Save 2 items from different surfaces → Watchlist tab shows both → remove one persists across restart |
| AI chat | Send a free-form recommendation request → response cites at least one specific title with reasoning |
| Auth | Anonymous → signup with email → existing anonymous watchlist migrates → restart app → still logged in |

The screenshot lands as a comment on the GitHub issue. **Verification is READ-ONLY** per `~/.claude/CLAUDE.md` §3 rule 6 — no PRs from the verifier, no production patches to make the walk pass.

### 5. Trust-ledger gate

[`ledger/TRUST.md`](ledger/TRUST.md) carries the verification state per surface. When an operator-walk passes:

1. Mark the surface 🟢 **VERIFIED-PASS** with the screenshot URL + date + SHA walked.
2. Update [`ledger/TRACKER.md`](ledger/TRACKER.md) — move the issue from open to "awaiting user closure".
3. **Do NOT close the issue yourself.** Only the user closes after they accept the walk evidence.

Every new PR that touches the same surface flips the ledger entry back to 🔴 **UNVERIFIED** until the next walk.

## Issue/PR linkage discipline

Per `~/.claude/CLAUDE.md` §3 rule 1:

- **Default to `Refs #N` in PR bodies.** Auto-close on merge is the enemy — the issue closes only after operator-walk-with-screenshot lands.
- **Exception**: pure CI-gate / docs-only fixes with no operator-visible surface MAY use `Closes #N`.
- This docs PR (issue #98) is documentation-only — does not change runtime behaviour — but per the founder mandate ("DO NOT close any issue yourself"), it still uses `Refs #98`. User closes after they accept the shape.

## Tests are not the DoD

`go test` and `tsc` clean is necessary but never sufficient. A walk-shaped bug (e.g., country filter accidentally removed from reels feed, #PR ca307a1 lineage) can pass every test and still flunk Gate 4. The walk is the metric.
