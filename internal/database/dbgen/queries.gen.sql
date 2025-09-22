-- name: AuthServiceDeleteAllUserSessions :exec
DELETE FROM sessions WHERE user_id = @user_id;
-- name: AuthServiceDeleteOldSessions :exec
DELETE FROM sessions WHERE created_at <= @date_threshold;
-- name: AuthServiceDeleteSession :exec
DELETE FROM sessions WHERE id = @id;
-- name: AuthServiceGetUserByToken :one
SELECT
  users.*,
  sessions.id as session_id
FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE pgp_sym_decrypt(sessions.token, @encryption_key) = @token::TEXT;
-- name: AuthServiceGetUserSessions :many
SELECT * FROM sessions WHERE user_id = @user_id ORDER BY created_at DESC;
-- name: AuthServiceLoginGetUserByEmail :one
SELECT * FROM users WHERE email = @email;

-- name: AuthServiceLoginCreateSession :one
INSERT INTO sessions (
  user_id, token, ip, user_agent
) VALUES (
  @user_id, pgp_sym_encrypt(@token::TEXT, @encryption_key::TEXT), @ip, @user_agent
) RETURNING *, pgp_sym_decrypt(token, @encryption_key::TEXT) AS decrypted_token;
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
-- name: BackupsServiceDeleteBackup :exec
DELETE FROM backups
WHERE id = @id;
-- name: BackupsServiceDuplicateBackup :one
INSERT INTO backups
SELECT (
  backups
  #= hstore('id', uuid_generate_v4()::text)
  #= hstore('name', (backups.name || ' (copy)')::text)
  #= hstore('is_active', false::text)
  #= hstore('created_at', now()::text)
  #= hstore('updated_at', now()::text)
).*
FROM backups
WHERE backups.id = @backup_id
RETURNING *;
-- name: BackupsServiceGetAllBackups :many
SELECT * FROM backups
ORDER BY created_at DESC;
-- name: BackupsServiceGetBackup :one
SELECT * FROM backups
WHERE id = @id;
-- name: BackupsServiceGetBackupsQty :one
SELECT 
  COUNT(*) AS all,
  COALESCE(SUM(CASE WHEN is_active = true THEN 1 ELSE 0 END), 0)::INTEGER AS active,
  COALESCE(SUM(CASE WHEN is_active = false THEN 1 ELSE 0 END), 0)::INTEGER AS inactive
FROM backups;
-- name: BackupsServicePaginateBackupsCount :one
SELECT COUNT(*) FROM backups;

-- name: BackupsServicePaginateBackups :many
SELECT
  backups.*,
  databases.name AS database_name,
  destinations.name AS destination_name
FROM backups
INNER JOIN databases ON backups.database_id = databases.id
LEFT JOIN destinations ON backups.destination_id = destinations.id
ORDER BY backups.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: BackupsServiceGetScheduleAllData :many
SELECT 
  id,
  is_active,
  cron_expression,
  time_zone
FROM backups
ORDER BY created_at DESC;
-- name: BackupsServiceToggleIsActive :one
UPDATE backups
SET is_active = NOT is_active
WHERE id = @backup_id
RETURNING *;
-- name: BackupsServiceUpdateBackup :one
UPDATE backups
SET
  database_id = COALESCE(sqlc.narg('database_id'), database_id),
  destination_id = COALESCE(sqlc.narg('destination_id'), destination_id),
  is_local = COALESCE(sqlc.narg('is_local'), is_local),
  name = COALESCE(sqlc.narg('name'), @name),
  cron_expression = COALESCE(sqlc.narg('cron_expression'), cron_expression),
  time_zone = COALESCE(sqlc.narg('time_zone'), time_zone),
  is_active = COALESCE(sqlc.narg('is_active'), is_active),
  dest_dir = COALESCE(sqlc.narg('dest_dir'), dest_dir),
  retention_days = COALESCE(sqlc.narg('retention_days'), retention_days),
  all_databases = COALESCE(sqlc.narg('all_databases'), all_databases),
  opt_data_only = COALESCE(sqlc.narg('opt_data_only'), opt_data_only),
  opt_schema_only = COALESCE(sqlc.narg('opt_schema_only'), opt_schema_only),
  opt_clean = COALESCE(sqlc.narg('opt_clean'), opt_clean),
  opt_if_exists = COALESCE(sqlc.narg('opt_if_exists'), opt_if_exists),
  opt_create = COALESCE(sqlc.narg('opt_create'), opt_create),
  opt_no_comments = COALESCE(sqlc.narg('opt_no_comments'), opt_no_comments),
  zip_password = CASE
    WHEN sqlc.narg('zip_password')::TEXT IS NULL THEN zip_password
    WHEN sqlc.narg('zip_password')::TEXT = '' THEN NULL
    ELSE pgp_sym_encrypt(sqlc.narg('zip_password')::TEXT, sqlc.arg('encryption_key')::TEXT)
  END
WHERE id = @id
RETURNING *;
-- name: DatabasesServiceCreateDatabase :one
INSERT INTO databases (
  name, connection_string, pg_version
)
VALUES (
  @name, pgp_sym_encrypt(@connection_string, @encryption_key), @pg_version
)
RETURNING *;
-- name: DatabasesServiceDeleteDatabase :exec
DELETE FROM databases
WHERE id = @id;
-- name: DatabasesServiceGetAllDatabases :many
SELECT
  *,
  pgp_sym_decrypt(connection_string, @encryption_key) AS decrypted_connection_string
FROM databases
ORDER BY created_at DESC;
-- name: DatabasesServiceGetDatabase :one
SELECT
  *,
  pgp_sym_decrypt(connection_string, @encryption_key) AS decrypted_connection_string
FROM databases
WHERE id = @id;
-- name: DatabasesServiceGetDatabasesQty :one
SELECT 
  COUNT(*) AS all,
  COALESCE(SUM(CASE WHEN test_ok = true THEN 1 ELSE 0 END), 0)::INTEGER AS healthy,
  COALESCE(SUM(CASE WHEN test_ok = false THEN 1 ELSE 0 END), 0)::INTEGER AS unhealthy
FROM databases;
-- name: DatabasesServicePaginateDatabasesCount :one
SELECT COUNT(*) FROM databases;

-- name: DatabasesServicePaginateDatabases :many
SELECT
  *,
  pgp_sym_decrypt(connection_string, @encryption_key) AS decrypted_connection_string
FROM databases
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: DatabasesServiceSetTestData :exec
UPDATE databases
SET test_ok = @test_ok,
    test_error = @test_error,
    last_test_at = NOW()
WHERE id = @database_id;
-- name: DatabasesServiceUpdateDatabase :one
UPDATE databases
SET
  name = COALESCE(sqlc.narg('name'), name),
  pg_version = COALESCE(sqlc.narg('pg_version'), pg_version),
  connection_string = CASE
    WHEN sqlc.narg('connection_string')::TEXT IS NOT NULL
    THEN pgp_sym_encrypt(
      sqlc.narg('connection_string')::TEXT, sqlc.arg('encryption_key')::TEXT
    )
    ELSE connection_string
  END
WHERE id = @id
RETURNING *;
-- name: DestinationsServiceCreateDestination :one
INSERT INTO destinations (
  name, bucket_name, region, endpoint,
  access_key, secret_key
)
VALUES (
  @name, @bucket_name, @region, @endpoint,
  pgp_sym_encrypt(@access_key, @encryption_key),
  pgp_sym_encrypt(@secret_key, @encryption_key)
)
RETURNING *;
-- name: DestinationsServiceDeleteDestination :exec
DELETE FROM destinations
WHERE id = @id;
-- name: DestinationsServiceGetAllDestinations :many
SELECT
  *,
  pgp_sym_decrypt(access_key, @encryption_key) AS decrypted_access_key,
  pgp_sym_decrypt(secret_key, @encryption_key) AS decrypted_secret_key
FROM destinations
ORDER BY created_at DESC;
-- name: DestinationsServiceGetDestination :one
SELECT
  *,
  pgp_sym_decrypt(access_key, @encryption_key) AS decrypted_access_key,
  pgp_sym_decrypt(secret_key, @encryption_key) AS decrypted_secret_key
FROM destinations
WHERE id = @id;
-- name: DestinationsServiceGetDestinationsQty :one
SELECT 
  COUNT(*) AS all,
  COALESCE(SUM(CASE WHEN test_ok = true THEN 1 ELSE 0 END), 0)::INTEGER AS healthy,
  COALESCE(SUM(CASE WHEN test_ok = false THEN 1 ELSE 0 END), 0)::INTEGER AS unhealthy
FROM destinations;
-- name: DestinationsServicePaginateDestinationsCount :one
SELECT COUNT(*) FROM destinations;

-- name: DestinationsServicePaginateDestinations :many
SELECT
  *,
  pgp_sym_decrypt(access_key, @encryption_key) AS decrypted_access_key,
  pgp_sym_decrypt(secret_key, @encryption_key) AS decrypted_secret_key
FROM destinations
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: DestinationsServiceSetTestData :exec
UPDATE destinations
SET test_ok = @test_ok,
    test_error = @test_error,
    last_test_at = NOW()
WHERE id = @destination_id;
-- name: DestinationsServiceUpdateDestination :one
UPDATE destinations
SET
  name = COALESCE(sqlc.narg('name'), name),
  bucket_name = COALESCE(sqlc.narg('bucket_name'), bucket_name),
  region = COALESCE(sqlc.narg('region'), region),
  endpoint = COALESCE(sqlc.narg('endpoint'), endpoint),
  access_key = CASE
    WHEN sqlc.narg('access_key')::TEXT IS NOT NULL
    THEN pgp_sym_encrypt(sqlc.narg('access_key')::TEXT, sqlc.arg('encryption_key')::TEXT)
    ELSE access_key
  END,
  secret_key = CASE
    WHEN sqlc.narg('secret_key')::TEXT IS NOT NULL
    THEN pgp_sym_encrypt(sqlc.narg('secret_key')::TEXT, sqlc.arg('encryption_key')::TEXT)
    ELSE secret_key
  END
WHERE id = @id
RETURNING *;
-- name: ExecutionsServiceCreateExecution :one
INSERT INTO executions (backup_id, status, message, path)
VALUES (@backup_id, @status, @message, @path)
RETURNING *;
-- name: ExecutionsServiceGetExecution :one
SELECT
  executions.*,
  databases.id AS database_id,
  databases.pg_version AS database_pg_version
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
WHERE executions.id = @id;
-- name: ExecutionsServiceGetExecutionsQty :one
SELECT 
  COUNT(*) AS all,
  COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0)::INTEGER AS running,
  COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0)::INTEGER AS success,
  COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)::INTEGER AS failed,
  COALESCE(SUM(CASE WHEN status = 'deleted' THEN 1 ELSE 0 END), 0)::INTEGER AS deleted
FROM executions;
-- name: ExecutionsServiceGetDownloadLinkOrPathData :one
SELECT
  executions.path AS path,
  backups.is_local AS is_local,
  destinations.bucket_name AS bucket_name,
  destinations.region AS region,
  destinations.endpoint AS endpoint,
  destinations.endpoint as destination_endpoint,
  (
    CASE WHEN destinations.access_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.access_key, sqlc.arg('decryption_key')::TEXT)
    ELSE ''
    END
  ) AS decrypted_access_key,
  (
    CASE WHEN destinations.secret_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.secret_key, sqlc.arg('decryption_key')::TEXT)
    ELSE ''
    END
  ) AS decrypted_secret_key
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE executions.id = @execution_id;
-- name: ExecutionsServiceListBackupExecutions :many
SELECT * FROM executions
WHERE backup_id = @backup_id
ORDER BY started_at DESC;
-- name: ExecutionsServicePaginateExecutionsCount :one
SELECT COUNT(executions.*)
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE
(
  sqlc.narg('backup_id')::UUID IS NULL
  OR
  backups.id = sqlc.narg('backup_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  databases.id = sqlc.narg('database_id')::UUID
)
AND
(
  sqlc.narg('destination_id')::UUID IS NULL
  OR
  destinations.id = sqlc.narg('destination_id')::UUID
);

-- name: ExecutionsServicePaginateExecutions :many
SELECT
  executions.*,
  backups.name AS backup_name,
  databases.name AS database_name,
  databases.pg_version AS database_pg_version,
  destinations.name AS destination_name,
  backups.is_local AS backup_is_local
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE
(
  sqlc.narg('backup_id')::UUID IS NULL
  OR
  backups.id = sqlc.narg('backup_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  databases.id = sqlc.narg('database_id')::UUID
)
AND
(
  sqlc.narg('destination_id')::UUID IS NULL
  OR
  destinations.id = sqlc.narg('destination_id')::UUID
)
ORDER BY executions.started_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: ExecutionsServiceGetBackupData :one
SELECT
  backups.is_active as backup_is_active,
  backups.is_local as backup_is_local,
  backups.dest_dir as backup_dest_dir,
  backups.opt_data_only as backup_opt_data_only,
  backups.opt_schema_only as backup_opt_schema_only,
  backups.opt_clean as backup_opt_clean,
  backups.opt_if_exists as backup_opt_if_exists,
  backups.opt_create as backup_opt_create,
  backups.opt_no_comments as backup_opt_no_comments,

  pgp_sym_decrypt(databases.connection_string, @encryption_key) AS decrypted_database_connection_string,
  databases.pg_version as database_pg_version,

  destinations.bucket_name as destination_bucket_name,
  destinations.region as destination_region,
  destinations.endpoint as destination_endpoint,
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
-- name: ExecutionsServiceGetExecutionForSoftDelete :one
SELECT
  executions.id as execution_id,
  executions.path as execution_path,

  backups.id as backup_id,
  backups.is_local as backup_is_local,

  destinations.bucket_name as destination_bucket_name,
  destinations.region as destination_region,
  destinations.endpoint as destination_endpoint,
  (
    CASE WHEN destinations.access_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.access_key, sqlc.arg('encryption_key')::TEXT)
    ELSE ''
    END
  ) AS decrypted_destination_access_key,
  (
    CASE WHEN destinations.secret_key IS NOT NULL
    THEN pgp_sym_decrypt(destinations.secret_key, sqlc.arg('encryption_key')::TEXT)
    ELSE ''
    END
  ) AS decrypted_destination_secret_key
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE executions.id = @execution_id;

-- name: ExecutionsServiceSoftDeleteExecution :exec
UPDATE executions
SET
  status = 'deleted',
  deleted_at = NOW()
WHERE id = @id;
-- name: ExecutionsServiceGetExpiredExecutions :many
SELECT executions.*
FROM executions
JOIN backups ON executions.backup_id = backups.id
WHERE
  backups.retention_days > 0
  AND executions.status != 'deleted'
  AND executions.finished_at IS NOT NULL
  AND (
    executions.finished_at + (backups.retention_days || ' days')::INTERVAL
  ) < NOW();
-- name: ExecutionsServiceUpdateExecution :one
UPDATE executions
SET
  status = COALESCE(sqlc.narg('status'), status),
  message = COALESCE(sqlc.narg('message'), message),
  path = COALESCE(sqlc.narg('path'), path),
  finished_at = COALESCE(sqlc.narg('finished_at'), finished_at),
  deleted_at = COALESCE(sqlc.narg('deleted_at'), deleted_at),
  file_size = COALESCE(sqlc.narg('file_size'), file_size)
WHERE id = @id
RETURNING *;
-- name: RestorationsServiceCreateRestoration :one
INSERT INTO restorations (execution_id, database_id, status, message)
VALUES (@execution_id, @database_id, @status, @message)
RETURNING *;
-- name: RestorationsServiceGetRestorationsQty :one
SELECT 
  COUNT(*) AS all,
  COALESCE(SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END), 0)::INTEGER AS running,
  COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0)::INTEGER AS success,
  COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)::INTEGER AS failed
FROM restorations;
-- name: RestorationsServicePaginateRestorationsCount :one
SELECT COUNT(restorations.*)
FROM restorations
INNER JOIN executions ON executions.id = restorations.execution_id
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN databases ON databases.id = restorations.database_id
WHERE
(
  sqlc.narg('execution_id')::UUID IS NULL
  OR
  restorations.execution_id = sqlc.narg('execution_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  restorations.database_id = sqlc.narg('database_id')::UUID
);

-- name: RestorationsServicePaginateRestorations :many
SELECT
  restorations.*,
  databases.name AS database_name,
  backups.name AS backup_name
FROM restorations
INNER JOIN executions ON executions.id = restorations.execution_id
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN databases ON databases.id = restorations.database_id
WHERE
(
  sqlc.narg('execution_id')::UUID IS NULL
  OR
  restorations.execution_id = sqlc.narg('execution_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  restorations.database_id = sqlc.narg('database_id')::UUID
)
ORDER BY restorations.started_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: RestorationsServiceUpdateRestoration :one
UPDATE restorations
SET
  status = COALESCE(sqlc.narg('status'), status),
  message = COALESCE(sqlc.narg('message'), message),
  finished_at = COALESCE(sqlc.narg('finished_at'), finished_at)
WHERE id = @id
RETURNING *;
-- name: UsersServiceChangePassword :exec
UPDATE users
SET password = @password
WHERE id = @id;
-- name: UsersServiceCreateUser :one
INSERT INTO users (name, email, password)
VALUES (@name, lower(@email), @password)
RETURNING *;
-- name: UsersServiceGetUsersQty :one
SELECT COUNT(*) FROM users;
-- name: UsersServiceGetUserByEmail :one
SELECT * FROM users WHERE email = @email;
-- name: UsersServiceUpdateUser :one
UPDATE users
SET
  name = COALESCE(sqlc.narg('name'), name),
  email = lower(@email),
  password = COALESCE(sqlc.narg('password'), password)
WHERE id = @id
RETURNING *;
-- name: WebhooksServiceCreateWebhook :one
INSERT INTO webhooks (
  name, is_active, event_type, target_ids,
  url, method, headers, body
) VALUES (
  @name, @is_active, @event_type, @target_ids,
  @url, @method, @headers, @body
) RETURNING *;
-- name: WebhooksServiceDeleteWebhook :exec
DELETE FROM webhooks WHERE id = @webhook_id;
-- name: WebhooksServiceDuplicateWebhook :one
INSERT INTO webhooks
SELECT (
  webhooks
  #= hstore('id', uuid_generate_v4()::text)
  #= hstore('name', (webhooks.name || ' (copy)')::text)
  #= hstore('is_active', false::text)
  #= hstore('created_at', now()::text)
  #= hstore('updated_at', now()::text)
).*
FROM webhooks
WHERE webhooks.id = @webhook_id
RETURNING *;
-- name: WebhooksServiceGetWebhook :one
SELECT * FROM webhooks WHERE id = @webhook_id;
-- name: WebhooksServicePaginateWebhooksCount :one
SELECT COUNT(*) FROM webhooks;

-- name: WebhooksServicePaginateWebhooks :many
SELECT * FROM webhooks
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
-- name: WebhooksServicePaginateWebhookExecutionsCount :one
SELECT COUNT(*) FROM webhook_executions
WHERE webhook_id = @webhook_id;

-- name: WebhooksServicePaginateWebhookExecutions :many
SELECT * FROM webhook_executions
WHERE webhook_id = @webhook_id
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');-- name: WebhooksServiceGetWebhooksToRun :many
SELECT * FROM webhooks
WHERE is_active = true
AND event_type = @event_type
AND @target_id::UUID = ANY(target_ids);

-- name: WebhooksServiceCreateWebhookExecution :one
INSERT INTO webhook_executions (
  webhook_id, req_method, req_headers, req_body,
  res_status, res_headers, res_body, res_duration
)
VALUES (
  @webhook_id, @req_method, @req_headers, @req_body,
  @res_status, @res_headers, @res_body, @res_duration
)
RETURNING *;
-- name: WebhooksServiceUpdateWebhook :one
UPDATE webhooks
SET
  name = COALESCE(sqlc.narg('name'), name),
  is_active = COALESCE(sqlc.narg('is_active'), is_active),
  event_type = COALESCE(sqlc.narg('event_type'), event_type),
  target_ids = COALESCE(sqlc.narg('target_ids'), target_ids),
  url = COALESCE(sqlc.narg('url'), url),
  method = COALESCE(sqlc.narg('method'), method),
  headers = COALESCE(sqlc.narg('headers'), headers),
  body = COALESCE(sqlc.narg('body'), body)
WHERE id = @webhook_id
RETURNING *;