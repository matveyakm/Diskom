#!/bin/bash
psql -v ON_ERROR_STOP=1 --username "postgres" --dbname "postgres" <<-EOSQL
  CREATE DATABASE ${DB_NAME};
  CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';
  GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
EOSQL

psql -v ON_ERROR_STOP=1 --username "postgres" --dbname "${DB_NAME}" <<-EOSQL
    GRANT USAGE, CREATE ON SCHEMA public TO ${DB_USER};
EOSQL

psql -v ON_ERROR_STOP=1 --username "${DB_USER}" --dbname "${DB_NAME}" <<-EOSQL
  CREATE TABLE public.${DB_TABLENAME} (
    id                   bigint PRIMARY KEY,
    ddate                timestamp,
	dnumber              text,
	note                 text,
	id_basis             smallint,
	status               smallint default 1,
	id_user              bigint,
	prefix               smallint,
	id_counterparty      bigint,
    first_name           text,
    last_name            text,
    middle_name          text,
    first_genitive       text,
    last_genitive        text,
    middle_genitive      text,
    first_accusative     text,
    last_accusative      text,
    middle_accusative    text,
    id_transaction       text,
    UNIQUE (id, id_basis)
	);

  COMMENT ON TABLE  ${DB_TABLENAME}                      IS 'Create counterparty document';
  COMMENT ON COLUMN ${DB_TABLENAME}.id                   IS 'Document ID';
  COMMENT ON COLUMN ${DB_TABLENAME}.ddate                IS 'Document date';
  COMMENT ON COLUMN ${DB_TABLENAME}.dnumber              IS 'Event number';
  COMMENT ON COLUMN ${DB_TABLENAME}.note                 IS 'Note';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_basis             IS 'Basis Id';
  COMMENT ON COLUMN ${DB_TABLENAME}.status               IS 'Document status (0 - deleted, 1 - saved, 2 - completed)';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_user              IS 'System user';
  COMMENT ON COLUMN ${DB_TABLENAME}.prefix               IS 'Prefix of DB';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_employee          IS 'Counterparty ID';
  COMMENT ON COLUMN ${DB_TABLENAME}.first_name           IS 'Name';
  COMMENT ON COLUMN ${DB_TABLENAME}.last_name            IS 'Surname';
  COMMENT ON COLUMN ${DB_TABLENAME}.middle_name          IS 'Patronymic';
  COMMENT ON COLUMN ${DB_TABLENAME}.first_genitive       IS 'Name in the genitive case';
  COMMENT ON COLUMN ${DB_TABLENAME}.last_genitive        IS 'Surname in the genitive case';
  COMMENT ON COLUMN ${DB_TABLENAME}.middle_genitive      IS 'Patronymic in the genitive case';
  COMMENT ON COLUMN ${DB_TABLENAME}.first_accusative     IS 'Name in the accusative case';
  COMMENT ON COLUMN ${DB_TABLENAME}.last_accusative      IS 'Surname in the accusative case';
  COMMENT ON COLUMN ${DB_TABLENAME}.middle_accusative    IS 'Patronymic in the accusative case';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_transaction       IS 'Transaction ID';

EOSQL
