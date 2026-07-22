-- name: DestinationsServiceGetDestinationsHealth :many
SELECT id, name, test_ok FROM destinations ORDER BY name ASC;
