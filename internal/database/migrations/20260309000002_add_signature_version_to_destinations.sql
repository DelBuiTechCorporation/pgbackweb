-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations
  ADD COLUMN signature_version TEXT NOT NULL DEFAULT 'v4',
  ADD CONSTRAINT destinations_signature_version_check
  CHECK (signature_version IN ('v2', 'v4'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations
  DROP CONSTRAINT IF EXISTS destinations_signature_version_check,
  DROP COLUMN IF EXISTS signature_version;
-- +goose StatementEnd