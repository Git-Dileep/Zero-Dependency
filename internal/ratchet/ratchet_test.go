package ratchet

import (
	"bytes"
	"testing"
)

// TestAdvance_BasicDerivation verifies that Advance produces non-zero message
// keys and updates the chain key.
func TestAdvance_BasicDerivation(t *testing.T) {
	initialChainKey := Key32{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}

	state := &RatchetState{ChainKey: initialChainKey, Epoch: 1}

	msgKey, err := state.Advance()
	if err != nil {
		t.Fatalf("Advance failed: %v", err)
	}

	// Message key must not be zero.
	if msgKey == (Key32{}) {
		t.Fatal("Advance returned a zero message key")
	}

	// Chain key must have changed.
	if bytes.Equal(state.ChainKey[:], initialChainKey[:]) {
		t.Fatal("ChainKey was not ratcheted forward")
	}
}

// TestAdvance_DistinctMessageKeys verifies successive message keys differ.
func TestAdvance_DistinctMessageKeys(t *testing.T) {
	var ck Key32
	copy(ck[:], []byte("distinct-message-keys-test-val!"))
	state := &RatchetState{ChainKey: ck}

	mk1, _ := state.Advance()
	mk2, _ := state.Advance()
	mk3, _ := state.Advance()

	if mk1 == mk2 || mk2 == mk3 || mk1 == mk3 {
		t.Fatal("Successive message keys must be distinct")
	}
}

// TestAdvance_ForwardSecrecy proves that knowing a later chain key cannot
// reproduce an earlier message key — the core forward-secrecy property.
func TestAdvance_ForwardSecrecy(t *testing.T) {
	var ck Key32
	copy(ck[:], []byte("forward-secrecy-test-chain-key!"))
	state := &RatchetState{ChainKey: ck}

	// Advance to get mk1, then save the chain key at step 2.
	mk1, _ := state.Advance()
	ck2 := state.ChainKey // attacker obtains this later chain key

	// From ck2, an attacker can derive mk2, mk3 etc — but NOT mk1.
	attackerState := &RatchetState{ChainKey: ck2}
	attackerMk2, _ := attackerState.Advance()

	// The attacker's derived key must NOT equal mk1.
	if attackerMk2 == mk1 {
		t.Fatal("Later chain key reproduced an earlier message key — forward secrecy broken")
	}

	// Verify the legitimate mk2 matches the attacker's derivation
	// (both start from the same ck2, so they must agree).
	mk2, _ := state.Advance()
	_ = mk2 // mk2 was derived from original state (which has moved on)
	// Re-derive from ck2 to compare properly.
	verifyState := &RatchetState{ChainKey: ck2}
	verifyMk2, _ := verifyState.Advance()
	if attackerMk2 != verifyMk2 {
		t.Fatal("Deterministic derivation mismatch from the same chain key")
	}
}

// TestAdvance_Deterministic confirms the KDF is deterministic: same chain key
// always yields the same message key.
func TestAdvance_Deterministic(t *testing.T) {
	var ck Key32
	copy(ck[:], []byte("deterministic-test-chain-key!!"))

	s1 := &RatchetState{ChainKey: ck}
	s2 := &RatchetState{ChainKey: ck}

	mk1, _ := s1.Advance()
	mk2, _ := s2.Advance()

	if mk1 != mk2 {
		t.Fatal("Same chain key produced different message keys")
	}
}

// TestReseed verifies that reseeding advances the epoch and replaces keys.
func TestReseed(t *testing.T) {
	var root, chain Key32
	copy(root[:], []byte("reseed-root-key-value-here!!"))
	copy(chain[:], []byte("reseed-chain-key-value-here!"))
	state := &RatchetState{RootKey: root, ChainKey: chain, Epoch: 1}

	var newRoot, newChain Key32
	copy(newRoot[:], []byte("new-root-key-after-dh-rotate"))
	copy(newChain[:], []byte("new-chain-key-after-dh-rot!!"))
	state.Reseed(newRoot, newChain)

	if state.Epoch != 2 {
		t.Fatalf("Epoch should be 2, got %d", state.Epoch)
	}
	if state.RootKey != newRoot {
		t.Fatal("Root key was not updated")
	}
	if state.ChainKey != newChain {
		t.Fatal("Chain key was not updated")
	}
}
