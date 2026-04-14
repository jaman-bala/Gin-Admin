-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE users ALTER COLUMN photo TYPE TEXT;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE users ALTER COLUMN photo TYPE VARCHAR(255);
