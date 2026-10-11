#!/bin/bash
# build-compose.sh — собирает docker-compose.yaml из фрагментов.
#
# Новый микросервис за 3 шага:
#   1. Положи его compose-блок в source/<type>/ms_<name>/system_description/compose_part.yaml
#      (verbatim, как в docker-compose.yaml: комментарий-заголовок + postgres_/api_ блоки,
#      без завершающей пустой строки).
#   2. Добавь строку "<type>/ms_<name>" в compose.order в нужное место
#      (без compose.order сборка тоже работает — автообнаружение, но порядок будет другой).
#   3. Volumes для нового сервиса допиши руками в docker-compose.base.yaml (автоматизация volumes — фаза 2).
#   4. Запусти ./build-compose.sh и проверь diff.
#
# Использование: ./build-compose.sh [--check]
#   --check: только сверить генерацию с текущим docker-compose.yaml, ничего не писать (для CI).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
BASE="$ROOT/docker-compose.base.yaml"
MANIFEST="$ROOT/compose.order"
TARGET="$ROOT/docker-compose.yaml"
FRAG="system_description/compose_part.yaml"

CHECK=false
if [ "${1:-}" = "--check" ]; then
  CHECK=true
elif [ $# -gt 0 ]; then
  echo "Usage: $0 [--check]" >&2
  exit 2
fi

# Список "<type>/ms_<name>" в порядке сборки.
list_services() {
  if [ -f "$MANIFEST" ]; then
    grep -v '^[[:space:]]*#' "$MANIFEST" | grep -v '^[[:space:]]*$' || true
  else
    echo "WARNING: $MANIFEST not found — auto-discovery, order may differ" >&2
    for t in directory document ledger report; do
      for d in "$ROOT"/source/$t/ms_*/; do
        [ -f "$d$FRAG" ] || continue
        echo "$t/$(basename "$d")"
      done
    done
  fi
}

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

# Голова base до маркера (маркеры в выход не попадают).
awk '/# compose:services:begin/ {exit} {print}' "$BASE" > "$TMP"

COUNT=0
while IFS= read -r line || [ -n "$line" ]; do
  svc="$(printf '%s' "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
  case "$svc" in ""|\#*) continue ;; esac
  frag="$ROOT/source/$svc/$FRAG"
  if [ ! -f "$frag" ]; then
    echo "ERROR: fragment not found: $frag" >&2
    exit 1
  fi
  cat "$frag" >> "$TMP"
  echo "" >> "$TMP"
  COUNT=$((COUNT + 1))
done < <(list_services)

if [ "$COUNT" -eq 0 ]; then
  echo "ERROR: no service fragments found, refusing to write empty compose" >&2
  exit 1
fi

# Хвост base после маркера.
awk 'seen{print} /# compose:services:end/ {seen=1}' "$BASE" >> "$TMP"

# Исходник заканчивается без перевода строки — воспроизводим байт-в-байт.
# (Холостой truncate невозможен: файл заведомо непуст.)
if [ -z "$(tail -c 1 "$TMP")" ]; then
  truncate -s -1 "$TMP"
fi

if $CHECK; then
  if diff -q "$TARGET" "$TMP" > /dev/null; then
    echo "OK: $TARGET matches generated output ($COUNT fragments)"
  else
    echo "MISMATCH: $TARGET differs from generated output:" >&2
    diff "$TARGET" "$TMP" | head -n 40 >&2
    exit 1
  fi
else
  cp "$TMP" "$TARGET"
  echo "Wrote $TARGET ($COUNT fragments)"
fi

docker compose config --quiet && echo "compose config: valid"
