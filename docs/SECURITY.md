# Security

> **WHAT**: Threat model, secrets policy, and identity model for Cinova.
> **AUTHORITY**: 📐 Canon. Privacy policy (user-facing, store-required) lives at `mobile/store-assets/privacy-policy.md` and is the source served to Apple/Google reviewers — keep this doc and that document consistent.
> **POINTER**: Operator how-tos (rotation procedures, manual rollback) in [RUNBOOKS.md](RUNBOOKS.md). Engineering rule "secrets are not committable" in [PRINCIPLES.md](PRINCIPLES.md).

## Identity model

Cinova has two identity tiers; both are JWT-backed.

| Tier | Created when | Persistence | What they can do |
|---|---|---|---|
| **Anonymous session** | App first launch (mobile auto-issues `POST /api/v1/auth/session`) | Local device storage; server-side `users` row with `email = NULL` | Save, rate, dismiss content. Recommendations personalise to their interaction history. |
| **Registered user** | `POST /api/v1/auth/signup` with email + password (migrates the anonymous session in-place) | Same `users` row, now with `email` set | Everything above + cross-device sync via login. |

Tokens:

- **Access JWT** — short-lived (15 min), signed with `JWT_SECRET` (HS256).
- **Refresh token** — long-lived (30 days), rotated on every `/auth/refresh` call. Revocable via `/auth/logout`.

Anonymous-to-registered migration preserves the `user_id` so existing watchlist/ratings carry over.

## Data classification

| Data | Sensitivity | Storage | Notes |
|---|---|---|---|
| Email + password hash | High | PostgreSQL `users` table | Passwords stored as bcrypt hash; never logged. |
| Watchlist, ratings, dismissals | Medium | PostgreSQL | Tied to `user_id`. |
| Anonymous session UUID | Low | Device local + PG `users.id` | Random UUIDv4. |
| Country preference | Low | PostgreSQL `users.country` | ISO 3166-1 alpha-2. |
| Push token | Medium | PostgreSQL `push_tokens` | One per device; revoked on logout. |
| Search history | Low | Valkey (1h TTL) + Langfuse traces | Used to improve NL search quality. |
| AI chat history | Medium | PostgreSQL (chat sessions table) + Langfuse traces | Linked to `user_id`. |
| TMDB IDs, movie/TV metadata | Public | PostgreSQL + Neo4j | Sourced from TMDB API; not user data. |

## Secrets policy

**No secret may be committed to this repo.** Verifiers: `git secrets` / `truffleHog` / `gitleaks` on pre-commit (add when CI capacity allows).

| Secret | Where it lives | Rotation |
|---|---|---|
| `TMDB_API_KEY` | `openova-private` Sealed Secret → K8s Secret `cinova-tmdb` | TMDB dashboard; rotate every 12 months or on suspected compromise. |
| `JWT_SECRET` | `openova-private` Sealed Secret → K8s Secret `cinova-jwt` | Rotate on incident; rotation invalidates all in-flight access tokens (acceptable — clients hold refresh tokens). |
| Postgres credentials | `openova-private` Sealed Secret → K8s Secret `cinova-postgres` | Rotate every 6 months. |
| Neo4j credentials | `openova-private` Sealed Secret → K8s Secret `cinova-neo4j` | Rotate every 6 months. |
| Valkey password | `openova-private` Sealed Secret → K8s Secret `cinova-valkey` | Rotate every 6 months. |
| `AXON_TOKEN` | `openova-private` Sealed Secret → K8s Secret `cinova-axon` | Tied to Axon gateway's token lifecycle; rotate on rotation of Axon. |
| `OPENOVA_PRIVATE_PAT` | GitHub repo secret on `dynolabs-io/cinova` | Rotate every 90 days; tied to the cinova-bot machine user. |
| `EXPO_TOKEN` | GitHub repo secret + EAS dashboard | Rotate every 90 days. |
| Apple Developer / Google Play credentials | EAS dashboard only | Apple-cert lifecycle; Google service-account key 12 months. |

Local dev uses `.env` files (gitignored) populated from `.env.example`. Do not commit a populated `.env`.

## Threat model (STRIDE summary)

| Threat | Vector | Mitigation |
|---|---|---|
| **Spoofing** — fake JWT | Forged token without secret | HS256 with rotated `JWT_SECRET`; short access TTL |
| **Tampering** — modified watchlist | Direct DB write | DB access only from API pods; pod NetworkPolicy blocks lateral traffic |
| **Repudiation** — "I didn't rate that" | User claims a rating wasn't theirs | All write endpoints are JWT-authenticated; user_id stamped on every interaction row; Langfuse trace per AI call |
| **Information disclosure** — leaked email/watchlist | DB dump / API enumeration | API enforces `user_id == jwt.sub`; rate limiting on auth endpoints; no PII in logs |
| **DoS** — flood search endpoint | Naïve client / abuse | Valkey-backed rate limit per IP + per user; Axon side has its own quota |
| **Elevation of privilege** — anonymous → admin | Privilege escalation | No admin role exists in the API; admin actions are out-of-band via cluster kubectl |

## Cryptography

- Passwords: bcrypt, cost factor 12.
- JWT: HS256, `JWT_SECRET` from sealed secret.
- TLS termination at the cluster ingress (cert-manager + Let's Encrypt) — backend never sees plaintext on the wire from clients.
- All secret storage at rest is via Kubernetes Sealed Secrets in `openova-private`.

## Reporting a vulnerability

Email `security@cinova.app`. Coordinated disclosure timeline: acknowledgement within 48 hours; patch + public advisory within 90 days for moderate, 30 days for critical.

## Per-user privacy summary (consumer-facing)

The store-facing privacy policy at `mobile/store-assets/privacy-policy.md` is the authoritative consumer-facing version (Apple/Google reviewers read that one). This doc covers the engineering view; that one covers the user-facing commitments. Keep them in sync — when one changes, the other must be checked.
