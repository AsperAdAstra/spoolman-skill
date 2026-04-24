# Spoolman — Agentic Skill + CLI

This repo ships a **Claude Code Agentic Skill** (`SKILL.md`) for managing [Spoolman](https://github.com/Donkie/Spoolman) filament inventory, powered by a purpose-built CLI called **`spoolctl`**.

- **The skill** (`SKILL.md`) is what Claude Code loads. It orchestrates command sequencing, confirmation gates, and LLM-facing workflows.
- **The CLI** (`spoolctl`) is what does the work. It owns all data operations — reads, writes, validation, and SpoolmanDB lookups — and is the sole interface to the Spoolman REST API.

Install the skill → Claude talks to Spoolman. The CLI must be present for the skill to function.

`spoolctl` also integrates with [SpoolmanDB](https://donkie.github.io/SpoolmanDB/) — a community database of 6 000+ filament profiles — for validation and auto-fill when adding new filaments.

Tested against **Spoolman 0.23.1**.

> **Building from source** uses [Task](https://taskfile.dev) (`brew install go-task`). Run `task --list` from the repo root to see all available tasks.

---

## Table of contents

1. [Installation](#installation)
2. [Configuration](#configuration)
3. [Global flags](#global-flags)
4. [Commands](#commands)
   - [env](#env)
   - [health](#health)
   - [info](#info)
   - [context](#context)
   - [vendor](#vendor)
   - [filament](#filament)
   - [spool](#spool)
   - [db](#db)
   - [completion](#completion)
5. [SpoolmanDB integration](#spoolmandb-integration)
6. [Validation report format](#validation-report-format)
7. [Error format](#error-format)
8. [Workflows](#workflows)
9. [Build from source](#build-from-source)

---

## Installation

Pre-built static binaries are in `scripts/` at the **repo root**. Zero runtime dependencies.

| Platform | Binary |
|---|---|
| macOS (Apple Silicon) | `scripts/spoolctl` |
| macOS (Intel) | `scripts/spoolctl-darwin-amd64` |
| Linux x86-64 | `scripts/spoolctl-linux-amd64` |
| Windows x86-64 | `scripts/spoolctl-windows-amd64.exe` |

Copy the appropriate binary onto your `PATH` or invoke it directly from the repo root:

```bash
# macOS / Linux
cp scripts/spoolctl /usr/local/bin/spoolctl
chmod +x /usr/local/bin/spoolctl

# or run in-place from repo root
./scripts/spoolctl health
```

---

## Configuration

`spoolctl` resolves the server URL in this order (first match wins):

| Priority | Source | How to set |
|---|---|---|
| 1 | `--server` flag | `spoolctl --server http://spoolman.lan:7912 health` |
| 2 | `SPOOLMAN_URL` env var | `export SPOOLMAN_URL=http://spoolman.lan:7912` |
| 3 | Config file | `~/.config/spoolctl/config.toml` |
| 4 | Built-in default | `http://localhost:7912` → resolved to `/api/v1` |

**Path handling:** if the URL you provide has no path (just a host and optional port), `/api/v1` is appended automatically. If the URL already contains a path (e.g. `http://host/custom/api`), it is used as-is.

```
http://spoolman.lan:7912           → http://spoolman.lan:7912/api/v1   (auto-append)
http://spoolman.lan:7912/api/v1   → http://spoolman.lan:7912/api/v1   (unchanged)
http://spoolman.lan/proxy/api/v1  → http://spoolman.lan/proxy/api/v1  (unchanged)
```

### Config file

`~/.config/spoolctl/config.toml`:

```toml
server = "http://spoolman.lan:7912"
```

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `SPOOLMAN_URL` | — | Server URL (see path handling above) |
| `SPOOLMAN_TIMEOUT` | `10s` | HTTP request timeout. Accepts Go duration syntax: `5s`, `30s`, `2m`. |
| `SPOOLMAN_INSECURE` | `0` | Set to `1` to skip TLS certificate verification. Useful for self-signed LAN certs. |
| `SPOOLMAN_CA_CERT` | — | Path to a PEM-encoded custom CA bundle. Use when your Spoolman server uses a private CA. |

### Verify your config

Always check where `spoolctl` is pointing before running write commands:

```bash
spoolctl env
```

```
server:      http://spoolman.lan:7912/api/v1
source:      env
timeout:     10s
insecure:    false
ca_cert:     (none)
config_file: /Users/you/.config/spoolctl/config.toml (not found)
```

`source` tells you which resolution step won: `flag`, `env`, `config-file`, or `default`.

---

## Global flags

These flags apply to every command:

| Flag | Short | Description |
|---|---|---|
| `--server URL` | | Override server URL for this invocation only |
| `--timeout DURATION` | | Override request timeout (e.g. `30s`) |
| `--verbose` | `-v` | Print provenance info to stderr (cache vs network, version warnings) |
| `--quiet` | `-q` | Suppress informational output; only print data or errors |

---

## Commands

### env

Print the resolved configuration. No network call is made.

```bash
spoolctl env
```

Output:
```
server:      http://spoolman.lan:7912/api/v1
source:      env
timeout:     10s
insecure:    false
ca_cert:     (none)
config_file: /Users/you/.config/spoolctl/config.toml (not found)
```

---

### health

Check that the Spoolman server is reachable and healthy. Exits non-zero if unhealthy.

```bash
spoolctl health
```

```
status: healthy
```

---

### info

Print server metadata: version, database type, directories.

```bash
spoolctl info
```

```json
{
  "version": "0.23.1",
  "debug_mode": false,
  "automatic_backups": true,
  "data_dir": "/home/app/.local/share/spoolman",
  "logs_dir": "/home/app/.local/share/spoolman",
  "backups_dir": "/home/app/.local/share/spoolman/backups",
  "db_type": "sqlite",
  "git_commit": "eafbc64",
  "build_date": "2026-02-03T16:49:40Z"
}
```

If the server version differs from the version `spoolctl` was tested against (0.23.1), a warning is printed to stdout unless `--quiet` is set.

---

### context

Emit a compact, token-efficient inventory snapshot designed for LLM grounding. Use this as the first call before any planning or mutation sequence.

```bash
spoolctl context
```

```
CTXv1 server=http://spoolman.lan:7912/api/v1 health=ok ts=2026-04-24T10:48:04Z
COUNTS vendors=3 filaments=12 spools=27 low=4 archived=3
MATERIALS PLA=18 PETG=6 TPU=2 ABS=1
LOW_SPOOLS id=5:82g:PLA-Basic-Black|id=12:95g:PETG-Gray|id=19:120g:ASA-White
RECENT_USE id=5:-220g:PLA-Basic-Black:2026-04-23|id=8:-42g:PETG-Gray:2026-04-22
DB_STATE source=upstream cache_dir=/Users/you/.cache/spoolctl
```

**Lines:**
- `CTXv1` — format version tag; backward-compatible.
- `COUNTS` — totals for active (non-archived) spools, filaments, vendors. `low` = spools with < 150 g remaining.
- `MATERIALS` — filament count by material, sorted by count descending.
- `LOW_SPOOLS` — up to 10 low spools, sorted by weight ascending. Format: `id=N:Wg:Label`. Truncated as `...+N` if more than 10.
- `RECENT_USE` — up to 5 most recently used spools. Format: `id=N:-Wg:Label:YYYY-MM-DD`.
- `DB_STATE` — SpoolmanDB cache location.

---

### vendor

Manage filament vendors (manufacturers).

#### vendor list

```bash
spoolctl vendor list [--name <substring>]
```

| Flag | Description |
|---|---|
| `--name` | Filter vendors whose name contains this string |

Returns a JSON array of vendor objects.

#### vendor get

```bash
spoolctl vendor get <id>
```

Returns a single vendor as JSON.

#### vendor add

```bash
spoolctl vendor add --name <name> [flags]
```

| Flag | Description |
|---|---|
| `--name` | Vendor name **(required)** |
| `--comment` | Free text comment |
| `--spool-weight` | Default empty spool weight in grams (used as fallback when filament has none) |
| `--external-id` | SpoolmanDB manufacturer ID |
| `--extra key=value` | Extra custom fields; may be repeated |

Returns the created vendor as JSON.

#### vendor edit

```bash
spoolctl vendor edit <id> --set key=value [--set key=value ...]
```

Editable keys: `name`, `comment`, `empty_spool_weight`, `external_id`.

```bash
spoolctl vendor edit 1 --set name="Bambu Lab" --set comment="Primary vendor"
```

Returns the updated vendor as JSON.

#### vendor rm

```bash
spoolctl vendor rm <id>
```

Deletes the vendor. Prints confirmation unless `--quiet`.

---

### filament

Manage filament types (the template for spools — material, diameter, temperatures, etc.).

#### filament list

```bash
spoolctl filament list [--vendor <id>] [--material <material>]
```

| Flag | Description |
|---|---|
| `--vendor` | Filter by vendor ID |
| `--material` | Filter by material name (e.g. `PLA`, `PETG`, `TPU`) |

Returns a JSON array of filament objects.

#### filament get

```bash
spoolctl filament get <id>
```

#### filament add

Two modes: manual entry or auto-fill from SpoolmanDB.

**Manual — required flags:** `--density` and `--diameter`.

```bash
spoolctl filament add \
  --vendor-id 1 \
  --name "Basic Black" \
  --material PLA \
  --density 1.24 \
  --diameter 1.75 \
  --weight 1000 \
  --spool-weight 250 \
  --extruder-temp 220 \
  --bed-temp 60 \
  --color-hex 000000 \
  --price 19.99
```

**From SpoolmanDB — auto-fills all fields:**

```bash
spoolctl filament add --from-db bambulab_pla_black_1000_175_n --vendor-id 1
```

All fields come from the SpoolmanDB record. You can still override individual fields with additional flags (e.g. `--color-hex`, `--price`, `--name`).

| Flag | Description |
|---|---|
| `--from-db <id>` | SpoolmanDB record ID; auto-fills all fields |
| `--vendor-id` | Vendor ID |
| `--name` | Filament name (e.g. "Basic Black") |
| `--material` | Material type (PLA, PETG, TPU, ABS, …) |
| `--density` | Density in g/cm³ **(required without --from-db)** |
| `--diameter` | Diameter in mm **(required without --from-db)** |
| `--weight` | Net filament weight on a full spool, in grams |
| `--spool-weight` | Empty spool (tare) weight in grams |
| `--extruder-temp` | Extruder temperature in °C |
| `--bed-temp` | Bed temperature in °C |
| `--color-hex` | Color as 6-character hex (e.g. `FF0000`) |
| `--price` | Price in server-configured currency |
| `--comment` | Free text comment |
| `--article-number` | EAN, QR code, or vendor part number |
| `--external-id` | SpoolmanDB record ID (set automatically with `--from-db`) |

#### filament edit

```bash
spoolctl filament edit <id> --set key=value [--set key=value ...]
```

Editable keys: `name`, `material`, `density`, `diameter`, `weight`, `spool_weight`, `settings_extruder_temp`, `settings_bed_temp`, `color_hex`, `comment`, `price`, `vendor_id`.

```bash
spoolctl filament edit 3 --set settings_extruder_temp=215 --set color_hex=1A1A1A
```

#### filament rm

```bash
spoolctl filament rm <id>
```

---

### spool

Manage individual physical spools.

#### spool list

```bash
spoolctl spool list [--filament <id>] [--archived]
```

| Flag | Description |
|---|---|
| `--filament` | Filter by filament type ID |
| `--archived` | Include archived spools (default: active only) |

#### spool get

```bash
spoolctl spool get <id>
```

Returns full spool JSON including the embedded filament object, `remaining_weight`, `used_weight`, `used_length`, `location`, etc.

#### spool add

```bash
spoolctl spool add --filament-id <id> [flags]
```

| Flag | Description |
|---|---|
| `--filament-id` | Filament type ID **(required)** |
| `--initial-weight` | Net filament weight at creation in grams (e.g. `1000`) |
| `--spool-weight` | Empty spool (tare) weight in grams. Overrides the filament type's default. |
| `--remaining-weight` | Set remaining weight directly (alternative to `--initial-weight`) |
| `--used-weight` | Set used weight directly (alternative to `--initial-weight`) |
| `--price` | Price paid for this spool |
| `--location` | Where the spool is stored (e.g. `"Shelf A"`, `"Dry box 2"`) |
| `--lot-nr` | Manufacturer lot / batch number |
| `--comment` | Free text comment |
| `--archived` | Create as archived immediately |

**Weight fields:** Spoolman tracks `used_weight` and computes `remaining_weight = initial_weight - used_weight`. Supply either `--initial-weight` (most common: brand-new spool) or one of the alternative weight flags for a partially-used spool.

#### spool edit

```bash
spoolctl spool edit <id> --set key=value [--set key=value ...]
```

Editable keys: `filament_id`, `location`, `comment`, `lot_nr`, `price`, `initial_weight`, `spool_weight`, `remaining_weight`, `used_weight`, `archived`.

```bash
# Move to a different location
spoolctl spool edit 4 --set location="Dry box 1"

# Archive a spool
spoolctl spool edit 4 --set archived=true

# Correct used weight
spoolctl spool edit 4 --set used_weight=123.5
```

#### spool use

Record filament consumed by a print job. The server subtracts from the spool's remaining weight.

```bash
spoolctl spool use <id> --weight <grams> [--length <mm>] [--ref <label>]
```

| Flag | Description |
|---|---|
| `--weight` | Grams consumed |
| `--length` | Millimeters consumed (can be used together with `--weight`) |
| `--ref` | Print job label for your records (informational; stored in `--ref` but not sent to server) |

At least one of `--weight` or `--length` is required.

```bash
# Record 42 g used
spoolctl spool use 3 --weight 42

# Record by length (e.g. from slicer output)
spoolctl spool use 3 --length 14200

# Both weight and length
spoolctl spool use 3 --weight 42 --length 14200
```

Returns the updated spool as JSON.

#### spool measure

Set the remaining filament weight by reading a physical scale. You weigh the spool (filament + plastic spool together) and pass the **gross** weight; the server subtracts the tare (empty spool weight) automatically.

```bash
spoolctl spool measure <id> --weight <gross-grams>
```

| Flag | Description |
|---|---|
| `--weight` | Current gross weight of the spool in grams **(required)** |

```bash
# Spool + filament reads 850 g on scale
spoolctl spool measure 3 --weight 850
```

Returns the updated spool as JSON.

#### spool rm

```bash
spoolctl spool rm <id>
```

---

### db

Access the [SpoolmanDB](https://donkie.github.io/SpoolmanDB/) community filament database. Two data sources are available per command:

| `--source` | Description |
|---|---|
| `upstream` (default) | Reads from local cache; fetches if cache is absent |
| `spoolman` | Queries your Spoolman server's built-in copy via `/external/filament` |

#### db filaments

```bash
spoolctl db filaments [--manufacturer <name>] [--material <type>] [--diameter <mm>] [--source upstream|spoolman]
```

| Flag | Description |
|---|---|
| `--manufacturer` | Filter by manufacturer name (case-insensitive substring) |
| `--material` | Filter by material (e.g. `PLA`, `PETG`) |
| `--diameter` | Filter by diameter in mm (e.g. `1.75`, `2.85`) |
| `--source` | `upstream` (default) or `spoolman` |

```bash
# Find all Bambu Lab PLA filaments at 1.75 mm
spoolctl db filaments --manufacturer "Bambu Lab" --material PLA --diameter 1.75

# Query your Spoolman server's built-in copy instead
spoolctl db filaments --material PETG --source spoolman
```

#### db materials

```bash
spoolctl db materials [--source upstream|spoolman]
```

Lists all known materials with default density, extruder temp, and bed temp.

#### db lookup

```bash
spoolctl db lookup <spoolmandb-id>
```

Returns the full SpoolmanDB record for the given ID. IDs use underscore format: `bambulab_pla_black_1000_175_n`.

```bash
spoolctl db lookup bambulab_pla_black_1000_175_n
```

```json
{
  "id": "bambulab_pla_black_1000_175_n",
  "manufacturer": "Bambu Lab",
  "name": "Black",
  "material": "PLA",
  "density": 1.24,
  "weight": 1000,
  "spool_weight": 250,
  "diameter": 1.75,
  "color_hex": "000000",
  "extruder_temp": 220,
  "bed_temp": 60,
  "translucent": false,
  "glow": false
}
```

#### db validate

Validate a filament spec file against SpoolmanDB. Accepts TOML or JSON.

```bash
spoolctl db validate --file <path> [--strict]
```

| Flag | Description |
|---|---|
| `--file` | Path to the spec file (`.toml` or `.json`) **(required)** |
| `--strict` | Treat warnings as errors (exit non-zero) |

See [Validation report format](#validation-report-format) and [SpoolmanDB integration](#spoolmandb-integration) for details.

#### db diff

Compare what your Spoolman server's built-in SpoolmanDB copy contains vs. the current upstream. Useful for checking whether your Spoolman installation needs an update.

```bash
spoolctl db diff
```

```json
{
  "server_count": 6800,
  "upstream_count": 6957,
  "upstream_only": 157,
  "server_only": 0,
  "diffs": [
    { "id": "bambulab_pla_newcolor_1000_175_n", "status": "upstream_only" },
    ...
  ]
}
```

#### db refresh

Force a re-fetch of the upstream SpoolmanDB cache, regardless of whether cache files exist. Always prints provenance.

```bash
spoolctl db refresh
```

```
# db source=network fetching SpoolmanDB...
refreshed: 6957 filaments, 33 materials
```

Cache is written to `~/.cache/spoolctl/filaments.json` and `~/.cache/spoolctl/materials.json`.

---

### completion

Generate shell completion scripts.

```bash
# Bash — add to ~/.bash_completion or /etc/bash_completion.d/
spoolctl completion bash >> ~/.bash_completion

# Zsh — add to a directory in $fpath
spoolctl completion zsh > "${fpath[1]}/_spoolctl"

# Fish
spoolctl completion fish > ~/.config/fish/completions/spoolctl.fish
```

---

## SpoolmanDB integration

SpoolmanDB is a community-maintained database of filament profiles: densities, temperatures, spool weights, color codes, and more. `spoolctl` uses it in two ways:

1. **Auto-fill (`filament add --from-db`)** — look up a record by ID and create a filament with all fields pre-populated.
2. **Validation (`db validate`)** — cross-check a spec file against known good values.

### SpoolmanDB ID format

IDs follow the pattern `{manufacturer}_{material}_{name}_{weight}_{diameter*100}_{n}`, all lowercase, spaces replaced with nothing (no dashes between the parts):

```
bambulab_pla_black_1000_175_n
polymaker_petg_polysonicblack_1000_175_n
esun_pla+_coldwhite_1000_175_n
```

The easiest way to find an ID:

```bash
spoolctl db filaments --manufacturer "Bambu Lab" --material PLA
```

Then copy the `id` field and pass it to `lookup` or `filament add --from-db`.

### Cache

Downloaded SpoolmanDB data is cached at `~/.cache/spoolctl/`. Commands read from cache when it exists; use `spoolctl db refresh` to force a re-fetch. With `--verbose`, every db command prints `# db source=cache age=Xm` or `# db source=network fetching SpoolmanDB...` to stderr.

---

## Validation report format

`spoolctl db validate` runs three passes and returns a JSON report:

```json
{
  "input": "spec.toml",
  "status": "warn",
  "matches": [
    { "field": "diameter", "value": 1.75, "db_value": 1.75 },
    { "field": "density",  "value": 1.24, "db_value": 1.24 }
  ],
  "warnings": [
    {
      "field": "extruder_temp",
      "value": 240,
      "expected_range": [195, 225],
      "material": "PLA"
    }
  ],
  "errors": [],
  "suggested_db_id": "bambulab_pla_black_1000_175_n",
  "match_confidence": "high",
  "auto_corrections": [
    { "field": "material", "from": "PLA +", "to": "PLA+" }
  ],
  "requires_confirmation": false
}
```

| Field | Description |
|---|---|
| `status` | `ok` / `warn` / `error` |
| `matches` | Fields that matched SpoolmanDB values |
| `warnings` | Advisory issues (out-of-range temps/density; unknown material) |
| `errors` | Hard failures (invalid enum values; large field mismatches) |
| `suggested_db_id` | Best-matching SpoolmanDB record ID |
| `match_confidence` | `high` (exact ID match) / `medium` (single candidate) / `low` (ambiguous) |
| `auto_corrections` | Safe normalizations applied before validation (spacing, punctuation) |
| `requires_confirmation` | `true` if human review is needed before writing |

**The three validation passes:**

1. **Hard match** — if `external_id` is set (or derivable), compare field-by-field against that SpoolmanDB record. Mismatches become errors.
2. **Material sanity** — density must be within ±10% of the material default; extruder/bed temps within ±15 °C.
3. **Enum sanity** — `finish` must be `matte` or `glossy`; `pattern` must be `marble` or `sparkle`; `spool_type` must be `plastic`, `cardboard`, or `metal`.

**Spec file fields** (TOML or JSON):

```toml
# spec.toml — all fields optional; use what you have
external_id    = "bambulab_pla_black_1000_175_n"   # triggers hard match
manufacturer   = "Bambu Lab"
name           = "Basic Black"
material       = "PLA"
density        = 1.24
diameter       = 1.75
weight         = 1000
spool_weight   = 250
extruder_temp  = 220
bed_temp       = 60
finish         = "matte"     # matte | glossy
pattern        = ""          # marble | sparkle
spool_type     = "plastic"   # plastic | cardboard | metal
```

---

## Error format

All errors are written to **stderr** as JSON with a non-zero exit code:

```json
{"error": "HTTP 404: Vendor not found", "status": 1}
```

For HTTP 4xx/5xx from Spoolman:

```json
{"error": "HTTP 422: Unprocessable Entity (detail: ...)", "status": 1}
```

This makes it safe to pipe stdout to `jq` without mixing error output into the data stream.

---

## Workflows

### Initial setup

```bash
# 1. Set server URL
export SPOOLMAN_URL=http://spoolman.lan:7912

# 2. Verify connectivity
spoolctl health
spoolctl env

# 3. Seed SpoolmanDB cache
spoolctl db refresh

# 4. Get a snapshot of current state
spoolctl context
```

### Add a new vendor + filament + spool

```bash
# Add vendor
spoolctl vendor add --name "Bambu Lab"
# → note the returned id, e.g. 1

# Search SpoolmanDB for the filament
spoolctl db filaments --manufacturer "Bambu Lab" --material PLA --diameter 1.75

# Look up the exact record
spoolctl db lookup bambulab_pla_black_1000_175_n

# Add filament auto-filled from SpoolmanDB
spoolctl filament add --from-db bambulab_pla_black_1000_175_n --vendor-id 1
# → note the returned id, e.g. 2

# Add the physical spool
spoolctl spool add --filament-id 2 --initial-weight 1000 --price 19.99 --location "Shelf A"
```

### Record usage after a print

```bash
# By weight (most accurate — weigh spool before and after)
spoolctl spool use 3 --weight 42

# By length (from slicer)
spoolctl spool use 3 --length 14200

# By scale measurement (gross weight of spool+filament)
spoolctl spool measure 3 --weight 850
```

### Validate a filament before adding

```bash
# Write a spec from label information
cat > /tmp/spec.toml << 'EOF'
manufacturer  = "Some Brand"
material      = "PLA +"
density       = 1.24
diameter      = 1.75
extruder_temp = 220
bed_temp      = 60
EOF

# Validate (auto-corrects "PLA +" → "PLA+" and checks temperatures)
spoolctl db validate --file /tmp/spec.toml

# Strict mode: any warning is a failure
spoolctl db validate --file /tmp/spec.toml --strict
```

### Archive a finished spool

```bash
spoolctl spool edit 7 --set archived=true
```

### List low-stock spools

```bash
# context shows LOW_SPOOLS line — quick overview
spoolctl context

# Full JSON detail
spoolctl spool list | jq '[.[] | select(.remaining_weight != null and .remaining_weight < 150)]'
```

### Check for stale SpoolmanDB

```bash
spoolctl db diff
# upstream_only > 0 means your Spoolman server hasn't been updated to include newer filament profiles
```

---

## Build from source

Requires **Go 1.22+** and [Task](https://taskfile.dev) (`brew install go-task`).

All commands run from the **repo root** where `Taskfile.yml` lives.

```bash
# Install Go dependencies
task deps

# Build for current platform → scripts/spoolctl
task build

# Cross-compile for all four platforms → scripts/
task cross

# Run unit tests
task test

# Remove built binaries
task clean
```

Binaries are written to `scripts/` at the repo root.

**Task targets:**

| Task | Description |
|---|---|
| `deps` | Run `go mod tidy` |
| `build` | Build for current OS/arch → `scripts/spoolctl` |
| `cross` | Build darwin-arm64, darwin-amd64, linux-amd64, windows-amd64 |
| `pack` | Cross-compile then produce one ZIP per platform in `out/` |
| `test` | Run `go test ./...` with verbose output |
| `clean` | Remove all binaries from `scripts/` |
| `distclean` | `clean` + remove `out/` |
| `size` | Show sizes of built binaries |

---

## Distribution

`task pack` cross-compiles all platforms and creates four ZIPs in `out/` at the repo root. `out/` is git-ignored.

```bash
task pack
# Produces:
#   out/spoolman-darwin-arm64.zip
#   out/spoolman-darwin-amd64.zip
#   out/spoolman-linux-amd64.zip
#   out/spoolman-windows-amd64.zip
```

**ZIP contents** (each archive has the same internal layout, with the platform-appropriate binary renamed to `spoolctl`):

```
spoolman/
  SKILL.md
  README.md
  scripts/
    spoolctl          ← spoolctl.exe on Windows
```

**Installing a ZIP as a Claude Code skill:**

```bash
# macOS / Linux
unzip out/spoolman-darwin-arm64.zip -d ~/.claude/skills/

# Verify
ls ~/.claude/skills/spoolman/
# SKILL.md  README.md  scripts/
```

The `spoolman/` directory name inside the ZIP matches the `name:` field in `SKILL.md` — Claude Code uses the directory name for skill discovery.

**Repo layout:**

```
filament-tracker/
  .gitignore                  ignores out/
  Taskfile.yml                build / pack tasks
  PLAN.md
  SKILL.md                    Agent skill definition and recipes
  scripts/                    Pre-built binaries (output of make build/cross)
  out/                        Distribution ZIPs (git-ignored, output of make pack)
    spoolctl
    spoolctl-darwin-amd64
    spoolctl-linux-amd64
    spoolctl-windows-amd64.exe
  spoolman-cli/               Go source and tests
    README.md                 This file
    src/
      go.mod                  Module: github.com/vibecoder/spoolctl
      cmd/spoolctl/main.go    Entry point
      internal/
        config/               Config resolution (flag→env→file→default)
        api/                  Spoolman REST client (pass-through JSON)
          client.go
          models.go
          vendor.go
          filament.go
          spool.go
          external.go
        db/                   SpoolmanDB fetch, cache, validator
          fetch.go
          spoolmandb.go
          validator.go
        cli/                  Cobra command tree
          root.go
          env.go / health.go / info.go / context.go
          vendor.go / filament.go / spool.go
          db.go / completion.go
    testdata/
      spoolmandb-snapshot/    Pinned filaments.json + materials.json for tests
      api-fixtures/           Recorded Spoolman API responses
```
