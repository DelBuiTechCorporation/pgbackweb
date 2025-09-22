-- name: BackupsServiceCreateBackup :one
INSERT INTO backups (
  database_id, destination_id, is_local, name, cron_expression, time_zone,
  is_active, dest_dir, retention_days, all_databases, opt_data_only, opt_schema_only,
  opt_clean, opt_if_exists, opt_create, opt_no_comments, zip_password
)
VALUES (
  @database_id, @destination_id, @is_local, @name, @cron_expression, @time_zone,
  @is_active, @dest_dir, @retention_days, @all_databases, @opt_data_only, @opt_schema_only,
  @opt_clean, @opt_if_exists, @opt_create, @opt_no_comments,
  CASE
    WHEN NULLIF(sqlc.arg('zip_password')::TEXT, '') IS NULL THEN NULL
    ELSE pgp_sym_encrypt(sqlc.arg('zip_password')::TEXT, sqlc.arg('encryption_key')::TEXT)
  END
)
RETURNING *;
