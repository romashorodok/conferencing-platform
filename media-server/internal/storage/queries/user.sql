
-- name: NewUser :one
INSERT INTO users (
    username,
    password
) VALUES (
    @username,
    @password
) RETURNING id;

-- name: GetUser :one
SELECT *
FROM users
WHERE users.id = @id;

-- name: GetUserByUsername :one
SELECt *
FROM users
WHERE users.username = @username;

-- name: GetUserPrivateKey :one
SELECT
    user_private_keys.private_key_id,
    private_keys.jws_message
FROM user_private_keys
LEFT JOIN private_keys ON private_keys.id = user_private_keys.private_key_id
WHERE user_private_keys.user_id = @user_id
LIMIT 1;

-- name: AttachUserPrivateKey :exec
INSERT INTO user_private_keys (
    user_id,
    private_key_id
) VALUES (
    @user_id,
    @private_key_id
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
    @user_id,
    @refresh_token_id
);

-- name: DetachUserRefreshToken :exec
DELETE FROM user_refresh_tokens
WHERE
    user_refresh_tokens.user_id = @user_id
AND
    user_refresh_tokens.refresh_token_id = @refresh_token_id;
