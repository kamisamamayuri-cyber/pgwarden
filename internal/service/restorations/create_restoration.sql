-- name: RestorationsServiceCreateRestoration :one
INSERT INTO restorations (execution_id, database_id, target_database_name, status, message)
VALUES (@execution_id, @database_id, @target_database_name, @status, @message)
RETURNING *;
