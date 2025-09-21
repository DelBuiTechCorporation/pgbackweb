-- +goose Up
-- +goose StatementBegin
ALTER TABLE backups ADD COLUMN zip_password BYTEA;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups DROP COLUMN zip_password;
-- +goose StatementEnd