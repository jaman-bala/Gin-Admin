-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS uuid AS $$
DECLARE
  unix_ts_ms bytea;
  uuid_bytes bytea;
BEGIN
  unix_ts_ms = substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) from 3);
  uuid_bytes = uuid_send(gen_random_uuid());
  uuid_bytes = overlay(uuid_bytes placing unix_ts_ms from 1 for 6);
  uuid_bytes = set_byte(uuid_bytes, 6, (b'0111' || get_byte(uuid_bytes, 6)::bit(4))::bit(8)::int);
  return encode(uuid_bytes, 'hex')::uuid;
END
$$ LANGUAGE PLPGSQL VOLATILE;
-- +goose StatementEnd

ALTER TABLE users ALTER COLUMN id SET DEFAULT uuid_generate_v7();

-- +goose Down
ALTER TABLE users ALTER COLUMN id SET DEFAULT uuid_generate_v4();

-- +goose StatementBegin
DROP FUNCTION IF EXISTS uuid_generate_v7();
-- +goose StatementEnd