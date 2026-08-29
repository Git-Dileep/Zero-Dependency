package aead

import (
	"errors"
	"testing"
)

// TestEncryptDecrypt_Roundtrip verifies basic encrypt-then-decrypt.
func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	var key Key32
	copy(key[:], []byte("aead-test-key-32-bytes-long!"))

	plaintext := []byte("Hello, MiniRatchet!")
	ad := []byte("associated-data-v1")

	ct, err := Encrypt(key, plaintext, ad)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if len(ct) == 0 {
		t.Fatal("Encrypt returned empty ciphertext")
	}

	pt, err := Decrypt(key, ct, ad)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if string(pt) != string(plaintext) {
		t.Errorf("Decrypt returned %q, want %q", pt, plaintext)
	}
}

// TestDecrypt_WrongKey verifies that decryption with a wrong key returns
// an *AuthenticationError.
func TestDecrypt_WrongKey(t *testing.T) {
	var key1, key2 Key32
	copy(key1[:], []byte("correct-key-for-encrypt!!!!!"))
	copy(key2[:], []byte("wrong-key-for-decrypt!!!!!!!"))

	ct, err := Encrypt(key1, []byte("secret"), []byte("ad"))
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = Decrypt(key2, ct, []byte("ad"))
	if err == nil {
		t.Fatal("Decrypt with wrong key succeeded; expected AuthenticationError")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T: %v", err, err)
	}
}

// TestDecrypt_TamperedCiphertext verifies tamper detection.
func TestDecrypt_TamperedCiphertext(t *testing.T) {
	var key Key32
	copy(key[:], []byte("tamper-detection-test-key!!!!"))

	ct, err := Encrypt(key, []byte("do not tamper"), []byte("ad"))
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	// Flip one byte in the ciphertext (past the nonce).
	if len(ct) > gcmNonceSize+1 {
		ct[gcmNonceSize+1] ^= 0xFF
	}

	_, err = Decrypt(key, ct, []byte("ad"))
	if err == nil {
		t.Fatal("Decrypt of tampered ciphertext succeeded; expected AuthenticationError")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T: %v", err, err)
	}
}

// TestDecrypt_WrongAssociatedData verifies that mismatched AD is rejected.
func TestDecrypt_WrongAssociatedData(t *testing.T) {
	var key Key32
	copy(key[:], []byte("ad-mismatch-test-key-value!!"))

	ct, err := Encrypt(key, []byte("payload"), []byte("correct-ad"))
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = Decrypt(key, ct, []byte("wrong-ad"))
	if err == nil {
		t.Fatal("Decrypt with wrong AD succeeded; expected AuthenticationError")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T: %v", err, err)
	}
}

// TestEncrypt_DifferentNonces verifies that the same plaintext encrypted
// twice produces different ciphertexts (random nonce).
func TestEncrypt_DifferentNonces(t *testing.T) {
	var key Key32
	copy(key[:], []byte("nonce-uniqueness-test-key!!!"))

	pt := []byte("same plaintext")
	ad := []byte("same ad")

	ct1, _ := Encrypt(key, pt, ad)
	ct2, _ := Encrypt(key, pt, ad)

	if string(ct1) == string(ct2) {
		t.Error("two encryptions of the same plaintext produced identical ciphertexts")
	}
}

// TestDecrypt_TooShortCiphertext verifies rejection of truncated input.
func TestDecrypt_TooShortCiphertext(t *testing.T) {
	var key Key32
	copy(key[:], []byte("short-ct-test-key-value!!!!!"))

	_, err := Decrypt(key, []byte("short"), nil)
	if err == nil {
		t.Fatal("Decrypt accepted ciphertext shorter than nonce size")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T: %v", err, err)
	}
}
