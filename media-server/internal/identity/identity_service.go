package identity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
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
	refreshToken string
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

func (s *IdentityService) getOrCreateUserAccessTokenPrivateKey(ctx context.Context, userID uuid.UUID) (pkeyID *uuid.UUID, pkeyJwsMessage string, err error) {
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
	pkeyID, pkeyJwsMessage, err := s.getOrCreateUserAccessTokenPrivateKey(ctx, user.ID)
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
		refreshToken: refreshToken,
	}, nil
}

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

type TokenContext struct {
	Aud      []string  `json:"aud"`
	Exp      int       `json:"exp"`
	Iss      string    `json:"iss"`
	Sub      string    `json:"sub"`
	TokenUse string    `json:"token:use"`
	UserID   uuid.UUID `json:"user:id"`

	kid            uuid.UUID
	pkeyJwsMessage string
}

func (s *IdentityService) TokenIdentity(ctx context.Context, insecureToken string) (*TokenContext, error) {
	untrustJws, err := jws.Parse([]byte(insecureToken))
	if err != nil {
		return nil, err
	}

	signKid := untrustJws.Signatures()[0].ProtectedHeaders().KeyID()

	privateKeyID, err := uuid.Parse(signKid)
	if err != nil {
		return nil, err
	}

	privKeys, err := s.queries.GetPrivateKeyWithUser(ctx, privateKeyID)
	if err != nil {
		return nil, err
	}

	if len(privKeys) > 1 {
		return nil, ErrSameUserSignedByOnePrivateKey
	} else if len(privKeys) < 1 || !privKeys[0].JwsMessage.Valid {
		return nil, ErrPrivateKeyNotFound
	}

	pkeyJwsMessage := privKeys[0].JwsMessage.RawMessage
	key, err := jwk.ParseKey([]byte(pkeyJwsMessage))
	if err != nil {
		return nil, err
	}

	pubKeys := keyset(privateKeyID.String(), key)

	trusted, err := jws.Verify([]byte(insecureToken), jws.WithKeySet(pubKeys, jws.WithRequireKid(true)))
	if err != nil {
		return nil, err
	}

	payload := &TokenContext{}

	if err = json.Unmarshal(trusted, payload); err != nil {
		return nil, err
	}

	notValid, err := jwt.Parse([]byte(insecureToken), jwt.WithVerify(false), jwt.WithValidate(false))
	if err != nil {
		return nil, err
	}

	if err := jwt.Validate(notValid); err != nil {
		return nil, err
	}

	payload.kid = privateKeyID
	payload.pkeyJwsMessage = string(pkeyJwsMessage)
	return payload, nil
}

func (s *IdentityService) ActualizeTokenPair(ctx context.Context, token *TokenContext) (*tokenPair, error) {
	if token.TokenUse != REFRESH_TOKEN {
		return nil, ErrRefreshTokenConstraintViolation
	}

	storageUser, err := s.queries.GetUser(ctx, token.UserID)
	if err != nil {
		return nil, err
	}

	u := &User{User: storageUser}

	aTokPkeyID, aTokJwsMessage, err := s.getOrCreateUserAccessTokenPrivateKey(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.token.CreateAccessToken(u, *aTokPkeyID, aTokJwsMessage)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.token.CreateRefreshToken(u, token.kid, token.pkeyJwsMessage)
	if err != nil {
		return nil, err
	}

	return &tokenPair{
		AccessToken:  accessToken,
		refreshToken: refreshToken,
	}, nil
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
