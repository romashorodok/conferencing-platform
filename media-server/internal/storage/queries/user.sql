
-- name: NewUser :one
INSERT INTO users (
    username,
    password
) VALUES (
    users.username = @username,
    users.password = @password
) RETURNING id;

-- name: GetUser :one
SELECT *
FROM users
WHERE users.id = @id;

-- name: AttachUserPrivateKey :exec
INSERT INTO user_private_keys (
    user_id,
    private_key_id
) VALUES (
    user_private_keys.user_id = @user_id,
    user_private_keys.private_key_id = @private_key_id
);

-- name: DetachUserPrivateKey :exec
DELETE FROM user_private_keys
WHERE
    user_private_keys.user_id = @user_id
AND
    user_private_keys.private_key_id = @private_key_id;

-- name: AttachUserRefreshToken :exec
INSERT INTO user_refresh_tokens (
    user_id,
    refresh_token_id
) VALUES (
    user_refresh_tokens.user_id = @user_id,
    user_refresh_tokens.refresh_token_id = @refresh_token_id
);

-- name: DetachUserRefreshToken :exec
DELETE FROM user_refresh_tokens
WHERE
    user_refresh_tokens.user_id = @user_id
AND
    user_refresh_tokens.refresh_token_id = @refresh_token_id;
