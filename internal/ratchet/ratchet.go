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
// message key and updating the chain key in place.
//
// The derivation uses HMAC-SHA256 as a KDF:
//   - messageKey  = HMAC-SHA256(ChainKey, 0x01)
//   - nextChainKey = HMAC-SHA256(ChainKey, 0x02)
//
// The old ChainKey bytes are zeroed before being replaced, ensuring that
// a compromised chain key cannot be used to recover earlier message keys
// (forward secrecy within an epoch).
func (rs *RatchetState) Advance() (messageKey Key32, err error) {
	// Derive the message key: HMAC-SHA256(ChainKey, 0x01).
	msgMAC := hmac.New(sha256.New, rs.ChainKey[:])
	msgMAC.Write([]byte{0x01})
	msgKeySlice := msgMAC.Sum(nil)
	copy(messageKey[:], msgKeySlice)

	// Derive the next chain key: HMAC-SHA256(ChainKey, 0x02).
	chainMAC := hmac.New(sha256.New, rs.ChainKey[:])
	chainMAC.Write([]byte{0x02})
	nextChainKey := chainMAC.Sum(nil)

	// Zero the old chain key bytes before replacing (forward secrecy).
	for i := range rs.ChainKey {
		rs.ChainKey[i] = 0
	}

	// Install the new chain key.
	copy(rs.ChainKey[:], nextChainKey)

	return messageKey, nil
}
