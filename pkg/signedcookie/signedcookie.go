package signedcookie

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/sjc5/kit/pkg/bytesutil"
)

type Manager struct {
	currentSecureCookieInstance  *securecookie.SecureCookie
	previousSecureCookieInstance *securecookie.SecureCookie
	options                      ManagerOptions
}

type ManagerOptions struct {
	CurrentCookieSecret  string // base64 encoded, 32 or 64 bytes
	PreviousCookieSecret string // base64 encoded, 32 or 64 bytes
	SameSite             http.SameSite
	Path                 string
}

func NewManager(options ManagerOptions) (*Manager, error) {
	currentBytes, err := bytesutil.FromBase64(string(options.CurrentCookieSecret))
	if err != nil {
		return nil, err
	}
	if len(currentBytes) != 32 && len(currentBytes) != 64 {
		return nil, errors.New("current cookie secret must be 32 or 64 bytes")
	}
	previousBytes, err := bytesutil.FromBase64(string(options.PreviousCookieSecret))
	if err != nil {
		return nil, err
	}
	if len(previousBytes) != 32 && len(previousBytes) != 64 {
		return nil, errors.New("previous cookie secret must be 32 or 64 bytes")
	}
	currentInstance := securecookie.New(currentBytes, nil)
	previousInstance := securecookie.New(previousBytes, nil)
	if options.SameSite == 0 {
		options.SameSite = http.SameSiteLaxMode
	}
	if options.Path == "" {
		options.Path = "/"
	}
	return &Manager{
		currentSecureCookieInstance:  currentInstance,
		previousSecureCookieInstance: previousInstance,
		options:                      options,
	}, nil
}

func (m Manager) SetCookie(w http.ResponseWriter, r *http.Request, key string, value string) error {
	encodedValue, err := m.Sign(key, value)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     key,
		Value:    encodedValue,
		SameSite: m.options.SameSite,
		HttpOnly: true,
		Secure:   true,
		Path:     m.options.Path,
	})
	return nil
}

func (m Manager) GetCookieValue(r *http.Request, key string) (string, error) {
	cookie, err := r.Cookie(key)
	if err != nil {
		return "", errors.New("cookie not found")
	}
	value, err := m.Read(key, cookie.Value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (m Manager) DeleteCookie(w http.ResponseWriter, key string) {
	http.SetCookie(w, &http.Cookie{
		Name:     key,
		Value:    "",
		SameSite: m.options.SameSite,
		HttpOnly: true,
		Secure:   true,
		Path:     m.options.Path,
		MaxAge:   -1,
	})
}

func (m Manager) Sign(key string, value string) (string, error) {
	encodedValue, err := m.currentSecureCookieInstance.Encode(key, value)
	if err != nil {
		return "", err
	}
	return encodedValue, nil
}

func (m Manager) Read(key string, encodedValue string) (string, error) {
	var value string
	err := m.currentSecureCookieInstance.Decode(key, encodedValue, &value)
	if err != nil {
		fmt.Printf("Falling back to previous cookie secret for key. Error: %s\n", err)
		err = m.previousSecureCookieInstance.Decode(key, encodedValue, &value)
	}
	return value, err
}
