---
name: dc-stack-ops
description: Use when building, running, restarting, or debugging the Disciplinary Committee stack — podman/podman-compose, pre-compose.sh, build-compose.sh, pre_traefik.sh, Traefik routing, connecting to the Postgres containers, scratch-DB verification, or the headless-Chrome UI harness.
---

# Disciplinary Committee — stack & ops

## Runtime

- Rootless **podman** + `podman-compose` (not Docker). `~/.bashrc` sets
  `DOCKER_HOST=unix:///run/user/1000/podman/podman.sock`, so plain `docker`
  targets podman.
- Docker/BuildKit does not work on this host (h2c 404); use `podman build`.
- All services share the bridge network `disciplinary-committee_vnet`
  (`172.25.0.0/24`). Traefik terminates TLS and routes by `Host(...) && PathPrefix(...)`.

## Hostnames / ports

- App: `https://discom.spbgu.localhost:8443/` (also
  `discom.spbgu.90.188.89.63.nip.io`). Traefik dashboard
  `https://dashboard.docker.localhost:8443/`; whoami `whoami.docker.localhost`;
  jaeger `http://localhost:16686`.
- `frontend-js` (protofront nginx) is the live front. Postgres containers listen
  on `172.25.0.x:5434`; APIs on `172.25.0.x+1:8443`.

## Bring the whole stack up (destructive — re-inits all DB volumes)

```bash
./pre-compose.sh
```
It: `podman-compose stop` + `down --volumes`; per module `go mod init/tidy`
(+ `refresh_dockerfile.sh`); `build-db.sh`; `build-compose.sh`;
`podman-compose up -d --build`.

- A schema/seed change only takes effect through `down --volumes` (init scripts
  run on an empty data dir). To apply just metadata changes you can re-init only
  the metadata volume, but `pre-compose.sh` is the standard path.
- `docker-compose.yaml` is generated (gitignored) — never edit by hand;
  `build-compose.sh` rebuilds it from `system_description/compose_part.yaml`
  fragments. Verify with `./build-compose.sh --check`.
- Rebuild a single image, e.g. front: `podman build -t
  disciplinary-committee_frontend-js -f protofront/Dockerfile protofront`, then
  run the **full** `podman-compose up -d` (a scoped `up` of one service has torn
  the stack down before).

## TLS certs

`./pre_traefik.sh` regenerates `certs/` (`server.crt/key` CN=discom.spbgu.localhost,
`local.crt/key`, nip.io SAN cert) and prints the traefik dashboard basic-auth
hash. `dynamic/tls.yaml` lists them.

## Connecting to Postgres

Containers do **not** accept socket/127.0.0.1; use the vnet IP.

```bash
PW='Yy5U%]%6,ep>t:WhfWo'   # POSTGRES_PASSWORD, also in .env / parameters.env
podman exec -e PGPASSWORD="$PW" postgres_system_object_directory \
  psql -h 172.25.0.99 -p 5434 -U postgres -d psadmin
```
Metadata DB is `psadmin`, view `system_object_directory`, app user
`mssystem_object_directory`. Scratch DB `dbg` lives in the same container.

## Scratch-DB verification (schema changes)

```bash
# regenerate and extract the main psql block from the generated db.sh
bash source/directory/ms_system_object_directory/build-db.sh
# then, in the metadata postgres container: DROP/CREATE DATABASE dbg,
# sed 's/\${DB_TABLENAME}/system_object_directory/; s/\${DB_USER}/mssystem_object_directory/'
# and psql -v ON_ERROR_STOP=1 -f. Check for "error"/"fatal", EXIT=0.
```
Then build `row_to_json` rows for all entities and diff against the legacy
`01_system_object.sql` JSON (details in `dc-system-object-schema`).

## UI harness / checks

- API smoke: `curl -k 'https://discom.spbgu.localhost:8443/api/system_object_directory?limit=100'`
  returns `{data:[...], pagination:{...}}`; a down API service returns
  `503 {"error":"api service unavailable"}` from nginx.
- Headless Chrome over CDP (port 9222) using `/tmp/opencode/run.sh` and
  `/tmp/opencode/menutest.mjs` (imports `/usr/share/nodejs/ws/index.js`) —
  confirms the 26-item menu and console errors.

## Files

- `pre-compose.sh`, `build-compose.sh`, `pre_traefik.sh`, `refresh_dockerfile.sh`.
- `docker-compose.base.yaml` — static infra (traefik, whoami, jaeger, frontend-js)
  with `# compose:services:begin/end` and `# compose:volumes:begin/end` markers.
- `Dockerfiles/` — per-service Dockerfiles generated from `Dockerfile.work`.
- `cert`/`dynamic/` — TLS config; `certs/` mostly gitignored.
