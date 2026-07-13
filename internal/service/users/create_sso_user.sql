-- name: UsersServiceCreateSsoUser :one
INSERT INTO users (name, email, password)
VALUES (@name, lower(@email), NULL)
RETURNING *;
