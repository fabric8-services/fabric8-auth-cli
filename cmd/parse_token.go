package cmd

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/square/go-jose.v2"
)

// NewParseTokenCommand a command to parse tokens on `fabric8-auth` service
func newParseTokenCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "parse",
		Short: "parse a token",
		Args:  cobra.MinimumNArgs(1),
		Run:   parse,
	}

	return c
}

func parse(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debug("parsing token...")
	// connect to autgh service on http://192.168.99.100:31000/api/token/keys
	c := &http.Client{Timeout: 10 * time.Second}
	keysEndpoint := "http://192.168.99.100:31000/api/token/keys"
	res, err := c.Get(keysEndpoint)
	if err != nil {
		log.Errorf("failed to retrieve the keys on '%s': %v", keysEndpoint, err)
	}
	defer func() {
		res.Body.Close()
	}()
	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	keys, err := unmarshalKeys(buf.Bytes())
	if err != nil {
		log.Fatalf("failed to retrieve the keys on '%s': %v", keysEndpoint, err)
	}
	token, err := jwt.Parse(args[0], keyFunc(keys))
	if err != nil {
		log.Fatalf("failed to parse token: %v", err)
	}
	log.Debugf("token parsed")
	jsonClaims, err := json.Marshal(token.Claims)
	if err != nil {
		log.Fatalf("failed to marshall token claims: %v", err)
	}
	fmt.Printf("%s\n", jsonClaims)
}

// JSONKeys the structure for the public keys returned by the `/api/token/keys` endpoint
type JSONKeys struct {
	Keys []interface{} `form:"keys" json:"keys" xml:"keys"`
}

func unmarshalKeys(jsonData []byte) (map[string]*rsa.PublicKey, error) {
	keys := make(map[string]*rsa.PublicKey)
	var raw JSONKeys
	err := json.Unmarshal(jsonData, &raw)
	if err != nil {
		return nil, err
	}
	for _, key := range raw.Keys {
		jsonKeyData, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		kid, rsaKey, err := unmarshalKey(jsonKeyData)
		if err != nil {
			return nil, err
		}
		keys[kid] = rsaKey
	}
	return keys, nil
}

func unmarshalKey(jsonData []byte) (string, *rsa.PublicKey, error) {
	var key *jose.JSONWebKey
	key = &jose.JSONWebKey{}
	err := key.UnmarshalJSON(jsonData)
	if err != nil {
		return "", nil, err
	}
	rsaKey, ok := key.Key.(*rsa.PublicKey)
	if !ok {
		return "", nil, errors.New("Key is not an *rsa.PublicKey")
	}
	return key.KeyID, rsaKey, nil
}

//
func keyFunc(keys map[string]*rsa.PublicKey) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		kid := token.Header["kid"]
		if kid == nil {
			log.Error("there is no 'kid' header in the token")
			return nil, errors.New("there is no 'kid' header in the token")
		}
		key, found := keys[kid.(string)]
		if !found || key == nil {
			log.Error("there is no public key with such ID")
			return nil, errors.Errorf("there is no public key with such ID: %s", kid)
		}
		return key, nil
	}
}
