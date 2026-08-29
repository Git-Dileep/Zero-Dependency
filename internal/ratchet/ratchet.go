// Package ratchet implements the symmetric ratchet state machine.
//
// It manages root keys, chain keys, and epoch counters, advancing the
// chain on each sent/received message to derive per-message encryption keys.
//
// Contract (do not change without updating all dependent phases):
//
//	type Key32 = [32]byte
//
//	type RatchetState struct {
//	    RootKey  Key32
//	    ChainKey Key32
//	    Epoch    uint32
//	}
//
//	func (rs *RatchetState) Advance() (messageKey Key32, err error)
//
// Phase 2 — full implementation.
package ratchet

import (
	"crypto/hmac"
	"crypto/sha256"
)

// Key32 is the canonical key type used for chain keys, message keys, and root keys.
type Key32 = [32]byte

// RatchetState holds the symmetric ratchet's mutable state.
// Callers advance the ratchet to derive per-message encryption keys.
type RatchetState struct {
	RootKey  Key32
	ChainKey Key32
	Epoch    uint32
}

// Advance steps the symmetric ratchet forward by one tick, deriving a new
// message key and updating the chain key in place. The old chain key bytes
// are securely overwritten (zeroed) in memory before replacement.
//
// Derivation (HMAC-SHA256 KDF chain):
//   messageKey  = HMAC-SHA256(ChainKey, 0x01)
//   nextChainKey = HMAC-SHA256(ChainKey, 0x02)
func (rs *RatchetState) Advance() (messageKey Key32, err error) {
	// Derive message key: HMAC-SHA256(ChainKey, 0x01)
	hmacMsg := hmac.New(sha256.New, rs.ChainKey[:])
	hmacMsg.Write([]byte{0x01})
	copy(messageKey[:], hmacMsg.Sum(nil))

	// Derive next chain key: HMAC-SHA256(ChainKey, 0x02)
	hmacNext := hmac.New(sha256.New, rs.ChainKey[:])
	hmacNext.Write([]byte{0x02})
	var nextCK Key32
	copy(nextCK[:], hmacNext.Sum(nil))

	// Securely overwrite the old ChainKey bytes before replacing.
	for i := range rs.ChainKey {
		rs.ChainKey[i] = 0
	}
	rs.ChainKey = nextCK

	return messageKey, nil
}

// Reseed resets the ratchet state with a new root key and chain key,
// typically called after a DH epoch rotation. Old key material is zeroed.
func (rs *RatchetState) Reseed(newRootKey, newChainKey Key32) {
	for i := range rs.RootKey {
		rs.RootKey[i] = 0
	}
	for i := range rs.ChainKey {
		rs.ChainKey[i] = 0
	}
	rs.RootKey = newRootKey
	rs.ChainKey = newChainKey
	rs.Epoch++
}
