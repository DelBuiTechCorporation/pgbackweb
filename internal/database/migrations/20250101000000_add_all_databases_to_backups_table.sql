-- +goose Up
-- +goose StatementBegin
ALTER TABLE backups ADD COLUMN all_databases BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups DROP COLUMN all_databases;
-- +goose StatementEnd