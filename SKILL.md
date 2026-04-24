---
name: spoolman
description: Manage Spoolman 3D-printer filament inventory using the spoolctl CLI. Use when the user mentions Spoolman, filament spools, spool inventory, add a spool, add filament, record print usage, check filament stock, validate against SpoolmanDB, 3D printer material tracking, filament database, low spool, or any filament-related inventory operation. Requires spoolctl binary in scripts/.
---

# Spoolman Skill

Manage [Spoolman](https://github.com/Donkie/Spoolman) filament inventory via the `spoolctl` CLI. All data operations (read, write, validate, normalize) are owned by the binary; this skill orchestrates command sequencing, image extraction, and confirmation gates.

## Quick start

```bash
# Verify connectivity and ground context before any operation
./scripts/spoolctl health && ./scripts/spoolctl context
```

## When NOT to use

Non-filament inventory (nozzles, solvents, tools). This skill is filament-only.

## Instructions

### 1. Ground context before mutations

Always start with:

```bash
./scripts/spoolctl context
```

This emits a compact, token-efficient snapshot (`CTXv1` format) covering vendor/filament/spool counts, low spools, recent usage, and DB cache state. Use it to plan the command sequence — never invent IDs.

### 2. Locate the server

Resolution order (first match wins):
1. `--server URL` flag
2. `SPOOLMAN_URL` env var
3. `~/.config/spoolctl/config.toml` → `server = "..."`
4. `http://localhost:7912/api/v1` (built-in default)

If the user mentions a remote host (e.g. "my Spoolman at spoolman.lan"), export `SPOOLMAN_URL` for the session or pass `--server` per command. URLs must include protocol and host. If no path is provided, `/api/v1` is appended automatically.

Run `./scripts/spoolctl env` once to confirm the target before any mutation.

If `health` fails, stop and report the error — never retry mutations.

### 3. Standard operations

All write commands emit the server's response JSON to stdout and errors to stderr as `{"error":"…","status":N}` with a non-zero exit code. Surface errors verbatim; do not retry writes.

#### List & query

```bash
./scripts/spoolctl spool list                              # all active spools
./scripts/spoolctl spool list --archived                   # include archived
./scripts/spoolctl spool get <id>                          # full spool detail
./scripts/spoolctl filament list --material PLA            # filter filaments
./scripts/spoolctl vendor list
```

#### Add vendor → filament → spool (full flow)

```bash
# Step 1: create vendor if needed
./scripts/spoolctl vendor add --name "Bambu Lab"

# Step 2: find the filament in SpoolmanDB
./scripts/spoolctl db filaments --manufacturer "Bambu Lab" --material PLA --diameter 1.75

# Step 3: look up the exact record
./scripts/spoolctl db lookup bambulab_pla_black_1000_175_n

# Step 4: create filament (auto-filled from SpoolmanDB)
./scripts/spoolctl filament add --from-db bambulab_pla_black_1000_175_n --vendor-id <vendor-id>

# Step 5: add the physical spool
./scripts/spoolctl spool add --filament-id <filament-id> --initial-weight 1000 --price 25
```

#### Record print usage

```bash
./scripts/spoolctl spool use <id> --weight 42              # by grams consumed
./scripts/spoolctl spool use <id> --length 14200           # by mm from slicer
./scripts/spoolctl spool measure <id> --weight 850         # gross scale reading (auto-subtracts tare)
```

#### Edit & archive

```bash
./scripts/spoolctl spool edit <id> --set location="Dry box 1"
./scripts/spoolctl spool edit <id> --set archived=true
./scripts/spoolctl filament edit <id> --set settings_extruder_temp=215
```

### 4. Image-derived data workflow

When the user provides a spool label photo:

1. Extract candidate fields from the image: `manufacturer`, `name`, `material`, `diameter`, `extruder_temp`, `bed_temp`, `spool_weight`, `color_hex`.
2. Write extracted fields to a temp file (e.g. `/tmp/extracted.json`).
3. Validate:
   ```bash
   ./scripts/spoolctl db validate --file /tmp/extracted.json
   ```
4. Interpret the report:
   - `requires_confirmation: false` **and** `match_confidence: high` → apply `auto_corrections`, then proceed with `filament add --from-db <suggested_db_id>`.
   - `requires_confirmation: true` **or** conflicts in `material`/`diameter` → **stop and ask the user** to confirm conflicting fields before any write.

Safe auto-corrections (apply without asking): case/spacing normalization (`PLA +` → `PLA+`), unit normalization (`1.75mm` → `1.75`).
Never auto-correct across material families or diameters.

### 5. SpoolmanDB operations

```bash
./scripts/spoolctl db filaments --manufacturer "Bambu Lab" --material PLA
./scripts/spoolctl db materials
./scripts/spoolctl db lookup bambulab_pla_black_1000_175_n
./scripts/spoolctl db validate --file spec.toml [--strict]
./scripts/spoolctl db diff                                 # server DB vs upstream
./scripts/spoolctl db refresh                              # force cache re-fetch
```

SpoolmanDB is cached at `~/.cache/spoolctl/`. Use `--verbose` to see whether data came from cache or network.

## Invariants

- **Always** call `./scripts/spoolctl context` before planning mutations.
- **Never** invent IDs — always `list` or `get` first.
- `spoolctl spool use` reduces weight; `spoolctl spool measure` sets absolute remaining from a scale reading.
- If `health` fails, stop — don't proceed with mutations.
- For image-derived inputs, always run `db validate` before any write.

## Debugging

```bash
./scripts/spoolctl env          # show resolved server URL, source, timeout
./scripts/spoolctl health       # connectivity check
./scripts/spoolctl info         # server version, db type
```

Add `--verbose` to any command to see cache provenance and version warnings on stderr.

---

## API field reference

### Vendor

| Field | Type | Notes |
|---|---|---|
| `id` | int | Read-only |
| `name` | string | Required |
| `comment` | string? | Free text |
| `empty_spool_weight` | float? | Grams |
| `external_id` | string? | SpoolmanDB manufacturer ID |

### Filament

| Field | Type | Notes |
|---|---|---|
| `id` | int | Read-only |
| `vendor_id` | int? | Reference to vendor |
| `name` | string? | e.g. "PLA Basic Black" |
| `material` | string? | e.g. "PLA", "PETG", "TPU" |
| `density` | float | **Required** g/cm³ |
| `diameter` | float | **Required** mm |
| `weight` | float? | Net spool weight g |
| `spool_weight` | float? | Empty spool weight g |
| `settings_extruder_temp` | int? | °C |
| `settings_bed_temp` | int? | °C |
| `color_hex` | string? | 6-char hex |
| `multi_color_hexes` | string? | Comma-separated hexes |
| `multi_color_direction` | string? | `coaxial` or `longitudinal` |
| `price` | float? | In server currency |
| `article_number` | string? | EAN/QR |
| `external_id` | string? | SpoolmanDB record ID |

### Spool

| Field | Type | Notes |
|---|---|---|
| `id` | int | Read-only |
| `filament_id` | int | **Required** |
| `initial_weight` | float? | Net filament weight g at creation |
| `spool_weight` | float? | Tare weight g (overrides filament default) |
| `remaining_weight` | float? | Computed unless set explicitly |
| `used_weight` | float | Accumulated usage g |
| `price` | float? | In server currency |
| `location` | string? | e.g. "Shelf A" |
| `lot_nr` | string? | Batch number |
| `archived` | bool | Default false |

### `spool use` parameters
- `--weight <g>`: reduce by this many grams
- `--length <mm>`: reduce by this many millimeters
- At least one required; both can be provided together

### `spool measure` parameters
- `--weight <g>`: current **gross** weight (filament + spool); server subtracts tare automatically

## Validation report format

`db validate` returns:

```json
{
  "input": "spec.toml",
  "status": "warn",
  "matches": [{ "field": "diameter", "value": 1.75, "db_value": 1.75 }],
  "warnings": [{ "field": "extruder_temp", "value": 240, "expected_range": [195, 225], "material": "PLA" }],
  "errors": [],
  "suggested_db_id": "bambulab_pla_black_1000_175_n",
  "match_confidence": "high",
  "auto_corrections": [{ "field": "material", "from": "PLA +", "to": "PLA+" }],
  "requires_confirmation": false
}
```

`status`: `ok` / `warn` / `error`. `match_confidence`: `high` / `medium` / `low`.

## Tested against Spoolman 0.23.1
