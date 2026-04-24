# Session Prompt — Spoolman Filament Inventory

Use the **spoolman** skill to manage 3D printer filament supplies via `spoolctl`.

Always start the session with:
```
./scripts/spoolctl health && ./scripts/spoolctl context
```
Stop if health fails. Use the `context` output to answer questions about current stock — never invent IDs or counts.

---

## What the user may ask

### "What do I have?" / stock queries
Answer directly from `context` output or drill down:
```
./scripts/spoolctl spool list
./scripts/spoolctl filament list --material PLA
./scripts/spoolctl spool get <id>
```

### "I have X grams left" / "I used Y grams" — adjusting amounts
```
# Consumed during a print (reduces remaining)
./scripts/spoolctl spool use <id> --weight <grams>

# Scale reading — gross weight of spool+filament (sets remaining absolutely)
./scripts/spoolctl spool measure <id> --weight <gross-grams>

# Direct correction
./scripts/spoolctl spool edit <id> --set remaining_weight=<grams>
```
Always confirm which spool before writing. Show current `remaining_weight` first so the user can verify the change makes sense.

### Adding new spools — from photo, label, or description
For each item:
1. Extract: `manufacturer`, `name`, `material`, `diameter`, `weight`, `spool_weight`, `extruder_temp`, `bed_temp`, `color_hex`, `price`. Skip illegible fields — do not guess.
2. Write to `/tmp/spool_intake.json` → `./scripts/spoolctl db validate --file /tmp/spool_intake.json`
3. On validation: high-confidence = proceed; medium = confirm with user; low/errors = stop and ask.
4. **Enrich from SpoolmanDB** — when a match is found (`suggested_db_id` present), look up the full record and use it to fill any gaps the user didn't provide:
   ```
   ./scripts/spoolctl db lookup <suggested_db_id>
   ```
   Fields to pull in if missing from the source: `density`, `extruder_temp`, `bed_temp`, `spool_weight`, `color_hex`, `article_number`, `external_id`. User-supplied values always win over DB values — only fill gaps, never overwrite.
5. Add in order: vendor (if new) → filament type (use `--from-db <id>` when confidence is high, manual flags otherwise) → spool.
6. Before each write: `About to add: [Brand] Material Name Diameter — Weight` — wait for approval.
7. Complete one spool before starting the next.

### Location / notes
```
./scripts/spoolctl spool edit <id> --set location="Dry box 1"
./scripts/spoolctl spool edit <id> --set comment="opened 2026-04-24"
```

### Archiving finished spools
```
./scripts/spoolctl spool edit <id> --set archived=true
```

---

After all changes, run `./scripts/spoolctl context` and show the updated `COUNTS` and `LOW_SPOOLS` lines.
