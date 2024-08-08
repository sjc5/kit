package cryptoutil

import (
	"errors"

	"golang.org/x/crypto/nacl/auth"
	"golang.org/x/crypto/nacl/sign"
)

func SignSymmetric(msg []byte, secretKey *[32]byte) ([]byte, error) {
	digest := auth.Sum(msg, secretKey)
	signedMsg := make([]byte, auth.Size+len(msg))
	copy(signedMsg, digest[:])
	copy(signedMsg[auth.Size:], msg)
	return signedMsg, nil
}

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

func VerifyAndReadAssymetric(signedMsg []byte, publicKey *[32]byte) ([]byte, error) {
	msg, ok := sign.Open(nil, signedMsg, publicKey)
	if !ok {
		return nil, errors.New("invalid signature")
	}
	return msg, nil
}
