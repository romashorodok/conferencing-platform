
-- name: NewRefreshToken :one
INSERT INTO refresh_tokens (
    private_key_id,
    plaintext,
    expires_at
) VALUES (
    refresh_tokens.private_key_id = @private_key_id,
    refresh_tokens.plaintext = @plaintext,
    refresh_tokens.expires_at = @expires_at
) RETURNING id;

-- name: DelRefreshToken :exec
DELETE FROM refresh_tokens WHERE refresh_tokens.id = @id;

