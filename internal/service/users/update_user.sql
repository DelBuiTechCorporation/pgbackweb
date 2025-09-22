-- name: UsersServiceUpdateUser :one
UPDATE users
SET
  name = COALESCE(sqlc.narg('name'), name),
  email = lower(@email),
  password = COALESCE(sqlc.narg('password'), password)
WHERE id = @id
RETURNING *;
