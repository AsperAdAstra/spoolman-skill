# Spoolman Skill — Design Plan

A skill that targets **Spoolman** — a self-hosted web service purpose-built for 3D-printer filament. Spoolman has a REST API but **no official CLI**, and there is a companion community database (**SpoolmanDB**) that's worth validating against.

This skill has two responsibilities:

1. **`spoolctl`** — a CLI for the Spoolman REST API (the missing piece), including all inventory read/write operations.
2. **`spoolctl db`** — subcommands that fetch and validate against SpoolmanDB's compiled `filaments.json` / `materials.json`, including comparison and auto-correction decisions.

Role split is strict: data retrieval, normalization, validation, storage, and comparison logic live in CLI commands; the Agent Skill orchestrates command sequencing, image-derived extraction, and user confirmation.

Everything is shipped as a single Anthropic-spec agent skill (`SKILL.md` + `scripts/` with prebuilt binaries), following a "thin skill, fat binary" split.

---

## 0. Decisions locked

- **Binary name:** `spoolctl` (one static Go binary, subcommands for api / db).
- **Language:** Go. Single static binary, zero runtime deps.
- **Spoolman base URL:** **remote servers are a first-class use case** — no silent fallback behavior.
  1. `--server URL` flag (per-invocation override).
  2. `SPOOLMAN_URL` env var (primary, documented, LLM-friendly).
  3. `~/.config/spoolctl/config.toml` → `server = "…"`.
  4. If none are set, default target is `http://localhost:7912/api/v1` (explicit default, not a retry fallback).
  URL input must include protocol and host (`http://` or `https://`). Prefix handling:
  - If the provided URL includes no path, client appends `/api/v1`.
  - If the provided URL already includes a path prefix, use it as-is (do not append or rewrite).
- **Extra env knobs** (all optional): `SPOOLMAN_TIMEOUT` (e.g. `10s`, default `10s`), `SPOOLMAN_INSECURE=1` (skip TLS verify for self-signed LAN certs), `SPOOLMAN_CA_CERT=/path/to/ca.pem` (custom CA bundle). No auth headers in v1 — if Spoolman ever adds auth, reserve `SPOOLMAN_TOKEN` for it.
- **`spoolctl env`** command prints the resolved config (masked) so Claude can verify where it's pointing before mutating.
- **Authentication:** none. Spoolman has no built-in auth — it's self-hosted LAN-only. Skill does not add its own auth layer.
- **API path policy:** default to **`/api/v1`** when URL path is unspecified. If the caller provides an explicit path prefix in `--server` / `SPOOLMAN_URL` / config, use that prefix exactly as provided.
- **SpoolmanDB access:** fetch `https://donkie.github.io/SpoolmanDB/filaments.json` and `/materials.json` (the compiled endpoints). Cache under `~/.cache/spoolctl/`. Cache policy is simple: if cache files exist, commands read from cache; `spoolctl db refresh` forcibly re-fetches and replaces cache.
- **Currency:** inherited from Spoolman's own server settings — the skill does not override.
- **No migration, no interop with other local trackers.** Spoolman is the sole source of truth. Initial inventory is re-entered through the CLI (UAT). Keeps the skill small and focused.
- **Skill packaging:** Anthropic Agent Skill spec. Binaries live in `scripts/` (per user's instruction) and `SKILL.md` invokes them with `./scripts/spoolctl …`.

---

## 1. Design principles

- **The CLI is a thin wrapper over the REST API.** No local cache of spools/filaments/vendors — Spoolman is the source of truth. The only cache is for SpoolmanDB's static JSON files.
- **One binary, subcommand tree.** Keeps installation and packaging trivial; keeps the skill's tool-invocation surface small.
- **CLI owns business logic.** Validation, normalization, and comparison outcomes are produced by `spoolctl`, not reimplemented in SKILL.md prompts.
- **Skill is an orchestrator.** The agent interprets intent, runs `spoolctl` commands in safe order, handles image extraction, and requests confirmation when required.
- **Fail loud on drift.** Validation against SpoolmanDB is advisory (warns, does not block) unless `--strict` is passed.
- **No hidden mutations.** Every write command echoes the server's response JSON so Claude (and the operator) can see exactly what happened.
- **LLM-optimal I/O.** API/db commands support stable JSON; a dedicated `context` command emits compact LLM-oriented text (not human prose, not JSON) for efficient grounding. Errors go to stderr as `{"error":"…","status":N}` with non-zero exit codes.

---

## 2. Scope — what the skill actually does

### 2a. `spoolctl` — Spoolman API client

Covers the resources Spoolman exposes under `/api/v1`:

| Resource | Spoolman endpoints (from source) | CLI surface |
|---|---|---|
| Vendor | `GET/POST /vendor`, `GET/PATCH/DELETE /vendor/{id}` | `spoolctl vendor {list,get,add,edit,rm}` |
| Filament | `GET/POST /filament`, `GET/PATCH/DELETE /filament/{id}` | `spoolctl filament {list,get,add,edit,rm}` |
| Spool | `GET/POST /spool`, `GET/PATCH/DELETE /spool/{id}`, `PUT /spool/{id}/use`, `PUT /spool/{id}/measure` | `spoolctl spool {list,get,add,edit,rm,use,measure}` |
| External DB | `GET /external/filaments`, `GET /external/materials` (Spoolman's own pass-through of SpoolmanDB) | `spoolctl db {filaments,materials}` (server-side) and `spoolctl db upstream …` (direct to SpoolmanDB) |
| Info / health | `GET /info`, `GET /health` | `spoolctl info`, `spoolctl health` |
| Agent context | Composite read-only snapshot for LLM grounding | `spoolctl context` |
| Settings / fields | `setting.*`, `field.*` | Phase 2 — not in v1. |
| Websockets | `/` and per-resource WS | Out of scope for a CLI (one-shot tool). |

Global flags: `--server URL`, `--json` (default on for scriptable commands), `-q`, `--timeout`, `--verbose`.

### 2b. `spoolctl db` — SpoolmanDB validator

Two sources of truth, selectable per-invocation:

- `spoolctl db --source=spoolman` (default) — hits the local server's `/api/v1/external/{filaments,materials}` endpoint. Reflects whatever SpoolmanDB snapshot that Spoolman instance ships with.
- `spoolctl db --source=upstream` — reads local cache when present; otherwise fetches `https://donkie.github.io/SpoolmanDB/{filaments,materials}.json` and seeds cache.

Commands:

- `spoolctl db filaments [--manufacturer M] [--material PLA] [--diameter 1.75]` — list/filter.
- `spoolctl db materials` — list materials (`PLA`, `PETG`, `ABS`, …) with density / temp defaults.
- `spoolctl db lookup <filament-id>` — show one record (e.g. `3d-fuel_pla+_almond_1000_175_n`).
- `spoolctl db validate --file <toml|json>` — validate a filament spec against SpoolmanDB (checked fields: `material`, `density`, `extruder_temp`, `bed_temp`, `spool_weight`, `diameter`). Reports `match | near-match | unknown` per field.
- `spoolctl db refresh` — force cache refresh (single-command refresh path).
- `spoolctl db diff --server <url>` — show drift between a running Spoolman's external DB and upstream SpoolmanDB (useful for "does my Spoolman need an update?").

The validator doesn't mutate anything. It emits a JSON report that `spoolctl filament add` can consume via `--from-db <id>` to auto-fill fields.
When `--verbose` is enabled, every db command must print cache provenance (`source=cache` or `source=network`) so operators can confirm where records came from.

---

## 3. CLI surface (full tree)

```
spoolctl
├── env                      # prints resolved server URL, timeout, TLS, config-source
├── info                     # GET /info
├── health                   # GET /health
├── context                  # compact LLM context snapshot (non-JSON)
│
├── vendor
│   ├── list [--name X]      # GET /vendor
│   ├── get <id>             # GET /vendor/{id}
│   ├── add --name ... [--comment ...] [--extra k=v]
│   ├── edit <id> --set k=v
│   └── rm <id>
│
├── filament
│   ├── list [--vendor V] [--material PLA] [--json]
│   ├── get <id>
│   ├── add --vendor-id N --material PLA --diameter 1.75 ...
│   ├── add --from-db <spoolmandb-id>        # autofill from SpoolmanDB
│   ├── edit <id> --set k=v
│   └── rm <id>
│
├── spool
│   ├── list [--filament F] [--archived] [--json]
│   ├── get <id>
│   ├── add --filament-id N [--initial-weight 1000] [--price 25] [--location ...]
│   ├── edit <id> --set k=v
│   ├── use <id> --weight 42                 # PUT /spool/{id}/use
│   ├── use <id> --length 1200 --ref print-0421
│   ├── measure <id> --weight 950            # PUT /spool/{id}/measure
│   └── rm <id>
│
├── db
│   ├── filaments [--manufacturer M] [--material PLA] [--diameter 1.75]
│   ├── materials
│   ├── lookup <spoolmandb-id>
│   ├── validate --file spec.toml [--strict]
│   ├── diff                                 # server.external vs upstream
│   └── refresh
│
└── completion {bash|zsh|fish}
```

Global flags on every command: `--server URL`, `--json`, `-q`, `--timeout 10s`, `--verbose`.

All write commands exit non-zero on HTTP 4xx/5xx and emit `{"error":"...","status":409,"detail":{...}}` on stderr.

---

## 4. Data contracts

### 4a. Spoolman resource JSON (pass-through)

`spoolctl` does **not** re-model Spoolman's types — it emits whatever Spoolman returns, verbatim. This avoids schema drift when Spoolman adds fields. Go structs are only used internally for request bodies.

For each resource we ship a reference doc generated from Spoolman's own pydantic models (captured at build time from `spoolman/api/v1/models.py`) into `SKILL.md` appendix, so Claude knows the field names without calling `get` first.

### 4b. SpoolmanDB filament record (upstream)

Captured from live endpoint:

```json
{
  "id": "3d-fuel_pla+_almond_1000_175_n",
  "manufacturer": "3D-Fuel",
  "name": "Almond",
  "material": "PLA+",
  "density": 1.22,
  "weight": 1000.0,
  "spool_weight": 225,
  "spool_type": null,
  "diameter": 1.75,
  "color_hex": "CFBCAE",
  "color_hexes": null,
  "extruder_temp": 220,
  "extruder_temp_range": null,
  "bed_temp": 60,
  "bed_temp_range": null,
  "finish": null,
  "multi_color_direction": null,
  "pattern": null,
  "translucent": false,
  "glow": false
}
```

### 4c. SpoolmanDB material record

```json
{ "material": "PLA", "density": 1.24, "extruder_temp": 210, "bed_temp": 50 }
```

### 4d. `spoolctl context` output contract (LLM format)

`spoolctl context` returns a compact line-oriented format designed for LLM parsing and token efficiency. It is intentionally neither human prose nor JSON.

Example:

```text
CTXv1 server=https://spoolman.lan/api/v1 health=ok source=db:cache ts=2026-04-24T12:34:56Z
COUNTS vendors=12 filaments=87 spools=143 low=9 archived=21
MATERIALS PLA=61 PETG=42 ABS=11 ASA=8 TPU=7 OTHER=14
LOW_SPOOLS id=44:81g:PLA-Basic-Black|id=78:95g:PETG-Gray|id=121:102g:ASA-White
RECENT_USE id=102:-42g:benchy-0421:2026-04-22|id=44:-18g:bracket-17:2026-04-23
DB_STATE source=upstream cache_age=3h suggested_refresh=false
```

Rules:
- Deterministic ordering and stable keys.
- Single-line sections with bounded cardinality (truncate with `...+N` suffix).
- Include provenance (`source=db:cache|db:network`) and timestamp.
- Must remain backward-compatible via version tag (`CTXv1`).
- This command is the preferred grounding call for the Agent Skill before planning mutations.

---

## 5. SpoolmanDB validation rules

When `spoolctl db validate --file spec.toml` runs over a filament spec (either a local file or a Spoolman filament payload), it does three passes:

1. **Hard match.** If a SpoolmanDB `id` is supplied or derivable, compare field-by-field. Report any mismatches.
2. **Material sanity.** The spec's `material` must exist in `materials.json`. Density, extruder temp, bed temp should fall within the material's known range (±10 % density, ±15 °C on temps). Outside → warning.
3. **Enum sanity.** `fill`, `finish`, `pattern`, `multi_color_direction`, `spool_type` must be in the SpoolmanDB schema enum set.

`--strict` turns every warning into a non-zero exit. Default mode just annotates.

### 5a. Image-derived data adjudication (agent workflow)

Primary use case: an LLM extracts candidate fields from a spool label image and passes them into validation before any write.

Flow:
1. LLM extracts candidate values from image (`manufacturer`, `name`, `material`, `diameter`, temps, spool weight, color hints).
2. Run `spoolctl db validate --file extracted.json --json`.
3. If only minor drift exists and a high-confidence `suggested_db_id` is returned, auto-correct to SpoolmanDB values and continue.
4. If confidence is low or key fields conflict, stop and ask user for confirmation before any `filament add` / `spool add`.

Auto-correction policy (safe fixes):
- Normalize case/spacing/punctuation variants (`PLA +` -> `PLA+`, manufacturer aliases, trivial name normalization).
- Normalize numeric formatting and units where unambiguous (e.g. `"1.75mm"` -> `1.75`).
- Replace extracted fields with authoritative values from matched SpoolmanDB record when match confidence is high.
- Never auto-correct across conflicting material families or diameters (e.g. PLA vs PETG, 1.75 vs 2.85).

Confirmation gate (must ask user):
- No confident SpoolmanDB match.
- Multiple near-matches with similar score.
- Key conflicts in `material`, `diameter`, or large temp/weight discrepancies.
- Any mutation command requested after validator status is `warn`/`error` with unresolved key conflicts.

Validator output (JSON):

```json
{
  "input": "spec.toml",
  "status": "warn",
  "matches": [ { "field": "diameter", "value": 1.75, "db_value": 1.75 } ],
  "warnings": [ { "field": "extruder_temp", "value": 240, "expected_range": [195,230], "material": "PLA" } ],
  "errors":   [],
  "suggested_db_id": "bambu-lab_pla-basic_black_1000_175_n",
  "match_confidence": "high",
  "auto_corrections": [
    { "field": "material", "from": "PLA +", "to": "PLA+" }
  ],
  "requires_confirmation": false
}
```

---

## 6. Skill package layout (Anthropic Agent Skill)

```
spoolman/
  SKILL.md                   # frontmatter (name, description, triggers) + recipes
  README.md                  # human-facing build/install
  scripts/                   # per user's instruction: binaries live here
    spoolctl                 # darwin-arm64 by default
    spoolctl-darwin-amd64
    spoolctl-linux-amd64
    spoolctl-windows-amd64.exe
  src/
    go.mod
    cmd/spoolctl/main.go
    internal/
      api/        # thin Spoolman REST client
      db/         # SpoolmanDB fetch + cache + validator
      cli/        # cobra command tree
      config/     # ~/.config/spoolctl/config.toml + env resolution
  testdata/
    spoolmandb-snapshot/     # pinned filaments.json / materials.json for tests
    api-fixtures/            # recorded Spoolman responses for client tests
  Makefile
```

**Size budget:** `spoolctl` should stay < 15 MB per platform (Go static binary, no CGO). Four platforms fits comfortably inside the Agent Skill packaging limits.

---

## 7. SKILL.md outline

- **Frontmatter.** `name: spoolman`. Description triggers: "spoolman", "filament spool", "3d printer inventory", "add spool", "record print usage", "filament database", "spoolmandb".
- **When to use.**
  - User mentions Spoolman explicitly, or any filament-specific action (list spools, record print usage, add a new spool, check stock).
  - User wants to validate a filament definition against SpoolmanDB.
- **Who does what.**
  - `spoolctl` owns data operations: read/write/validate/normalize/compare.
  - Agent Skill owns orchestration: understand user request, run commands in order, interpret image extraction output, and request confirmation when required.
- **When NOT to use.** Non-filament inventory (nozzles, solvents, parts). The skill is filament-only by design.
- **Locating the server.** Resolution order: `--server` flag → `$SPOOLMAN_URL` → `~/.config/spoolctl/config.toml` → explicit default `http://localhost:7912/api/v1` when none are set. If the user mentions a remote host (e.g. "my spoolman at spoolman.lan:7912"), either export `SPOOLMAN_URL` for the session or pass `--server`. URLs must include protocol and host. If a path prefix is provided, use it as-is; if no path is provided, append `/api/v1`. Run `scripts/spoolctl env` once to confirm target before mutations. If `health` fails, surface the error — never guess or silently retarget.
- **Quickstart recipe.** `scripts/spoolctl health && scripts/spoolctl context` to ground context efficiently for LLM.
- **Common recipes** (each shows the exact command; all paths relative to the skill):
  - "Ground current state before any change" → `scripts/spoolctl context`
  - "What spools do I have?" → `scripts/spoolctl spool list --json`
  - "Add a new spool of Bambu PLA black" → `scripts/spoolctl db lookup bambu-lab_pla-basic_black_1000_175_n` then `scripts/spoolctl filament add --from-db bambu-lab_pla-basic_black_1000_175_n --vendor-id N` then `scripts/spoolctl spool add --filament-id M --initial-weight 1000 --price 25`
  - "I used 42 g on a print" → `scripts/spoolctl spool use <id> --weight 42 --ref benchy-0421 --json`
  - "Does my filament spec look sane?" → `scripts/spoolctl db validate --file filament.toml --json`
  - "I extracted this from a spool photo, can you add it?" → run `scripts/spoolctl db validate --file extracted.json --json`; if `requires_confirmation=false`, continue with `scripts/spoolctl filament add --from-db <suggested_db_id> ...`; otherwise ask user to confirm conflicting fields before any write.
  - "Is my Spoolman's built-in DB stale?" → `scripts/spoolctl db diff --json`
- **Invariants for Claude.**
  - Prefer `scripts/spoolctl context` for grounding; use JSON commands for detail queries and mutations.
  - Always pass `--json` when parsing API/db command payloads.
  - Never invent spool/filament/vendor IDs — always `list` or `get` first.
  - If `spoolctl health` fails, stop and report — don't retry mutations.
  - For image-derived inputs, run `db validate` before any write. Apply only safe auto-corrections; ask user before writing if `requires_confirmation=true` or key conflicts exist.
- **Error handling.** Every non-2xx surfaces as `{"error":"…","status":N}` on stderr. Surface verbatim; do not retry.
- **Debugging / provenance.** `--verbose` must explicitly show whether SpoolmanDB data came from cache or network.

---

## 8. Build, install, package

- `make build` → `scripts/spoolctl` for current OS/arch.
- `make cross` → all four platforms.
- `make test` → unit tests (api client with recorded fixtures, db validator golden files).
- `make skill` → produces a zipped skill folder ready for Anthropic skill installation; verifies `SKILL.md` frontmatter with the `skill-creator` / `skill-writer` skill at execution time.
- Dependencies: Go stdlib + `cobra` + `pelletier/go-toml/v2`. No HTTP libraries beyond stdlib.
- Target Go version: 1.22+.

---

## 9. Out of scope (v1)

- **Spoolman websockets.** CLI is one-shot; live updates belong in the UI.
- **Settings / custom fields / labels API.** `setting.*`, `field.*`, label-printing endpoints. Add in v2 if requested.
- **Migration from any other inventory format.** Initial population is done through the CLI (UAT). No `migrate` subcommand.
- **Interop with other local trackers.** Spoolman is the sole source of truth.
- **Auth.** Spoolman doesn't have it; we don't either. If Spoolman adds auth upstream, we'll follow.
- **Multi-server federation.** One server per invocation.
- **Non-filament categories.** Nozzles, solvents, tools — out of scope for this skill.
- **Rewriting SpoolmanDB.** We only read it.

---

## 10. Suggested build order (for execution phase)

1. **Scaffold Go project** + cobra skeleton + `spoolctl env / health / info`. Confirms config resolution (flag → env → file → default) and network end-to-end.
2. **Read-only API client:** `vendor list/get`, `filament list/get`, `spool list/get`. Stable JSON output. Fixture-backed tests against recorded responses.
3. **Write API:** `vendor add/edit/rm`, `filament add/edit/rm`, `spool add/edit/rm/use/measure`. Each command has a `--json` contract test.
4. **SpoolmanDB fetch + cache** under `~/.cache/spoolctl/`; `spoolctl db {filaments,materials,lookup,refresh}`.
5. **Validator:** `spoolctl db validate` with hard / material / enum passes. Golden-file tests against `testdata/spoolmandb-snapshot/`.
6. **Cross-compile** + `scripts/` population.
7. **SKILL.md authoring.** Use the `skill-creator` (or `skill-writer`) skill to generate the frontmatter and validate against the Anthropic Agent Skill spec. Add recipes.
8. **UAT.** Operator re-enters real inventory via `spoolctl vendor add` → `filament add --from-db …` → `spool add …`. Anything that feels awkward becomes a v1.1 ticket.
9. **Ship.** Packaged skill folder + README.

---

## 11. Open questions (flag during execution, not blocking the plan)

- **Which Spoolman version to pin against?** The API schema may evolve. We should capture the `info.version` we tested against and warn if the server reports a newer one.
- **Do we want `spoolctl completion`?** Cheap with cobra; nice-to-have. Default yes.
- **Caching strategy for `spoolctl spool list` in tight loops.** v1: no cache, trust the server. If latency becomes a problem in Claude's multi-call flows, revisit.
- **Colour handling for multi-colour / silk / glow filaments.** SpoolmanDB has `color_hexes`, `glow`, `translucent`, `finish`. `filament add --from-db` should faithfully forward these; worth a golden-file test that round-trips a multi-colour record.
