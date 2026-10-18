---
name: dc-frontend
description: Use when editing the Disciplinary Committee UI — the protofront vanilla-JS desktop shell (protofront/views, protofront/core, pipeline/pipelineEngine.js, nginx.conf) or the React/Vite app under frontend/. Covers the system_object_directory API contract and the item-form/workflow pipeline.
---

# Disciplinary Committee — frontend

Two front-ends exist:

- `protofront/` — **the live UI** (container `frontend-js`, nginx). Plain
  `<script>` tags, no bundler; each file defines a global class/function and
  load order in `index.html` matters (core → apiService → pipeline → views →
  window/windowManager → menu → index).
- `frontend/` — newer React + Vite + TypeScript + MUI + `@tanstack/react-query`
  app with Playwright e2e; partially wired, not the live front. Scripts:
  `dev`, `build` (`tsc -b && vite build`), `lint`, `preview`, `test:e2e`.

## protofront architecture

Desktop metaphor: `startMenuApp.js` (start menu), `menuManager.js` +
`menuApiService.js` (build the menu from the metadata API), `windowManager.js` /
`window.js` (windows + taskbar), `views/taskbar.js`, `views/taskButton.js`,
`timerManager.js` (clock).

Data/edit layer:
- `core/httpClient.js` — fetch wrapper (uses `cache: "no-store"`).
- `core/responseCache.js`, `core/refFields.js` — response/ref-field caches.
- `apiService.js` — REST calls per entity (`/api/<tablename>`).
- `core/configParser.js` — `parseActionConfig`, `parseBlob`,
  `parseColumnsConfig`; turns `table_columns`, `item_fields`, and workflow
  action JSON strings into config objects, including `itemBlocks` for tabs.
- `pipeline/pipelineEngine.js` — executes a workflow action: iterates
  `processing` steps using `step.step`/`Step`, `step.method`, `step.end_point`,
  `step.map` (also `set_to_server`/`set_to_app`), `step.fields`, `step.FK`,
  `step.filter`, `step.conditions`, `step.hide_in_preview` (strictly true hides
  the step in the confirm preview).
- `views/itemForm.js` — builds the tabbed item form from `itemBlocks`;
  `views/windowStyles.js` holds `.item-form-tab*` styles; `views/fieldEditors.js`,
  `views/enhancedSelect.js`, `views/tableView.js`, `views/windowFrame.js`.

## API contract

`GET /api/system_object_directory` → `{ data: [...], pagination: { page_size: 20, ... } }`.
Per-object fields come from the metadata view (`model/Object`):
`id, name, description, status, prefix, prefix_number, level, is_group, id_group,
ref, output_order, icon, object_type_name, object_status_name, table_columns,
table_title, item_title, item_fields, on_save_new, on_save_edit, on_acceptance,
on_delete, privileges, related_documents, business_processes, printing_template,
id_transaction`. The `on_*` fields are JSON strings or null; `related_documents`
is a comma-separated id string or null; `privileges` is
`{"privileges":[{"open_list":"true"},...]}`.

The menu fetches with `limit=100` and paginates.

## nginx / caching (protofront/nginx.conf)

- JS served with `expires 1h` + `Cache-Control: must-revalidate`.
- `/index.html` `no-store`.
- `/api/` proxies to Traefik; when the API service is down it returns
  `503 {"error":"api service unavailable"}`.

After changing protofront, rebuild the image and restart the stack (see
`dc-stack-ops`).

## Metadata that drives the UI

Everything the UI renders comes from the `system_object_directory` view
(`dc-system-object-schema`): `table_columns` → list columns, `item_fields` →
item-form fields + blocks/tabs, and the four `on_*` action JSONs → workflows.
Change behavior there rather than hard-coding in the front.
