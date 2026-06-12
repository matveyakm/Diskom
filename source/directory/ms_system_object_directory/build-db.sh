#!/bin/bash
set -e

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
SOURCE_DIR="$(cd "$BASE_DIR/../.." && pwd)"
DBSHELL_DIR="$BASE_DIR/dbshell"
TARGET="$DBSHELL_DIR/db.sh"

# Auto-discover all system_description/01_system_object.sql files
# Order: groups, then directory, document, ledger
collect_inserts() {
  # Groups from ms_system_object_directory
  for f in "$BASE_DIR"/system_description/*.sql; do
    [ -f "$f" ] && echo "$f"
  done

  # Directory entities
  for dir in "$SOURCE_DIR"/directory/ms_*/; do
    f="$dir/system_description/01_system_object.sql"
    [ -f "$f" ] && echo "$f"
  done

  # Document entities
  for dir in "$SOURCE_DIR"/document/ms_*/; do
    f="$dir/system_description/01_system_object.sql"
    [ -f "$f" ] && echo "$f"
  done

  # Ledger entities
  for dir in "$SOURCE_DIR"/ledger/ms_*/; do
    f="$dir/system_description/01_system_object.sql"
    [ -f "$f" ] && echo "$f"
  done

  # Report entities
  for dir in "$SOURCE_DIR"/report/ms_*/; do
    f="$dir/system_description/01_system_object.sql"
    [ -f "$f" ] && echo "$f"
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
