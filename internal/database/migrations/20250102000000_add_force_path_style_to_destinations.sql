-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations ADD COLUMN force_path_style BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations DROP COLUMN force_path_style;
-- +goose StatementEnd