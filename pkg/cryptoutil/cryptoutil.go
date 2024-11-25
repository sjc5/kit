// Package cryptoutil provides utility functions for cryptographic operations.
// It is the consumer's responsibility to ensure that inputs are reasonably
// sized so as to avoid memory exhaustion attacks.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/sjc5/kit/pkg/bytesutil"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/auth"
)

type Base64 = string

const (
	// Size of AES-GCM nonce
	aesNonceSize = 12
	// Size of XChaCha20-Poly1305 nonce
	xChaCha20Poly1305NonceSize = 24
	// Size of authentication tag for AES-GCM and XChaCha20-Poly1305
	authTagSize = 16
)

/////////////////////////////////////////////////////////////////////
// SYMMETRIC MESSAGE SIGNING
/////////////////////////////////////////////////////////////////////

// SignSymmetric signs a message using a symmetric key. It is a convenience
// wrapper around the nacl/auth package, which uses HMAC-SHA-512-256.
func SignSymmetric(msg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
	digest := auth.Sum(msg, secretKey)
	signedMsg := make([]byte, auth.Size+len(msg))
	copy(signedMsg, digest[:])
	copy(signedMsg[auth.Size:], msg)
	return signedMsg, nil
}

// VerifyAndReadSymmetric verifies a signed message using a symmetric key and
// returns the original message. It is a convenience wrapper around the
// nacl/auth package, which uses HMAC-SHA-512-256.
func VerifyAndReadSymmetric(signedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
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

/////////////////////////////////////////////////////////////////////
// ASYMMETRIC MESSAGE SIGNING
/////////////////////////////////////////////////////////////////////

// VerifyAndReadAsymmetric verifies a signed message using an Ed25519 public key and
// returns the original message.
func VerifyAndReadAsymmetric(signedMsg []byte, publicKey *[32]byte) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key is required")
	}
	if len(signedMsg) < ed25519.SignatureSize {
		return nil, errors.New("message shorter than signature size")
	}

	ok := ed25519.Verify(publicKey[:], signedMsg[ed25519.SignatureSize:], signedMsg[:ed25519.SignatureSize])
	if !ok {
		return nil, errors.New("invalid signature")
	}

	return signedMsg[ed25519.SignatureSize:], nil
}

// VerifyAndReadAsymmetricBase64 verifies a signed message using a base64
// encoded Ed25519 public key and returns the original message.
func VerifyAndReadAsymmetricBase64(signedMsg Base64, publicKey Base64) ([]byte, error) {
	signedMsgBytes, err := bytesutil.FromBase64(signedMsg)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := bytesutil.FromBase64(publicKey)
	if err != nil {
		return nil, err
	}
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	var publicKey32 [32]byte
	copy(publicKey32[:], publicKeyBytes)

	return VerifyAndReadAsymmetric(signedMsgBytes, &publicKey32)
}

/////////////////////////////////////////////////////////////////////
// FAST HASHING
/////////////////////////////////////////////////////////////////////

// Sha256Hash returns the SHA-256 hash of a message as a byte slice.
func Sha256Hash(msg []byte) []byte {
	hash := sha256.Sum256(msg)
	return hash[:]
}

/////////////////////////////////////////////////////////////////////
// SYMMETRIC ENCRYPTION
/////////////////////////////////////////////////////////////////////

// EncryptSymmetricXChaCha20Poly1305 encrypts a message using XChaCha20-Poly1305.
func EncryptSymmetricXChaCha20Poly1305(msg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}

	aead, err := chacha20poly1305.NewX(secretKey[:])
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, xChaCha20Poly1305NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	encrypted := aead.Seal(nonce, nonce, msg, nil)
	return encrypted, nil
}

// DecryptSymmetricXChaCha20Poly1305 decrypts a message using XChaCha20-Poly1305.
func DecryptSymmetricXChaCha20Poly1305(encryptedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
	if len(encryptedMsg) < xChaCha20Poly1305NonceSize+authTagSize {
		return nil, errors.New("message shorter than nonce + tag size")
	}

	aead, err := chacha20poly1305.NewX(secretKey[:])
	if err != nil {
		return nil, err
	}

	nonce := encryptedMsg[:xChaCha20Poly1305NonceSize]
	ciphertext := encryptedMsg[xChaCha20Poly1305NonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed")
	}

	return plaintext, nil
}

// EncryptSymmetricAESGCM encrypts a message using AES-256-GCM.
func EncryptSymmetricAESGCM(msg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}

	block, err := aes.NewCipher(secretKey[:])
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	encrypted := aesGCM.Seal(nonce, nonce, msg, nil)
	return encrypted, nil
}

// DecryptSymmetricAESGCM decrypts a message using AES-256-GCM.
func DecryptSymmetricAESGCM(encryptedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
	if len(encryptedMsg) < aesNonceSize+authTagSize {
		return nil, errors.New("message shorter than nonce + tag size")
	}

	block, err := aes.NewCipher(secretKey[:])
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := encryptedMsg[:aesNonceSize]
	ciphertext := encryptedMsg[aesNonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed")
	}

	return plaintext, nil
}
