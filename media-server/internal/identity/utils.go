package identity

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func RSA256PkeyAsJwsMessage() ([]byte, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("unable generate rsa private key. Error: %s\n", err)
	}

	key, err := jwk.FromRaw(pk)
	if err != nil {
		return nil, fmt.Errorf("unable cast private key to jwk key. Error: %s\n", err)
	}

	proxy := struct{ Key jwk.Key }{Key: key}

	jsonBytes, err := json.Marshal(proxy.Key)
	if err != nil {
		return nil, fmt.Errorf("unable serialize jwk key as json message. Error: %s\n", err)
	}

	return jsonBytes, nil
}
