package signedcookie

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sjc5/kit/pkg/bytesutil"
	"github.com/sjc5/kit/pkg/cryptoutil"
)

const (
	SecretSize = 32 // SecretSize is the size, in bytes, of a cookie secret.
)

type Manager struct {
	secretsBytes secretsBytes
}

// Secrets is a latest-first list of 32-byte, base64-encoded secrets.
type Secrets []string
type secretsBytes [][SecretSize]byte

func NewManager(secrets Secrets) (*Manager, error) {
	if len(secrets) < 1 {
		return nil, errors.New("at least one secret is required")
	}
	secretsBytes := make([][SecretSize]byte, len(secrets))
	for i, secret := range secrets {
		bytes, err := bytesutil.FromBase64(secret)
		if err != nil {
			return nil, fmt.Errorf("error decoding base64: %v", err)
		}
		if len(bytes) != SecretSize {
			return nil, fmt.Errorf("secret %d is not %d bytes", i, SecretSize)
		}
		copy(secretsBytes[i][:], bytes)
	}
	return &Manager{
		secretsBytes: secretsBytes,
	}, nil
}

func (m Manager) Set(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) error {
	encodedValue, err := m.Sign(cookie.Value)
	if err != nil {
		return err
	}
	localCookie := *cookie
	localCookie.Value = encodedValue
	http.SetCookie(w, &localCookie)
	return nil
}

func (m Manager) Get(r *http.Request, key string) (string, error) {
	cookie, err := r.Cookie(key)
	if err != nil {
		return "", errors.New("cookie not found")
	}
	value, err := m.Read(cookie.Value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (m Manager) Delete(w http.ResponseWriter, cookie *http.Cookie) {
	localCookie := *cookie
	localCookie.Value = ""
	localCookie.MaxAge = -1
	http.SetCookie(w, &localCookie)
}

func (m Manager) Sign(rawValue string) (string, error) {
	val, error := cryptoutil.SignSymmetric([]byte(rawValue), &m.secretsBytes[0])
	if error != nil {
		return "", error
	}
	return bytesutil.ToBase64(val), nil
}

func (m Manager) Read(signedValue string) (string, error) {
	bytes, err := bytesutil.FromBase64(signedValue)
	if err != nil {
		return "", fmt.Errorf("error decoding base64: %v", err)
	}
	for _, secret := range m.secretsBytes {
		value, err := cryptoutil.ReadSymmetric(bytes, &secret)
		if err == nil {
			return string(value), nil
		}
	}
	return "", errors.New("cookie not valid")
}
