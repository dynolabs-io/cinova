# Glossary

> **WHAT**: Canonical terms used in Cinova code, docs, and PR/issue language. Plus a banned-terms list — synonyms we don't use because they create confusion.
> **AUTHORITY**: 📐 Canon. If you see a banned term in code or docs, flag it in PR review. If you find a missing term that should be canonical, add it here (don't reinvent vocabulary elsewhere).

## Canonical terms

### Product & scoring

- **CinovaScore** — the 0–100 quality signal Cinova displays in place of raw star ratings. Computed at ingest time, stored as `cinova_score` in PostgreSQL, surfaced on every list/detail card. See [ARCHITECTURE.md § CinovaScore](ARCHITECTURE.md#cinovascore--computation) for the formula.
- **Bayesian rating** — the smoothed average `(v * R + m * C) / (v + m)` used as the 80% input to CinovaScore. `m=1000` is the prior weight; `C≈6.5` is the corpus mean.
- **Graph prestige** — the 0–1 PageRank-like signal from the Neo4j influence graph; the 20% input to CinovaScore. Measures how many other directors a title's directors have influenced.
- **CinovaScore tier** — the human-readable label band:

  | Score | Tier |
  |---|---|
  | 85–100 | Masterpiece |
  | 75–84 | Excellent |
  | 65–74 | Great |
  | 55–64 | Good |
  | 45–54 | Average |
  | 0–44 | Below Average |

- **Scoring profile** — a per-user preset that biases recommendations:

  | Preset | Bias |
  |---|---|
  | Mainstream | Default — broad commercial taste |
  | Cinephile | Higher graph-prestige weight |
  | Arthouse | International + festival-circuit weight |
  | Blockbuster | Higher vote-count + recency weight |
  | AwardSeason | Award + critic-score weight |

- **Influence Graph** — the Neo4j graph of `(:Person)-[:INFLUENCED]->(:Person)` and `(:Movie/:TV)-[:HAS_PERSON]->(:Person)` relationships sourced from Wikidata.
- **Discover Reels** — the vertical-swipe trailer feed surface in the mobile app (mobile route `(tabs)/discover`). Each "reel" is one vertical YouTube trailer.
- **Vertical trailer** — a portrait-orientation YouTube clip ingested via the `--mode=vertical-trailers` job. Stored as `youtube_key_vertical` on the movie/TV row.
- **Watchlist** — the user's collection of `saved` + `rated` + `dismissed` interactions, queryable via `/api/v1/me/watchlist`.

### Infrastructure & integration

- **Axon** — the OpenOva-internal Claude API gateway. Cinova calls Claude only through Axon — never `api.anthropic.com` directly. See [PRINCIPLES.md](PRINCIPLES.md) rule 6.
- **Langfuse** — LLM-call observability layer. Every Axon call from cinova produces a Langfuse trace.
- **Langflow** — workflow-engine integration used for AI chat orchestration.
- **NL → Cypher** — the natural-language search pipeline: user text → Axon (Claude) → Cypher query → Neo4j → ranked results.
- **TMDB delta** — daily ingestion mode that pulls TMDB's `changes` endpoint since last run.
- **TMDB bulk** — first-time / new-country ingestion mode that pulls full TMDB catalogue. Takes hours.
- **Provider sync** — the 6-hour CronJob that refreshes streaming-provider availability per country.
- **Sovereign instance** — a deployed OpenOva platform instance. Cinova currently deploys to the `contabo-mkt` Sovereign (legacy). See user-global `~/.claude/CLAUDE.md` §0.
- **`openova-private`** — the legacy Sovereign instance repo at `openova-io/openova-private`. Holds the Flux manifests under `clusters/contabo-mkt/apps/cinova/`.

### Process

- **Operator walk** — the DoD gate 4 walk: open the mobile app on a real device, exercise the surface, screenshot the result, attach to the issue. See [DOD.md](DOD.md).
- **`status/uat`** — Kanban column for "code done, walk pending". Per user-global §6, mutually exclusive with other `status/*` labels — remove the old before adding the new.

## Banned terms

Do not use these in code, docs, commit messages, or PR descriptions. Use the canonical replacement.

| Banned | Use instead | Why |
|---|---|---|
| `score` (bare, in ranking contexts) | `cinova_score` | "score" is ambiguous when TMDB rating, RT score, IMDB score, and CinovaScore all coexist. |
| `recommendation` (in user-visible copy) | "For You" / "Discover" | "Recommendation" is an internal/back-end term. |
| `personalised feed` (in code) | `for_you_feed` | Consistency with the API surface. |
| `Anthropic SDK` (in cinova backend Go code) | `Axon client` | All Claude calls route through Axon. |
| `Redis` (in new code) | `Valkey` | Valkey is the deployed implementation; the protocol is Redis-compatible but the canonical name is Valkey. |
| `Open AI` / `OpenAI` (in cinova backend Go code) | Don't — cinova does not use OpenAI. | The AI provider is Anthropic via Axon. |
| `users without an account` | `anonymous users` / `anonymous session` | Canonical term. |
| `MVP` / `for now` / `quick fix` (in 📐 PERMANENT docs) | nothing — finish or don't ship | See [PRINCIPLES.md](PRINCIPLES.md) anti-pattern catalog. |
| `cluster` (in docs referring to deploy target without qualification) | `contabo-mkt` (the specific Sovereign) | "The cluster" is ambiguous across OpenOva ecosystem. |
