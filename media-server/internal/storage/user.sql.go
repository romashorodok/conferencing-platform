// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: user.sql

package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

const attachUserPrivateKey = `-- name: AttachUserPrivateKey :exec
INSERT INTO user_private_keys (
    user_id,
    private_key_id
) VALUES (
    $1,
    $2
)
`

type AttachUserPrivateKeyParams struct {
	UserID       uuid.UUID
	PrivateKeyID uuid.UUID
}

func (q *Queries) AttachUserPrivateKey(ctx context.Context, arg AttachUserPrivateKeyParams) error {
	_, err := q.exec(ctx, q.attachUserPrivateKeyStmt, attachUserPrivateKey, arg.UserID, arg.PrivateKeyID)
	return err
}

const attachUserRefreshToken = `-- name: AttachUserRefreshToken :exec
INSERT INTO user_refresh_tokens (
    user_id,
    refresh_token_id
) VALUES (
    $1,
    $2
)
`

type AttachUserRefreshTokenParams struct {
	UserID         uuid.UUID
	RefreshTokenID uuid.UUID
}

func (q *Queries) AttachUserRefreshToken(ctx context.Context, arg AttachUserRefreshTokenParams) error {
	_, err := q.exec(ctx, q.attachUserRefreshTokenStmt, attachUserRefreshToken, arg.UserID, arg.RefreshTokenID)
	return err
}

const detachUserPrivateKey = `-- name: DetachUserPrivateKey :exec
DELETE FROM user_private_keys
WHERE
    user_private_keys.user_id = $1
AND
    user_private_keys.private_key_id = $2
`

type DetachUserPrivateKeyParams struct {
	UserID       uuid.UUID
	PrivateKeyID uuid.UUID
}

func (q *Queries) DetachUserPrivateKey(ctx context.Context, arg DetachUserPrivateKeyParams) error {
	_, err := q.exec(ctx, q.detachUserPrivateKeyStmt, detachUserPrivateKey, arg.UserID, arg.PrivateKeyID)
	return err
}

const detachUserRefreshToken = `-- name: DetachUserRefreshToken :exec
DELETE FROM user_refresh_tokens
WHERE
    user_refresh_tokens.user_id = $1
AND
    user_refresh_tokens.refresh_token_id = $2
`

type DetachUserRefreshTokenParams struct {
	UserID         uuid.UUID
	RefreshTokenID uuid.UUID
}

func (q *Queries) DetachUserRefreshToken(ctx context.Context, arg DetachUserRefreshTokenParams) error {
	_, err := q.exec(ctx, q.detachUserRefreshTokenStmt, detachUserRefreshToken, arg.UserID, arg.RefreshTokenID)
	return err
}

const getUser = `-- name: GetUser :one
SELECT id, username, password
FROM users
WHERE users.id = $1
`

func (q *Queries) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	row := q.queryRow(ctx, q.getUserStmt, getUser, id)
	var i User
	err := row.Scan(&i.ID, &i.Username, &i.Password)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECt id, username, password
FROM users
WHERE users.username = $1
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := q.queryRow(ctx, q.getUserByUsernameStmt, getUserByUsername, username)
	var i User
	err := row.Scan(&i.ID, &i.Username, &i.Password)
	return i, err
}

const getUserPrivateKey = `-- name: GetUserPrivateKey :one
SELECT
    user_private_keys.private_key_id,
    private_keys.jws_message
FROM user_private_keys
LEFT JOIN private_keys ON private_keys.id = user_private_keys.private_key_id
WHERE user_private_keys.user_id = $1
LIMIT 1
`

type GetUserPrivateKeyRow struct {
	PrivateKeyID uuid.UUID
	JwsMessage   pqtype.NullRawMessage
}

func (q *Queries) GetUserPrivateKey(ctx context.Context, userID uuid.UUID) (GetUserPrivateKeyRow, error) {
	row := q.queryRow(ctx, q.getUserPrivateKeyStmt, getUserPrivateKey, userID)
	var i GetUserPrivateKeyRow
	err := row.Scan(&i.PrivateKeyID, &i.JwsMessage)
	return i, err
}

const newUser = `-- name: NewUser :one
INSERT INTO users (
    username,
    password
) VALUES (
    $1,
    $2
) RETURNING id
`

type NewUserParams struct {
	Username string
	Password string
}

func (q *Queries) NewUser(ctx context.Context, arg NewUserParams) (uuid.UUID, error) {
	row := q.queryRow(ctx, q.newUserStmt, newUser, arg.Username, arg.Password)
	var id uuid.UUID
	err := row.Scan(&id)
	return id, err
}