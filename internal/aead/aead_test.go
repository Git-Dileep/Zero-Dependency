package aead

import (
	"bytes"
	"testing"
)

// TestEncryptDecrypt_Roundtrip verifies basic encrypt-then-decrypt works.
func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := Key32{
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
	}

	plaintext := []byte("Hello MiniRatchet!")
	ad := []byte("msg_count=1")

	ct, err := Encrypt(key, plaintext, ad)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Contains(ct, plaintext) {
		t.Fatal("Ciphertext contains plaintext in the clear")
	}

	decrypted, err := Decrypt(key, ct, ad)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("Decrypted text does not match original plaintext")
	}
}

// TestDecrypt_TamperDetection flips a ciphertext byte and expects failure.
func TestDecrypt_TamperDetection(t *testing.T) {
	var key Key32
	copy(key[:], []byte("tamper-detection-test-key-val"))

	ct, err := Encrypt(key, []byte("secret"), []byte("ad"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Flip a byte in the middle of the ciphertext.
	ct[len(ct)/2] ^= 0xFF

	_, err = Decrypt(key, ct, []byte("ad"))
	if err == nil {
		t.Fatal("Expected decryption to fail after tampering")
	}

	if _, ok := err.(*AuthenticationError); !ok {
		t.Fatalf("Expected *AuthenticationError, got %T: %v", err, err)
	}
}

// TestDecrypt_WrongKey verifies that a different key produces AuthenticationError.
func TestDecrypt_WrongKey(t *testing.T) {
	var key1, key2 Key32
	copy(key1[:], []byte("correct-key-for-encryption!!"))
	copy(key2[:], []byte("wrong-key-for-decryption!!!!"))

	ct, err := Encrypt(key1, []byte("secret"), nil)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key2, ct, nil)
	if err == nil {
		t.Fatal("Expected decryption with wrong key to fail")
	}

	if _, ok := err.(*AuthenticationError); !ok {
		t.Fatalf("Expected *AuthenticationError, got %T: %v", err, err)
	}
}

// TestDecrypt_WrongAssociatedData verifies that mismatched AD fails.
func TestDecrypt_WrongAssociatedData(t *testing.T) {
	var key Key32
	copy(key[:], []byte("associated-data-test-key!!!!"))

	ct, err := Encrypt(key, []byte("secret"), []byte("msg=1"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(key, ct, []byte("msg=2"))
	if err == nil {
		t.Fatal("Expected decryption with wrong associated data to fail")
	}
}

// TestEncrypt_UniqueNonces verifies that two encryptions of the same plaintext
// with the same key produce different ciphertexts (fresh nonces).
func TestEncrypt_UniqueNonces(t *testing.T) {
	var key Key32
	copy(key[:], []byte("nonce-uniqueness-test-key!!!!"))

	pt := []byte("identical plaintext")
	ct1, _ := Encrypt(key, pt, nil)
	ct2, _ := Encrypt(key, pt, nil)

	if bytes.Equal(ct1, ct2) {
		t.Fatal("Two encryptions of the same plaintext produced identical ciphertext — nonce reuse!")
	}
}

// TestDecrypt_CiphertextTooShort verifies short ciphertext is rejected.
func TestDecrypt_CiphertextTooShort(t *testing.T) {
	var key Key32
	_, err := Decrypt(key, []byte("short"), nil)
	if err == nil {
		t.Fatal("Expected error for ciphertext shorter than nonce size")
	}
}
