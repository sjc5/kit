// Package cryptoutil provides utility functions for cryptographic operations.
package cryptoutil

import (
	"errors"

	"golang.org/x/crypto/nacl/auth"
	"golang.org/x/crypto/nacl/sign"
)

// SignSymmetric signs a message using a symmetric key. It is a convenience
// wrapper around the nacl/auth package.
func SignSymmetric(msg []byte, secretKey *[32]byte) ([]byte, error) {
	digest := auth.Sum(msg, secretKey)
	signedMsg := make([]byte, auth.Size+len(msg))
	copy(signedMsg, digest[:])
	copy(signedMsg[auth.Size:], msg)
	return signedMsg, nil
}

// VerifyAndReadSymmetric verifies a signed message using a symmetric key and
// returns the original message. It is a convenience wrapper around the
// nacl/auth package.
func VerifyAndReadSymmetric(signedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	if len(signedMsg) < auth.Size {
		return nil, errors.New("invalid signature")
	}
	digest := make([]byte, auth.Size)
	copy(digest, signedMsg[:auth.Size])
	msg := signedMsg[auth.Size:]
	if !auth.Verify(digest, msg, secretKey) {
		return nil, errors.New("invalid signature")
	}
	return msg, nil
}

// VerifyAndReadAssymetric verifies a signed message using a public key and
// returns the original message. It is a convenience wrapper around the
// nacl/sign package.
func VerifyAndReadAssymetric(signedMsg []byte, publicKey *[32]byte) ([]byte, error) {
	msg, ok := sign.Open(nil, signedMsg, publicKey)
	if !ok {
		return nil, errors.New("invalid signature")
	}
	return msg, nil
}
