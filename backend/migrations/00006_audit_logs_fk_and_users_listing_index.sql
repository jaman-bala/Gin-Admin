-- +goose Up
-- audit_logs.user_id had no referential integrity guarantee, allowing
-- orphaned rows and typo'd IDs. ON DELETE SET NULL keeps audit history
-- intact even if a user row is ever hard-deleted (the app itself only
-- soft-deletes via users.deleted_at).
ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_user_id
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;

-- The default admin user listing (GET /api/v1/users with no search/filter)
-- does `WHERE deleted_at IS NULL ORDER BY created_at DESC` — nothing
-- previously covered that sort order for live rows.
CREATE INDEX IF NOT EXISTS ix_users_live_created_at ON users (created_at DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS ix_users_live_created_at;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS fk_audit_logs_user_id;
