// Package aead provides authenticated encryption with associated data (AEAD)
// using AES-256-GCM, built entirely from Go's standard library.
//
// Contract (do not change without updating all dependent phases):
//
//	type Key32 = [32]byte
//
//	func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error)
//	func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error)
//
// Phase 2 — full implementation.
package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Key32 is the canonical 32-byte key type shared across all MiniRatchet packages.
type Key32 = [32]byte

// AuthenticationError is returned when decryption fails due to a wrong key,
// tampered ciphertext, or mismatched associated data. The demo layer surfaces
// this error type to judges during the "stolen key" scenario.
type AuthenticationError struct {
	Msg string
}

func (e *AuthenticationError) Error() string {
	return e.Msg
}

// Encrypt encrypts plaintext under the given key with associated data,
// returning a ciphertext that includes a random 12-byte GCM nonce prepended.
//
// Uses AES-256-GCM (crypto/aes + crypto/cipher) with a fresh nonce from
// crypto/rand per message — nonce reuse is impossible as long as the CSPRNG
// functions correctly.
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate a fresh 12-byte nonce via crypto/rand.
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext + auth tag to dst (nonce here),
	// so the output is: nonce || ciphertext || tag.
	ciphertext = aesgcm.Seal(nonce, nonce, plaintext, associatedData)
	return ciphertext, nil
}

// Decrypt extracts the 12-byte nonce prepended to the ciphertext, then
// decrypts and authenticates the remainder using AES-256-GCM.
//
// Returns an *AuthenticationError if the tag verification fails — this is
// the error that appears on screen when the demo attempts decryption with
// a stolen/wrong key.
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err = aesgcm.Open(nil, nonce, actualCiphertext, associatedData)
	if err != nil {
		return nil, &AuthenticationError{Msg: "authentication failed: wrong key or tampered ciphertext"}
	}

	return plaintext, nil
}
