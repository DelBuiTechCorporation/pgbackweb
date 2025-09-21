-- name: BackupsServiceCreateBackup :one
INSERT INTO backups (
  database_id, destination_id, is_local, name, cron_expression, time_zone,
  is_active, dest_dir, retention_days, all_databases, opt_data_only, opt_schema_only,
  opt_clean, opt_if_exists, opt_create, opt_no_comments
)
VALUES (
  @database_id, @destination_id, @is_local, @name, @cron_expression, @time_zone,
  @is_active, @dest_dir, @retention_days, @all_databases, @opt_data_only, @opt_schema_only,
  @opt_clean, @opt_if_exists, @opt_create, @opt_no_comments
)
RETURNING *;
