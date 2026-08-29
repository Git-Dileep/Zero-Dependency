package aead

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

type Key32 = [32]byte

type AuthenticationError struct {
	Msg string
}

func (e *AuthenticationError) Error() string {
	return e.Msg
}

// Encrypt encrypts plaintext using AES-256-GCM keyed by the provided message key.
// It prepends a randomly generated 12-byte nonce to the resulting ciphertext.
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext and the authentication tag to the prefix (nonce here)
	ciphertext = aesgcm.Seal(nonce, nonce, plaintext, associatedData)
	return ciphertext, nil
}

// Decrypt extracts the 12-byte nonce prepended to the ciphertext and decrypts it.
// It returns an AuthenticationError if authentication fails.
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
