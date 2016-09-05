package signature

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
)

/*
 * Sign-es the message and encodes it in Base64 format.
 */
func SignBase64EncodeMessage(msg string) (string, error) {
	h := sha256.New()
	h.Write([]byte(msg))
	d := h.Sum(nil)
	signed, err := rsa.SignPKCS1v15(nil, private_key, crypto.SHA256, d)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signed), nil
}

var private_key *rsa.PrivateKey

func init() {
	var err error
	if private_key, err = loadPrivateKey("./private_key.pem"); err != nil {
		panic(fmt.Errorf("Can't load private key '%s'", err.Error()))
	}
}

func loadPrivateKey(key_path string) (*rsa.PrivateKey, error) {
	contents, err := ioutil.ReadFile(key_path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, errors.New("can't decode key, no key found")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		rsa, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return rsa, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", block.Type)
	}
}
