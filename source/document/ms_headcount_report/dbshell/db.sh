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
    id			bigint PRIMARY KEY,
    ddate			timestamp,
    dnumber			text,
    id_committee			bigint,
    date_from			timestamp,
    date_to			timestamp,
    comment			text,
    id_basis			smallint,
    status			smallint default 1,
    prefix			smallint,
    id_user			bigint,
    id_transaction			text,
  UNIQUE (id, id_basis)
  );

  COMMENT ON TABLE  public.${DB_TABLENAME} IS 'Headcount Report';
  COMMENT ON COLUMN public.${DB_TABLENAME}.id IS 'Id';
  COMMENT ON COLUMN public.${DB_TABLENAME}.ddate IS 'Ddate';
  COMMENT ON COLUMN public.${DB_TABLENAME}.dnumber IS 'Dnumber';
  COMMENT ON COLUMN public.${DB_TABLENAME}.id_committee IS 'Id Committee';
  COMMENT ON COLUMN public.${DB_TABLENAME}.date_from IS 'Date From';
  COMMENT ON COLUMN public.${DB_TABLENAME}.date_to IS 'Date To';
  COMMENT ON COLUMN public.${DB_TABLENAME}.comment IS 'Comment';
  COMMENT ON COLUMN public.${DB_TABLENAME}.id_basis IS 'Id Basis';
  COMMENT ON COLUMN public.${DB_TABLENAME}.status IS 'Status';
  COMMENT ON COLUMN public.${DB_TABLENAME}.prefix IS 'Prefix';
  COMMENT ON COLUMN public.${DB_TABLENAME}.id_user IS 'Id User';
  COMMENT ON COLUMN public.${DB_TABLENAME}.id_transaction IS 'Id Transaction';
EOSQL
