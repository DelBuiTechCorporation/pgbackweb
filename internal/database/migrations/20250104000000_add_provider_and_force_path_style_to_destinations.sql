-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'aws';
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS force_path_style BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations DROP COLUMN IF EXISTS force_path_style;
ALTER TABLE destinations DROP COLUMN IF EXISTS provider;
-- +goose StatementEnd