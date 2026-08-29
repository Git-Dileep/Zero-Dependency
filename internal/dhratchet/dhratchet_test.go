package dhratchet

import (
	"bytes"
	"testing"
)

// TestNewEpochKeyPair verifies that key generation produces valid, distinct keys.
func TestNewEpochKeyPair(t *testing.T) {
	priv1, pub1, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("NewEpochKeyPair() #1 error: %v", err)
	}
	if priv1 == nil || pub1 == nil {
		t.Fatal("NewEpochKeyPair() returned nil key")
	}

	priv2, pub2, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("NewEpochKeyPair() #2 error: %v", err)
	}

	// Two independently generated key pairs must differ.
	if bytes.Equal(pub1.Bytes(), pub2.Bytes()) {
		t.Error("two independently generated public keys are identical — RNG failure?")
	}
	_ = priv2
}

// TestDeriveRootKey_SharedSecret verifies the core property: two parties who
// independently generate key pairs and perform ECDH derive the same shared
// root key when starting from the same old root key.
func TestDeriveRootKey_SharedSecret(t *testing.T) {
	// Alice generates her epoch key pair.
	alicePriv, alicePub, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Alice NewEpochKeyPair: %v", err)
	}

	// Bob generates his epoch key pair.
	bobPriv, bobPub, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Bob NewEpochKeyPair: %v", err)
	}

	// Both sides compute the shared secret via X25519 ECDH.
	aliceShared, err := alicePriv.ECDH(bobPub)
	if err != nil {
		t.Fatalf("Alice ECDH: %v", err)
	}
	bobShared, err := bobPriv.ECDH(alicePub)
	if err != nil {
		t.Fatalf("Bob ECDH: %v", err)
	}

	// X25519 is commutative: alice_priv * bob_pub == bob_priv * alice_pub.
	if !bytes.Equal(aliceShared, bobShared) {
		t.Fatal("ECDH shared secrets differ — X25519 commutativity violated")
	}

	// Both sides derive a new root key from the same old root key.
	var oldRootKey Key32
	copy(oldRootKey[:], []byte("miniratchet-initial-root-key!!")) // 29 bytes, rest zero-padded

	aliceNewRoot, err := DeriveRootKey(aliceShared, oldRootKey)
	if err != nil {
		t.Fatalf("Alice DeriveRootKey: %v", err)
	}
	bobNewRoot, err := DeriveRootKey(bobShared, oldRootKey)
	if err != nil {
		t.Fatalf("Bob DeriveRootKey: %v", err)
	}

	// The derived root keys MUST match.
	if aliceNewRoot != bobNewRoot {
		t.Error("Alice and Bob derived different root keys from the same shared secret")
	}

	// Sanity: new root key should not be all zeros.
	if aliceNewRoot == (Key32{}) {
		t.Error("derived root key is all zeros — KDF produced trivial output")
	}
}

// TestDeriveRootKey_DifferentOldRoots verifies that different old root keys
// yield different new root keys, even with the same shared secret.
func TestDeriveRootKey_DifferentOldRoots(t *testing.T) {
	sharedSecret := []byte("test-shared-secret-32-bytes-long")

	var oldRoot1, oldRoot2 Key32
	copy(oldRoot1[:], []byte("root-key-one-padding-padding-pad"))
	copy(oldRoot2[:], []byte("root-key-two-padding-padding-pad"))

	newRoot1, err := DeriveRootKey(sharedSecret, oldRoot1)
	if err != nil {
		t.Fatalf("DeriveRootKey #1: %v", err)
	}
	newRoot2, err := DeriveRootKey(sharedSecret, oldRoot2)
	if err != nil {
		t.Fatalf("DeriveRootKey #2: %v", err)
	}

	if newRoot1 == newRoot2 {
		t.Error("different old root keys produced the same new root key")
	}
}

// TestDeriveRootKey_EmptySecret verifies that an empty shared secret is rejected.
func TestDeriveRootKey_EmptySecret(t *testing.T) {
	var oldRoot Key32
	_, err := DeriveRootKey(nil, oldRoot)
	if err == nil {
		t.Error("DeriveRootKey accepted nil shared secret; expected error")
	}

	_, err = DeriveRootKey([]byte{}, oldRoot)
	if err == nil {
		t.Error("DeriveRootKey accepted empty shared secret; expected error")
	}
}

// TestComputeSharedSecret verifies the helper function and its symmetry.
func TestComputeSharedSecret(t *testing.T) {
	alicePriv, _, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Alice keygen: %v", err)
	}
	bobPriv, _, err := NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Bob keygen: %v", err)
	}

	aliceSecret, err := ComputeSharedSecret(alicePriv, bobPriv.PublicKey())
	if err != nil {
		t.Fatalf("Alice ComputeSharedSecret: %v", err)
	}

	bobSecret, err := ComputeSharedSecret(bobPriv, alicePriv.PublicKey())
	if err != nil {
		t.Fatalf("Bob ComputeSharedSecret: %v", err)
	}

	if !bytes.Equal(aliceSecret, bobSecret) {
		t.Error("ComputeSharedSecret is not symmetric")
	}
}

// TestComputeSharedSecret_NilKeys verifies nil-key rejection.
func TestComputeSharedSecret_NilKeys(t *testing.T) {
	priv, _, err := NewEpochKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ComputeSharedSecret(nil, priv.PublicKey()); err == nil {
		t.Error("accepted nil private key")
	}
	if _, err := ComputeSharedSecret(priv, nil); err == nil {
		t.Error("accepted nil public key")
	}
}

// TestDeriveRootKey_Deterministic verifies that the same inputs always produce
// the same output (no internal randomness).
func TestDeriveRootKey_Deterministic(t *testing.T) {
	secret := []byte("determinism-test-shared-secret!!")
	var oldRoot Key32
	copy(oldRoot[:], []byte("determinism-old-root-key-value!!"))

	root1, _ := DeriveRootKey(secret, oldRoot)
	root2, _ := DeriveRootKey(secret, oldRoot)

	if root1 != root2 {
		t.Error("DeriveRootKey is non-deterministic")
	}
}
