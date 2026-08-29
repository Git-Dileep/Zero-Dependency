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
// Phase 2 scaffold — implementation will be filled in by Phase 2.
package aead

// Key32 is the canonical 32-byte key type shared across all MiniRatchet packages.
type Key32 = [32]byte

// Encrypt encrypts plaintext under the given key with associated data,
// returning a ciphertext that includes the GCM nonce prepended.
//
// TODO(phase2): implement AES-256-GCM encrypt.
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error) {
	return nil, nil // placeholder
}

// Decrypt decrypts ciphertext that was produced by Encrypt, validating the
// associated data and returning the original plaintext.
//
// TODO(phase2): implement AES-256-GCM decrypt.
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error) {
	return nil, nil // placeholder
}
