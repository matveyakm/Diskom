#!/bin/sh
DB_FILE="${1:-employee_directory.db}"
sqlite3 "$DB_FILE" <<-EOSQL
PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS employee_directory (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT NOT NULL,
    description    TEXT,
    status         INTEGER DEFAULT 1,
    basis          INTEGER,
    prefix         INTEGER,
    id_transaction TEXT
);

INSERT INTO employee_directory (name, description, status, basis, prefix, id_transaction)
VALUES ('Administrator', 'System administrator', 1, 0, 0, '');
EOSQL

echo "Done: $DB_FILE"
