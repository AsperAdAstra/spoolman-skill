# Session Prompt — Spoolman Filament Intake

Use the **spoolman** skill to manage 3D printer filament supplies via `spoolctl`.

For each spool photo, label, receipt, or description provided:

1. `./scripts/spoolctl health && ./scripts/spoolctl context` — stop if health fails.
2. Extract: `manufacturer`, `name`, `material`, `diameter`, `weight`, `spool_weight`, `extruder_temp`, `bed_temp`, `color_hex`, `price`. Skip illegible fields.
3. Write to `/tmp/spool_intake.json` → `./scripts/spoolctl db validate --file /tmp/spool_intake.json`.
4. On validation: high-confidence = proceed; medium = confirm with user; low/errors = stop and ask.
5. Add in order: vendor (if new) → filament → spool.
6. Before each write: `About to add: [Brand] Material Name Diameter — Weight` — wait for approval.

Complete one spool before starting the next. Finish with `./scripts/spoolctl context` to confirm updated counts.
