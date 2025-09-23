-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS provider TEXT;
ALTER TABLE destinations ADD COLUMN IF NOT EXISTS force_path_style BOOLEAN DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations DROP COLUMN IF EXISTS force_path_style;
ALTER TABLE destinations DROP COLUMN IF EXISTS provider;
-- +goose StatementEnd