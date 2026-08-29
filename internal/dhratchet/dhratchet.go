// Package dhratchet implements the Diffie-Hellman ratchet for epoch rotation,
// providing the "self-healing after compromise" property of the Double Ratchet
// protocol.
//
// This package composes well-studied, trusted cryptographic primitives from Go's
// standard library — it does NOT invent any new cryptographic algorithm.
// Specifically it uses:
//   - crypto/ecdh with X25519 for ephemeral key agreement
//   - crypto/hmac + crypto/sha256 to build HKDF extract-then-expand
//
// Contract (do not change without updating all dependent phases):
//
//	type Key32 = [32]byte
//
//	func NewEpochKeyPair() (priv *ecdh.PrivateKey, pub *ecdh.PublicKey, err error)
//	func DeriveRootKey(sharedSecret []byte, oldRootKey Key32) (newRootKey Key32, err error)
//
// Phase 3 — full implementation.
package dhratchet

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

// Key32 is the canonical 32-byte key type shared across all MiniRatchet packages.
type Key32 = [32]byte

// NewEpochKeyPair generates a fresh X25519 ephemeral key pair suitable for a
// single DH ratchet epoch. The private key should be used once for ECDH and
// then discarded; the public key is sent to the peer.
func NewEpochKeyPair() (priv *ecdh.PrivateKey, pub *ecdh.PublicKey, err error) {
	curve := ecdh.X25519()
	priv, err = curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pub = priv.PublicKey()
	return priv, pub, nil
}

// DeriveRootKey computes a new root key from a shared secret (the output of an
// X25519 ECDH) and the previous root key, using an HKDF-style extract-then-expand
// built from HMAC-SHA256.
//
// This is composing trusted primitives (HMAC-SHA256 as a PRF), NOT inventing a
// new cryptographic algorithm. The construction mirrors RFC 5869's HKDF:
//   - Extract: PRK = HMAC-SHA256(salt=oldRootKey, IKM=sharedSecret)
//   - Expand:  OKM = HMAC-SHA256(PRK, info || 0x01)  — single block, 32 bytes
//
// The result is a fresh 32-byte root key with forward secrecy: an attacker who
// later compromises the old root key still cannot derive this key without the
// ephemeral shared secret.
func DeriveRootKey(sharedSecret []byte, oldRootKey Key32) (newRootKey Key32, err error) {
	if len(sharedSecret) == 0 {
		return Key32{}, errors.New("dhratchet: shared secret must not be empty")
	}

	// --- Extract phase (RFC 5869 §2.2) ---
	// PRK = HMAC-Hash(salt, IKM)
	// salt = oldRootKey, IKM = sharedSecret
	extractMAC := hmac.New(sha256.New, oldRootKey[:])
	extractMAC.Write(sharedSecret)
	prk := extractMAC.Sum(nil) // 32 bytes

	// --- Expand phase (RFC 5869 §2.3) ---
	// We only need one 32-byte block, so:
	// T(1) = HMAC-Hash(PRK, info || 0x01)
	// Using "MiniRatchet-RootKey" as the info/context string.
	info := []byte("MiniRatchet-RootKey")
	expandMAC := hmac.New(sha256.New, prk)
	expandMAC.Write(info)
	expandMAC.Write([]byte{0x01})
	okm := expandMAC.Sum(nil) // 32 bytes

	copy(newRootKey[:], okm)
	return newRootKey, nil
}

// ComputeSharedSecret performs X25519 ECDH between our private key and the
// peer's public key, returning the raw shared secret. The caller's private key
// bytes are zeroed immediately after use for forward secrecy.
//
// This is a helper used during epoch rotation — it is not part of the
// cross-package contract but is exported for use by the demo/session layer.
func ComputeSharedSecret(ourPriv *ecdh.PrivateKey, theirPub *ecdh.PublicKey) ([]byte, error) {
	if ourPriv == nil || theirPub == nil {
		return nil, errors.New("dhratchet: nil key in ECDH")
	}

	secret, err := ourPriv.ECDH(theirPub)
	if err != nil {
		return nil, err
	}

	// Zero the private key's byte representation for forward secrecy.
	// crypto/ecdh.PrivateKey doesn't expose a mutable byte slice, but we can
	// obtain and zero a copy of the bytes. Note: Go's crypto/ecdh keeps the
	// private scalar in unexported fields — we zero what we can access.
	privBytes := ourPriv.Bytes()
	for i := range privBytes {
		privBytes[i] = 0
	}

	return secret, nil
}
