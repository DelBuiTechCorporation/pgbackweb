-- name: BackupsServiceCreateBackup :one
INSERT INTO backups (
  database_id, destination_id, is_local, name, cron_expression, time_zone,
  is_active, dest_dir, retention_days, opt_data_only, opt_schema_only,
  opt_clean, opt_if_exists, opt_create, opt_no_comments,
  max_part_size_mb, compression_level
)
VALUES (
  @database_id, @destination_id, @is_local, @name, @cron_expression, @time_zone,
  @is_active, @dest_dir, @retention_days, @opt_data_only, @opt_schema_only,
  @opt_clean, @opt_if_exists, @opt_create, @opt_no_comments,
  sqlc.narg('max_part_size_mb'), sqlc.narg('compression_level')
)
RETURNING *;
