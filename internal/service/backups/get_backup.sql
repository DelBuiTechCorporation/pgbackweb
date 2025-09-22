-- name: BackupsServiceGetBackup :one
SELECT
  id, database_id, destination_id, name, cron_expression, time_zone, is_active,
  dest_dir, retention_days, opt_data_only, opt_schema_only, opt_clean,
  opt_if_exists, opt_create, opt_no_comments, created_at, updated_at,
  is_local, all_databases,
  (
    CASE WHEN zip_password IS NOT NULL
    THEN pgp_sym_decrypt(zip_password, @encryption_key)
    ELSE ''
    END
  ) AS decrypted_zip_password
FROM backups
WHERE id = @id;
