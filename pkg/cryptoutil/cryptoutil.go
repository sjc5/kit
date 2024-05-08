package cryptoutil

import (
	"errors"

	"golang.org/x/crypto/nacl/auth"
	"golang.org/x/crypto/nacl/sign"
)

func SignSymmetric(rawMessage []byte, secretBytes *[32]byte) ([]byte, error) {
	digest := auth.Sum(rawMessage, secretBytes)
	bytesNeeded := auth.Size + len(rawMessage)
	encodedValue := make([]byte, bytesNeeded)
	copy(encodedValue, digest[:])
	copy(encodedValue[auth.Size:], rawMessage)
	return encodedValue, nil
}

func ReadSymmetric(signedMessage []byte, secretBytes *[32]byte) ([]byte, error) {
	digest := make([]byte, auth.Size)
	copy(digest, signedMessage[:auth.Size])
	rawValue := signedMessage[auth.Size:]
	if !auth.Verify(digest, rawValue, secretBytes) {
		return nil, errors.New("invalid signature")
	}
	return rawValue, nil
}

func ReadAssymetric(signedMessage []byte, publicKey *[32]byte) ([]byte, error) {
	message, ok := sign.Open(nil, signedMessage, publicKey)
	if !ok {
		return nil, errors.New("invalid signature")
	}
	return message, nil
}
