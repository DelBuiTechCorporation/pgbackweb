-- +goose Up
-- +goose StatementBegin
ALTER TABLE backups ADD COLUMN max_part_size_mb INT;
ALTER TABLE backups ADD COLUMN compression_level SMALLINT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE backups DROP COLUMN max_part_size_mb;
ALTER TABLE backups DROP COLUMN compression_level;
-- +goose StatementEnd
