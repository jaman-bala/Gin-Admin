-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Allow re-registering a phone that belongs to a soft-deleted account:
-- uniqueness applies to live rows only.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_key;
CREATE UNIQUE INDEX IF NOT EXISTS ux_users_phone_active ON users (phone) WHERE deleted_at IS NULL;

-- Trigram indexes to back ILIKE '%...%' search on the admin user list.
CREATE INDEX IF NOT EXISTS ix_users_first_name_trgm ON users USING gin (first_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS ix_users_last_name_trgm  ON users USING gin (last_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS ix_users_phone_trgm      ON users USING gin (phone gin_trgm_ops);

-- Audit log access patterns: newest-first listing and entity/user filters.
CREATE INDEX IF NOT EXISTS ix_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS ix_audit_logs_entity_id  ON audit_logs (entity_id);
CREATE INDEX IF NOT EXISTS ix_audit_logs_user_id    ON audit_logs (user_id);

-- +goose Down
DROP INDEX IF EXISTS ix_audit_logs_user_id;
DROP INDEX IF EXISTS ix_audit_logs_entity_id;
DROP INDEX IF EXISTS ix_audit_logs_created_at;
DROP INDEX IF EXISTS ix_users_phone_trgm;
DROP INDEX IF EXISTS ix_users_last_name_trgm;
DROP INDEX IF EXISTS ix_users_first_name_trgm;
DROP INDEX IF EXISTS ux_users_phone_active;
ALTER TABLE users ADD CONSTRAINT users_phone_key UNIQUE (phone);
