
-- name: NewPrivateKey :one
INSERT INTO private_keys (
    jws_message
) VALUES (
    @jws_message
) RETURNING id;

-- name: GetPrivateKey :one
SELECT
    jws_message
FROM private_keys
WHERE private_keys.id = @id;
