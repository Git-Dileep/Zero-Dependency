package aead

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := Key32{
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	}

	plaintext := []byte("Hello MiniRatchet!")
	associatedData := []byte("msg_count=1")

	ciphertext, err := Encrypt(key, plaintext, associatedData)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Make sure ciphertext is not plaintext
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("Ciphertext contains plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext, associatedData)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("Decrypted text does not match plaintext")
	}

	// Test tamper detection
	ciphertext[15] ^= 0x01
	_, err = Decrypt(key, ciphertext, associatedData)
	if err == nil {
		t.Fatal("Expected decryption to fail after tampering")
	}

	if _, ok := err.(*AuthenticationError); !ok {
		t.Fatalf("Expected AuthenticationError, got %T: %v", err, err)
	}
}
