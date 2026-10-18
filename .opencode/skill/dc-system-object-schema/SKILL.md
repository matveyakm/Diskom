---
name: dc-system-object-schema
description: Use when editing system_description SQL, dbshell/db.sh.base, seed data, or the normalized metadata schema (system_object, workflow_process, privilege, related_document, filter_set/map_set/fk_set) of source/directory/ms_system_object_directory in the Disciplinary Committee project. Covers the 01->02 legacy-JSON conversion, build-db.sh, the 03 late pass, and scratch-DB parity verification.
---

# Disciplinary Committee — system_object metadata schema

`ms_system_object_directory` stores the metadata that drives the whole UI. Its
runtime script `dbshell/db.sh` is **generated**; edit `dbshell/db.sh.base` and
the per-entity `system_description/*.sql` seeds instead.

## Layout / naming in `system_description/`

- `01_system_object.sql` — legacy format: the whole object as SQL `SELECT`
  literals including raw JSON strings for `table_columns`, `item_fields`, and the
  four workflow actions (`onSaveNew`, `onSaveEdit`, `onAcceptance`, `onDelete`).
  Treat as the **source of truth for parity**.
- `02_system_object_relational.sql` — normalized seed (one file per entity).
  `build-db.sh` prefers `02_*_relational.sql` over `01_*` when both exist.
- `02_XX_group_*_relational.sql` — group seeds live in the metadata service
  itself (`02_01_group_directories`, `02_02_group_documents`,
  `02_03_group_reports`, `02_04_group_ledgers`, `02_05_group_about`).
- `03_*related*.sql` — **late pass** files, executed by `build-db.sh` after
  every entity exists, used for cross-object references (FK by `object_link.path`).
- `compose_part.yaml` — verbatim compose fragment (postgres + api service), fed
  to `build-compose.sh`.

## Tables (in `db.sh.base`)

Core: `system_object` (id, name, object_link→object_link.id, object_type,
object_status, id_system_group self-FK, level, output_order, object_icon,
prefix, prefix_number), `object_link` (path = DB tablename), `object_icon`,
`object_type`, `object_status`.

Forms: `form_type`, `object_fields`, `object_forms`, `tab_forms` (name, title,
tag, object_form, output_order, visible), `field_editor_type` (manual/dropdown/form),
`fields_tab` (tab_form, object_field, label, visible, required, editor_type,
editor_ref→system_object.id, `"default"`).

Workflow (per `object_form`, keyed by `events_type` name + `step`):
`events_type` (0 onSaveNew, 1 onSaveEdit, 2 onAcceptance, 3 onDelete),
`method_type` (0 post, 1 put, 2 get, 3 delete), `workflow_process`
(name, object_form, events_type, method_type, step, end_point, filter_set,
hide_in_preview, conditions JSON text, fields JSON text, set_to_server→map_set,
set_to_app→map_set, fk_set→fk_set).

Sets: `filter_set` + `filter_set_item` (filter_type, output_order),
`map_set` + `map_set_item` (target_field, server_field, application_field,
value_type, send_value, output_order), `fk_set` + `fk_set_item`
(context_field, response_field, output_order). One set row per step that needs it.

Object-level (NOT on `workflow_process`): `privilege` (system_object→system_object.id,
privilege_type→privilege_types.id, output_order, UNIQUE(system_object, privilege_type)),
`related_document` (system_object, document→system_object.id, output_order,
UNIQUE(system_object, document)).

Lookups seeded: `filter_type` (0 id, 1 id_document, 2 id_employee,
3 id_counterparty, 4 id_department, 5 id_job_title, 6 id_organization,
7 id_regulations_clauses), `privilege_types` (0 open_list, 1 open_item, 2 create,
3 edit, 4 delete, 5 accept, 6 create_related_documents).

## The view `public.${DB_TABLENAME}` (system_object_directory)

Read-only view reconstructing the legacy catalog row for the Go model
(`model/model.go` → `Object`). Notable columns:
- `ref` from `object_link.path`, `icon` from `object_icon.path`.
- `table_title` = first `List%` form name else first form else object name;
  `item_title` = first non-`List%` form name else `''`.
- `table_columns` / `item_fields` = JSON strings built from tab/field tables;
  `item_fields` = `{"fields":[...],"blocks":[{id,title,tag,fields:[...]}]}`.
- `on_save_new`/`on_save_edit`/`on_acceptance`/`on_delete` = per-event JSON
  `{"action":...,"processing":[...]}`, **NULL when the event has no steps**.
  Step keys: `step`, `method`, `end_point`, `filter` (first filter),
  `filters` (all, additive), `hide_in_preview` (only when true), `map`
  (merged set_to_server + set_to_app), `FK`, `fields`, `conditions`. Absent keys
  are stripped via `jsonb_strip_nulls`, so the shape matches `01_*.sql`.
- `privileges` = `{"privileges":[{"<name>":"true|false"},...]}` over all
  `privilege_types` (all false when none).
- `related_documents` = comma-separated related document ids (string) or NULL.
- `map_value(text)→jsonb` helper: JSON if value starts `{`/`[`, numeric if
  numeric-looking, else string (preserves CONCAT expressions as JSON).

## Gotchas

- **Postgres**: within one statement, the primary command cannot see rows
  inserted by a data-modifying CTE. Cross-object FKs must be inserted via
  referencing CTEs or a `03_*related*.sql` late file — never with a trailing
  `UPDATE ... WHERE id = (SELECT ...)` over newly inserted rows.
- **`ANALYZE` is mandatory** after seeding. The tables are tiny (below
  autoanalyze); without stats the planner assumes ~1000 rows and the view scan
  can take 15s+. `db.sh.base` ANALYZE list must include any new table.
- Keep `--SYSTEM_OBJECT_INSERTS_START` in `db.sh.base`; `build-db.sh` splits the
  file on that marker and injects entity seeds in between.

## Build & verify (scratch DB)

`build-db.sh` assembles `dbshell/db.sh` = base header + all entity seeds
(group → directory → document → ledger → report) + `03_*related*.sql` late
files + footer (function, view, ANALYZE).

```bash
# 1. regenerate db.sh (gitignored)
bash source/directory/ms_system_object_directory/build-db.sh

# 2. extract the main SQL block and load into a throwaway DB "dbg"
#    (see dc-stack-ops for the postgres connection command)
#    ${DB_TABLENAME} -> system_object_directory, ${DB_USER} -> mssystem_object_directory
```

Parity check: for each `01_system_object.sql`, parse its JSON literals and
compare against the rebuilt view row (query `system_object_directory` in `dbg`
as `row_to_json`, keyed by `ref`), normalizing `Step`/`step` casing and dropping
the additive `filters` key. Expect 0 problems across all 21 entities.
