-- name: ExecutionsServiceGetBackupData :one
SELECT
  backups.is_active as backup_is_active,
  backups.is_local as backup_is_local,
  backups.dest_dir as backup_dest_dir,
  backups.all_databases as backup_all_databases,
  backups.opt_data_only as backup_opt_data_only,
  backups.opt_schema_only as backup_opt_schema_only,
  backups.opt_clean as backup_opt_clean,
  backups.opt_if_exists as backup_opt_if_exists,
  backups.opt_create as backup_opt_create,	
  backups.opt_no_comments as backup_opt_no_comments,
  (
    CASE WHEN backups.zip_password IS NOT NULL
    THEN pgp_sym_decrypt(backups.zip_password, @encryption_key)
    ELSE ''
    END
  ) AS decrypted_backup_zip_password,

  pgp_sym_decrypt(databases.connection_string, @encryption_key) AS decrypted_database_connection_string,
  databases.pg_version as database_pg_version,

  destinations.bucket_name as destination_bucket_name,
  destinations.region as destination_region,
  destinations.endpoint as destination_endpoint,
  destinations.force_path_style as destination_force_path_style,
  (
    CASE WHEN destinations.access_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.access_key, @encryption_key)
    ELSE ''
    END
  ) AS decrypted_destination_access_key,
  (
    CASE WHEN destinations.secret_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.secret_key, @encryption_key)
    ELSE ''
    END
  ) AS decrypted_destination_secret_key
FROM backups
INNER JOIN databases ON backups.database_id = databases.id
LEFT JOIN destinations ON backups.destination_id = destinations.id
WHERE backups.id = @backup_id;
