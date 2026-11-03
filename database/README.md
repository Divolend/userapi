# UserAPI schema and migrations

`schema/current.sql` is the actual PostgreSQL `pg_dump --schema-only --no-owner --no-privileges -n public --exclude-table=public.schema_migrations` export from TEST. `schema/baseline.sql` is the original schema before lab 8. Both contain DDL, never user data or passwords.

## Stage 1 manual demonstration

Migration 000002 adds optional email (varchar(254), NOT NULL DEFAULT ''). The app reads/writes it using explicit column lists and SQL parameters. Existing rows receive an empty email. POST/PATCH validate email syntax. The PostgreSQL integration test checks create/read/update, rejection of invalid email and persistence after rejection.

Test the feature on TEST, export current.sql, then apply migration 2 manually on STAGE before deploying the dev commit. Keep this step separate from the later automatic migration demonstration. The original pre-lab-8 app uses SELECT * and cannot read the expanded row until the new app is deployed; perform the manual change immediately before delivery.

## Tools

SQL security: Semgrep 1.183.0 with five versioned project rules in ci/sql-security.yml. All SQL files are scanned. Findings or analyzer errors fail CI. Rules forbid grants to PUBLIC, superuser role changes, COPY PROGRAM, SQL password literals and SECURITY DEFINER. This is a focused textual security policy, not a complete PostgreSQL semantic analyzer or proof against all SQL injection. Gosec separately checks application Go code. SQLCheck targets SQL anti-patterns; SQLFluff targets syntax/style. They do not replace these security rules.

Migrations: golang-migrate 4.20.1, official linux-amd64 archive SHA256 b1e13121d4178b7804a9fb29dca62d57b732390f7e5ae33f2cd65e9b192e1bd4. Chosen because the project already uses numbered up/down SQL, PostgreSQL is supported, and no Java/XML changelog is needed. Liquibase offers schema diff/changelogs but adds a JVM and a different change format. Flyway uses versioned SQL and supports validation; golang-migrate matches the project's original technical choice.

`ci/dbctl.py` operates on the selected host's database. TEST/STAGE use local Docker containers; PROD uses the existing native PostgreSQL database rest_api_db. No database passwords are printed or committed. For an existing version-1 schema, automatic baseline registration is allowed only after exact normalized comparison with baseline.sql. Dirty state and automatic downgrade are rejected. `--adopt-exported` is only for manually applying and recording the TEST setup, after equality with current.sql; deployment pipelines must not use it.

Schema comparison excludes the migration bookkeeping table, comments, session SET/config statements, ownership and ACLs, and pg_dump restrict tokens. All application tables, columns, types, defaults, constraints, sequences, indexes and relationships remain in comparison. This compares structure, not database roles/privileges or user data. Before PROD migration take a custom-format full database backup, preserve it outside Git with mode 0600, and verify the archive and a restore into a separate temporary database.

Official references:
- https://docs.semgrep.dev/writing-rules/generic-pattern-matching
- https://docs.semgrep.dev/writing-rules/rule-syntax
- https://github.com/jarulraj/sqlcheck
- https://docs.sqlfluff.com/en/stable/
- https://github.com/golang-migrate/migrate
- https://github.com/golang-migrate/migrate/releases/tag/v4.20.1
- https://www.postgresql.org/docs/16/app-pgdump.html
