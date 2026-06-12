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
	id_basis             bigint,
	status               smallint default 1,
	id_user              bigint,
	prefix               smallint,
    id_employee          bigint NOT NULL,
    id_organization      bigint NOT NULL,
    id_department        bigint NOT NULL,
    id_job_title         bigint NOT NULL,
    justification        text,
	id_transaction       text,
	UNIQUE (id, id_basis)
	);

  COMMENT ON TABLE  ${DB_TABLENAME}                      IS 'Employee dismissal document';
  COMMENT ON COLUMN ${DB_TABLENAME}.id                   IS 'Document ID';
  COMMENT ON COLUMN ${DB_TABLENAME}.ddate                IS 'Document date';
  COMMENT ON COLUMN ${DB_TABLENAME}.dnumber              IS 'Event number';
  COMMENT ON COLUMN ${DB_TABLENAME}.note                 IS 'Note';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_basis             IS 'Basis ID';
  COMMENT ON COLUMN ${DB_TABLENAME}.status               IS 'Document status (0 - deleted, 1 - saved, 2 - completed)';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_user              IS 'System user';
  COMMENT ON COLUMN ${DB_TABLENAME}.prefix               IS 'Prefix of DB';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_employee          IS 'Employee dismissal Id';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_organization      IS 'The organization from which the employee is being dismissed';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_department        IS 'The department from which the employee is being dismissed';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_job_title         IS 'The position from which the employee is being dismissed';
  COMMENT ON COLUMN ${DB_TABLENAME}.justification        IS 'Justification for dismissal';
  COMMENT ON COLUMN ${DB_TABLENAME}.id_transaction       IS 'Transaction Id';

EOSQL
