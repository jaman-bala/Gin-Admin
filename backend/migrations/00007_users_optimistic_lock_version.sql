-- +goose Up
-- Patch/PatchSelf did read-then-write with no concurrency guard: two admins
-- editing the same user at once silently lost one writer's changes. version
-- is bumped on every UPDATE and checked in the WHERE clause (optimistic
-- locking) — a write against a stale version now fails instead of
-- overwriting a concurrent change.
ALTER TABLE users ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE users DROP COLUMN version;
