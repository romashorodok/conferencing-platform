// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package storage

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PrivateKey struct {
	ID         uuid.UUID
	JwsMessage json.RawMessage
}

type RefreshToken struct {
	ID           uuid.UUID
	PrivateKeyID uuid.UUID
	Plaintext    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

type User struct {
	ID       uuid.UUID
	Username string
	Password string
}

type UserPrivateKey struct {
	UserID       uuid.UUID
	PrivateKeyID uuid.UUID
}

type UserRefreshToken struct {
	UserID         uuid.UUID
	RefreshTokenID uuid.UUID
}
