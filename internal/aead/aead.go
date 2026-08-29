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
	"fmt"
	"io"
)

// Key32 is the canonical 32-byte key type shared across all MiniRatchet packages.
type Key32 = [32]byte

// AuthenticationError is returned by Decrypt when the ciphertext has been
// tampered with or the wrong key is used. Callers can type-assert on this
// to distinguish authentication failures from other errors.
type AuthenticationError struct {
	Reason string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("aead: authentication failed: %s", e.Reason)
}

// gcmNonceSize is the standard 12-byte nonce size for AES-GCM.
const gcmNonceSize = 12

// Encrypt encrypts plaintext under the given 256-bit key with associated data,
// returning a ciphertext that includes the GCM nonce prepended.
//
// Output format: [12-byte nonce][GCM ciphertext + tag]
//
// The nonce is generated from crypto/rand for each call, so the same
// plaintext encrypted twice will produce different ciphertexts.
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("aead: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aead: cipher.NewGCM: %w", err)
	}

	// Generate a random 12-byte nonce.
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("aead: nonce generation: %w", err)
	}

	// Seal: encrypts and authenticates plaintext with the associated data.
	// The ciphertext is appended to the nonce so the output is [nonce][ct+tag].
	sealed := gcm.Seal(nonce, nonce, plaintext, associatedData)

	return sealed, nil
}

// Decrypt decrypts ciphertext that was produced by Encrypt, validating the
// associated data and returning the original plaintext.
//
// Returns an *AuthenticationError if the ciphertext has been tampered with,
// the wrong key is used, or the associated data does not match.
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error) {
	if len(ciphertext) < gcmNonceSize {
		return nil, &AuthenticationError{Reason: "ciphertext too short"}
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("aead: aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aead: cipher.NewGCM: %w", err)
	}

	// Split nonce and actual ciphertext+tag.
	nonce := ciphertext[:gcmNonceSize]
	ct := ciphertext[gcmNonceSize:]

	// Open: decrypts and verifies authenticity.
	plaintext, err = gcm.Open(nil, nonce, ct, associatedData)
	if err != nil {
		return nil, &AuthenticationError{Reason: err.Error()}
	}

	return plaintext, nil
}
