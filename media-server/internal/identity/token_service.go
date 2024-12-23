package identity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	_ISSUER   = "0.0.0.0"
	TOKEN_USE = "token:use"

	ACCESS_TOKEN  = "access_token"
	REFRESH_TOKEN = "refresh_token"
)

var (
	_DEFAULT_CLAIMS             = []string{"can:join"}
	_ACCESS_TOKEN_EXPIRES_AFTER = time.Minute * 1
)

// By that function may be obtained pub keys
func keyset(kid string, pkey jwk.Key) jwk.Set {
	keyset := jwk.NewSet()
	pbkey, _ := jwk.PublicKeyOf(pkey)
	_ = pbkey.Set(jwk.AlgorithmKey, jwa.RS256)
	_ = pbkey.Set(jwk.KeyIDKey, kid)
	_ = keyset.AddKey(pbkey)
	return keyset
}

func signToken(pkeyJwsMessage string, headers jws.Headers, token jwt.Token) (string, error) {
	signKey, err := jwk.ParseKey([]byte(pkeyJwsMessage))
	if err != nil {
		return "", err
	}

	// TODO: Use elliptic curve sign
	// jwa.ECDH_ES

	byteToken, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", err
	}
	return string(byteToken), nil
}

type TokenService struct{}

func (s *TokenService) CreateAccessToken(user *User, pkeyID uuid.UUID, pkeyJwsMessage string) (string, error) {
	expiresAt := time.Now().Add(_ACCESS_TOKEN_EXPIRES_AFTER)

	b := jwt.NewBuilder().
		Issuer(_ISSUER).
		Audience(_DEFAULT_CLAIMS).
		Subject(user.Username).
		Expiration(expiresAt)

	token, err := b.Build()
	if err != nil {
		return "", err
	}

	if err = token.Set("user:id", user.ID); err != nil {
		return "", fmt.Errorf("Unable set `user:id` claim. Error: %s", err)
	}

	if err = token.Set("token:use", ACCESS_TOKEN); err != nil {
		return "", fmt.Errorf("unable set `token:use` claim. Error: %s", err)
	}

	headers := jws.NewHeaders()
	if err = headers.Set(jws.KeyIDKey, pkeyID.String()); err != nil {
		return "", fmt.Errorf("unable set header `kid`. Error: %s", err)
	}

	return signToken(pkeyJwsMessage, headers, token)
}

func (s *TokenService) CreateRefreshToken(user *User, pkeyID uuid.UUID, pkeyJwsMessage string) (string, error) {
	expiresAt := time.Now().AddDate(1, 0, 0)

	b := jwt.NewBuilder().
		Issuer(_ISSUER).
		Audience(_DEFAULT_CLAIMS).
		Subject(user.Username).
		Expiration(expiresAt)

	token, err := b.Build()
	if err != nil {
		return "", err
	}

	if err = token.Set("user:id", user.ID); err != nil {
		return "", fmt.Errorf("Unable set `user:id` claim. Error: %s", err)
	}

	if err = token.Set(TOKEN_USE, REFRESH_TOKEN); err != nil {
		return "", fmt.Errorf("unable set `token:use` claim. Error: %s", err)
	}

	headers := jws.NewHeaders()
	if err = headers.Set(jws.KeyIDKey, pkeyID.String()); err != nil {
		return "", fmt.Errorf("unable set header `kid`. Error: %s", err)
	}

	return signToken(pkeyJwsMessage, headers, token)
}

func NewTokenService() *TokenService {
	return &TokenService{}
}
