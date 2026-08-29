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
// Phase 2 scaffold — implementation will be filled in by Phase 2.
package ratchet

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
// TODO(phase2): implement KDF chain step.
func (rs *RatchetState) Advance() (messageKey Key32, err error) {
	return Key32{}, nil // placeholder
}
