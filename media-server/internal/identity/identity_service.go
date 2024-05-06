package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/romashorodok/conferencing-platform/media-server/internal/storage"
	"github.com/romashorodok/conferencing-platform/media-server/pkg/sqlutil"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	storage.User
}

type IdentityService struct {
	queries *storage.Queries
	db      *sql.DB
	token   *TokenService
}

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *IdentityService) newUserPrivateKey(ctx context.Context, userID uuid.UUID) (pkeyID *uuid.UUID, pkeyJws string, err error) {
	var jwsMessage []byte
	jwsMessage, err = RSA256PkeyAsJwsMessage()
	if err != nil {
		return
	}
	err = sqlutil.WithTransaction(s.db, func(q *storage.Queries) error {
		pkey, err := q.NewPrivateKey(ctx, jwsMessage)
		if err != nil {
			return err
		}

		err = q.AttachUserPrivateKey(ctx, storage.AttachUserPrivateKeyParams{
			UserID:       userID,
			PrivateKeyID: pkey,
		})
		if err != nil {
			return err
		}

		var jwsKeySet jwk.Set
		jwsKeySet, err = jwk.Parse(jwsMessage)
		if err != nil {
			return err
		}

		var pkeyJwsMessage []byte
		pkeyJwsMessage, err = json.Marshal(&jwsKeySet)
		if err != nil {
			return err
		}

		pkeyJws = string(pkeyJwsMessage)
		pkeyID = &pkey
		return nil
	})
	return
}

func (s *IdentityService) getOrCreateUserPrivateKey(ctx context.Context, userID uuid.UUID) (pkeyID *uuid.UUID, pkeyJwsMessage string, err error) {
	pkeyResult, err := s.queries.GetUserPrivateKey(ctx, userID)
	if err != nil {
		pkeyID, pkeyJwsMessage, err = s.newUserPrivateKey(ctx, userID)
		if err != nil {
			return nil, "", err
		}
		return pkeyID, pkeyJwsMessage, nil
	} else {
		var pkeyJws []byte
		pkeyJws, err = json.Marshal(pkeyResult.JwsMessage.RawMessage)
		if err != nil {
			return nil, "", err
		}
		return &pkeyResult.PrivateKeyID, string(pkeyJws), nil
	}
}

func (s *IdentityService) userNewTokenPair(ctx context.Context, user *User) (*tokenPair, error) {
	pkeyID, pkeyJwsMessage, err := s.getOrCreateUserPrivateKey(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	log.Printf("[User %s] access token jws: %s", user.ID, pkeyJwsMessage)

	accessToken, err := s.token.CreateAccessToken(user, *pkeyID, pkeyJwsMessage)
	if err != nil {
		return nil, err
	}

	pkeyID, pkeyJwsMessage, err = s.newUserPrivateKey(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.CreateRefreshToken(user, *pkeyID, pkeyJwsMessage)
	if err != nil {
		return nil, err
	}
	log.Printf("[User %s] refresh token jws: %s", user.ID, pkeyJwsMessage)

	return &tokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

type signInResult struct{}

func (s *IdentityService) SignIn(ctx context.Context, username, password string) (*tokenPair, error) {
	if username == "" || password == "" {
		return nil, ErrEmptyField
	}

	u, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, errors.Join(ErrInvalidPassword, err)
	}

	pair, err := s.userNewTokenPair(ctx, &User{User: u})
	log.Println("pair", pair, "err:", err)

	return pair, nil
}

func (s *IdentityService) SignUp(ctx context.Context, username, password string) (*tokenPair, error) {
	if username == "" || password == "" {
		return nil, ErrEmptyField
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), 12)

	_, err = s.queries.NewUser(ctx, storage.NewUserParams{
		Username: username,
		Password: string(hashedPass),
	})
	if err != nil {
		return nil, err
	}

	return s.SignIn(ctx, username, password)
}

type NewIdentityServiceParams struct {
	fx.In

	TokenService *TokenService
	Queries      *storage.Queries
	DB           *sql.DB
}

func NewIdentityService(params NewIdentityServiceParams) *IdentityService {
	return &IdentityService{
		queries: params.Queries,
		token:   params.TokenService,
		db:      params.DB,
	}
}
