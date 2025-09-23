-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations
ADD COLUMN provider TEXT NOT NULL DEFAULT 'minio',
ADD COLUMN force_path_style BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations
DROP COLUMN IF EXISTS provider,
DROP COLUMN IF EXISTS force_path_style;
-- +goose StatementEnd