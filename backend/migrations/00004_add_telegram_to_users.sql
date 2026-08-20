-- +goose Up
ALTER TABLE users ADD COLUMN telegram TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN telegram;