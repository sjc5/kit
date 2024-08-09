// Package signedcookie provides a secure cookie manager for Go web applications.
package signedcookie

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/bytesutil"
	"github.com/sjc5/kit/pkg/cryptoutil"
)

////////////////////////////////////////////////////////////////////
/////// CORE SIGNED COOKIES MANAGER
////////////////////////////////////////////////////////////////////

const SecretSize = 32 // SecretSize is the size, in bytes, of a cookie secret.

// Manager handles the creation, signing, and verification of secure cookies.
type Manager struct {
	secretsBytes secretsBytes
}

// Secrets is a latest-first list of 32-byte, base64-encoded secrets.
type Secrets []string
type secretsBytes [][SecretSize]byte

// NewManager creates a new Manager instance with the provided secrets.
// It returns an error if no secrets are provided or if any secret is invalid.
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

// VerifyAndReadCookieValue retrieves and verifies the value of a signed cookie.
// It returns an error if the cookie is not found or is invalid.
func (m Manager) VerifyAndReadCookieValue(r *http.Request, key string) (string, error) {
	cookie, err := r.Cookie(key)
	if err != nil {
		return "", err
	}
	value, err := m.verifyAndReadValue(cookie.Value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// NewDeletionCookie creates a new cookie that will delete the specified cookie when sent to the client.
func (m Manager) NewDeletionCookie(cookie *http.Cookie) *http.Cookie {
	newCookie := *cookie
	newCookie.Value = ""
	newCookie.MaxAge = -1
	return &newCookie
}

// NewSignedCookie creates a new cookie with a signed value based on the provided unsigned cookie.
func (m Manager) NewSignedCookie(unsignedCookie *http.Cookie) (*http.Cookie, error) {
	signedValue, err := m.signValue(unsignedCookie.Value)
	if err != nil {
		return nil, err
	}
	newCookie := *unsignedCookie
	newCookie.Value = signedValue
	return &newCookie, nil
}

// signValue signs the provided value using the latest secret.
// It returns the base64-encoded signed value or an error if signing fails.
func (m Manager) signValue(unsignedValue string) (string, error) {
	val, error := cryptoutil.SignSymmetric([]byte(unsignedValue), &m.secretsBytes[0])
	if error != nil {
		return "", error
	}
	return bytesutil.ToBase64(val), nil
}

// verifyAndReadValue verifies and reads the signed value.
// It returns the original unsigned value or an error if verification fails.
func (m Manager) verifyAndReadValue(signedValue string) (string, error) {
	bytes, err := bytesutil.FromBase64(signedValue)
	if err != nil {
		return "", fmt.Errorf("error decoding base64: %v", err)
	}
	for _, secret := range m.secretsBytes {
		value, err := cryptoutil.VerifyAndReadSymmetric(bytes, &secret)
		if err == nil {
			return string(value), nil
		}
	}
	return "", errors.New("cookie not valid")
}

////////////////////////////////////////////////////////////////////
/////// SIGNED COOKIE HELPERS
////////////////////////////////////////////////////////////////////

// BaseCookie is a type alias for http.Cookie.
// It is used for providing and overriding default cookie settings.
// Note that the Name, HttpOnly, and Secure fields are ignored.
// The expires field is not ignored, but it will be overridden by
// an explicitly set TTL.
type BaseCookie *http.Cookie

// SignedCookie provides methods for working with signed cookies of a specific type T.
type SignedCookie[T any] struct {
	Manager    *Manager
	TTL        time.Duration
	BaseCookie BaseCookie
}

// NewSignedCookie creates a new signed cookie with the provided value and optional override settings.
func (sc *SignedCookie[T]) NewSignedCookie(unsignedValue *T, overrideBaseCookie BaseCookie) (*http.Cookie, error) {
	unsignedCookie, err := sc.newUnsignedCookie(unsignedValue, overrideBaseCookie)
	if err != nil {
		return nil, err
	}

	return sc.Manager.NewSignedCookie(unsignedCookie)
}

// NewDeletionCookie creates a new cookie that will delete the current cookie when sent to the client.
func (sc *SignedCookie[T]) NewDeletionCookie() *http.Cookie {
	cookie := newSecureCookieWithoutValue(sc.BaseCookie.Name, nil, nil)
	cookie.MaxAge = -1
	return cookie
}

// VerifyAndReadCookieValue retrieves and verifies the value of the signed cookie from the request.
// It returns the decoded value of type T or an error if retrieval or verification fails.
func (sc *SignedCookie[T]) VerifyAndReadCookieValue(r *http.Request) (*T, error) {
	var instance T

	value, err := sc.Manager.VerifyAndReadCookieValue(r, sc.BaseCookie.Name)
	if err != nil {
		return nil, err
	}

	dataBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	err = bytesutil.FromGobInto(dataBytes, &instance)
	if err != nil {
		return nil, err
	}

	return &instance, nil
}

// newSecureCookieWithoutValue creates a new secure cookie with the provided name, expiration, and base settings.
// It ensures that the cookie is marked as HTTP-only and secure.
func newSecureCookieWithoutValue(name string, expires *time.Time, baseCookie BaseCookie) *http.Cookie {
	newCookie := http.Cookie{}

	if baseCookie != nil {
		newCookie = *baseCookie
	}

	newCookie.Name = name
	if expires != nil {
		newCookie.Expires = *expires
	}

	newCookie.HttpOnly = true
	newCookie.Secure = true

	return &newCookie
}

// newUnsignedCookie creates an unsigned cookie with the provided value and settings.
func (sc *SignedCookie[T]) newUnsignedCookie(unsignedValue *T, overrideBaseCookie BaseCookie) (*http.Cookie, error) {
	dataBytes, err := bytesutil.ToGob(unsignedValue)
	if err != nil {
		return nil, err
	}

	var baseCookieToUse BaseCookie
	if overrideBaseCookie != nil {
		baseCookieToUse = overrideBaseCookie
	} else if sc.BaseCookie != nil {
		baseCookieToUse = sc.BaseCookie
	}

	var expires time.Time
	if sc.TTL != 0 {
		expires = time.Now().Add(sc.TTL)
	}

	unsignedCookie := newSecureCookieWithoutValue(sc.BaseCookie.Name, &expires, baseCookieToUse)
	unsignedCookie.Value = base64.StdEncoding.EncodeToString(dataBytes)

	return unsignedCookie, nil
}
