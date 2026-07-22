-- name: DatabasesServiceGetDatabasesHealth :many
SELECT id, name, test_ok FROM databases ORDER BY name ASC;
