-- name: CreateUser :one
INSERT INTO
  users (
    id,
    created_at,
    updated_at,
    email,
    hashed_password
  )
VALUES
  ($1, NOW(), NOW(), $2, $3) RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT
  *
FROM
  users
WHERE
  email = $1;

-- name: GetUserFromRefreshToken :one
SELECT
  *
FROM
  users
  INNER JOIN refresh_tokens ON refresh_tokens.user_id = users.id
WHERE
  refresh_tokens.token = $1;

-- name: UpdateUser :one
UPDATE users
SET
  email = $1,
  hashed_password = $2
WHERE
  id = $3 RETURNING *;
