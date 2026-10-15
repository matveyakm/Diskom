#!/bin/bash
set -e

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
SOURCE_DIR="$(cd "$BASE_DIR/../.." && pwd)"
DBSHELL_DIR="$BASE_DIR/dbshell"
TARGET="$DBSHELL_DIR/db.sh"

# Auto-discover system_description seeds.
# New relational files (02_*_relational.sql, converted from old JSON blobs)
# win over legacy 01_system_object.sql when both exist.
# Order: groups, then directory, document, ledger
pick_entity() {
  if [ -f "$1/system_description/02_system_object_relational.sql" ]; then
    echo "$1/system_description/02_system_object_relational.sql"
  elif [ -f "$1/system_description/01_system_object.sql" ]; then
    echo "$1/system_description/01_system_object.sql"
  fi
}
collect_inserts() {
  # Groups from ms_system_object_directory (new set wins as a whole)
  if ls "$BASE_DIR"/system_description/02_*_relational.sql >/dev/null 2>&1; then
    for f in "$BASE_DIR"/system_description/02_*_relational.sql; do
      echo "$f"
    done
  else
    for f in "$BASE_DIR"/system_description/*.sql; do
      [ -f "$f" ] && echo "$f"
    done
  fi

  # Directory entities
  for dir in "$SOURCE_DIR"/directory/ms_*/; do
    pick_entity "$dir"
  done

  # Document entities
  for dir in "$SOURCE_DIR"/document/ms_*/; do
    pick_entity "$dir"
  done

  # Ledger entities
  for dir in "$SOURCE_DIR"/ledger/ms_*/; do
    pick_entity "$dir"
  done

  # Report entities
  for dir in "$SOURCE_DIR"/report/ms_*/; do
    pick_entity "$dir"
  done

}

{
  # Header (up to marker)
  awk '1; /--SYSTEM_OBJECT_INSERTS_START/ { exit }' "$DBSHELL_DIR/db.sh.base"

  # All inserts in order with blank line between
  first=true
  while IFS= read -r src; do
    $first || echo ""
    first=false
    cat "$src"
  done < <(collect_inserts)

  # Footer (from marker onward)
  awk '/--SYSTEM_OBJECT_INSERTS_START/,0' "$DBSHELL_DIR/db.sh.base" | tail -n +2
} > "$TARGET"

chmod +x "$TARGET"
echo "Built: $TARGET ($(wc -l < "$TARGET") lines)"
