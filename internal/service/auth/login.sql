-- name: AuthServiceLoginGetUserByEmail :one
SELECT * FROM users WHERE email = @email;
