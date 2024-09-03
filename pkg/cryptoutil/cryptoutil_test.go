package cryptoutil

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/nacl/auth"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/sign"
)

func new32() *[32]byte {
	return &[32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func TestSignSymmetric(t *testing.T) {
	secretKey := new32()
	message := []byte("test message")

	signedMsg, err := SignSymmetric(message, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(signedMsg) != auth.Size+len(message) {
		t.Fatalf("expected signed message length %d, got %d", auth.Size+len(message), len(signedMsg))
	}

	// Test that the signed message contains the original message
	if !bytes.Equal(signedMsg[auth.Size:], message) {
		t.Fatalf("expected signed message to contain original message")
	}
}

func TestVerifyAndReadSymmetric(t *testing.T) {
	secretKey := new32()
	message := []byte("test message")

	signedMsg, _ := SignSymmetric(message, secretKey)

	// Successful verification
	retrievedMsg, err := VerifyAndReadSymmetric(signedMsg, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid signature (corrupt the signed message)
	signedMsg[0] ^= 0xFF // flip a bit in the signature
	_, err = VerifyAndReadSymmetric(signedMsg, secretKey)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}

	// Truncated message
	truncatedMsg := signedMsg[:auth.Size-1]
	_, err = VerifyAndReadSymmetric(truncatedMsg, secretKey)
	if err == nil {
		t.Fatalf("expected error due to truncated message, got nil")
	}
}

func TestVerifyAndReadAssymetric(t *testing.T) {
	publicKey, privateKey, _ := sign.GenerateKey(nil)
	message := []byte("test message")

	signedMsg := sign.Sign(nil, message, privateKey)

	// Successful verification
	retrievedMsg, err := VerifyAndReadAssymetric(signedMsg, publicKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid signature (corrupt the signed message)
	signedMsg[0] ^= 0xFF // flip a bit in the signature
	_, err = VerifyAndReadAssymetric(signedMsg, publicKey)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}

	// Truncated message
	truncatedMsg := signedMsg[:len(signedMsg)-1]
	_, err = VerifyAndReadAssymetric(truncatedMsg, publicKey)
	if err == nil {
		t.Fatalf("expected error due to truncated message, got nil")
	}
}

func TestVerifyAndReadAssymetricBase64(t *testing.T) {
	publicKey, privateKey, _ := sign.GenerateKey(nil)
	message := []byte("test message")

	signedMsg := sign.Sign(nil, message, privateKey)
	signedMsgBase64 := base64.StdEncoding.EncodeToString(signedMsg)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey[:])

	// Successful verification
	retrievedMsg, err := VerifyAndReadAssymetricBase64(signedMsgBase64, publicKeyBase64)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid base64 signature
	_, err = VerifyAndReadAssymetricBase64("invalid_base64", publicKeyBase64)
	if err == nil {
		t.Fatalf("expected error due to invalid base64 signature, got nil")
	}

	// Invalid base64 public key
	_, err = VerifyAndReadAssymetricBase64(signedMsgBase64, "invalid_base64")
	if err == nil {
		t.Fatalf("expected error due to invalid base64 public key, got nil")
	}

	// Invalid signature (corrupt the signed message)
	signedMsgBase64 = base64.StdEncoding.EncodeToString(append(signedMsg[:len(signedMsg)-1], signedMsg[len(signedMsg)-1]^0xFF))
	_, err = VerifyAndReadAssymetricBase64(signedMsgBase64, publicKeyBase64)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}
}

func TestEdgeCases(t *testing.T) {
	secretKey := new32()
	publicKey, _, _ := box.GenerateKey(rand.Reader)

	// Empty message for symmetric signing
	signedMsg, err := SignSymmetric([]byte{}, secretKey)
	if err != nil {
		t.Fatalf("expected no error for empty message, got %v", err)
	}
	if len(signedMsg) != auth.Size {
		t.Fatalf("expected signed message length %d, got %d", auth.Size, len(signedMsg))
	}

	// Empty signed message for symmetric verification
	_, err = VerifyAndReadSymmetric([]byte{}, secretKey)
	if err == nil {
		t.Fatalf("expected error due to empty signed message, got nil")
	}

	// Empty signed message for asymmetric verification
	_, err = VerifyAndReadAssymetric([]byte{}, publicKey)
	if err == nil {
		t.Fatalf("expected error due to empty signed message, got nil")
	}

	// Nil secret key for symmetric signing
	_, err = SignSymmetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil secret key, got nil")
	}

	// Nil secret key for symmetric verification
	_, err = VerifyAndReadSymmetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil secret key, got nil")
	}

	// Nil public key for asymmetric verification
	_, err = VerifyAndReadAssymetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil public key, got nil")
	}
}
