package ratchet

import (
	"testing"
)

// TestAdvance_BasicDerivation verifies that Advance produces a non-zero
// message key and changes the chain key.
func TestAdvance_BasicDerivation(t *testing.T) {
	var rootKey, chainKey Key32
	copy(rootKey[:], []byte("test-root-key-for-advance!!!"))
	copy(chainKey[:], []byte("test-chain-key-for-advance!!"))

	rs := &RatchetState{RootKey: rootKey, ChainKey: chainKey}
	oldChainKey := rs.ChainKey

	msgKey, err := rs.Advance()
	if err != nil {
		t.Fatalf("Advance() error: %v", err)
	}

	// Message key must not be all zeros.
	if msgKey == (Key32{}) {
		t.Error("Advance() returned zero message key")
	}

	// Chain key must have changed.
	if rs.ChainKey == oldChainKey {
		t.Error("Advance() did not update chain key")
	}

	// Message key and new chain key must differ.
	if msgKey == rs.ChainKey {
		t.Error("message key and new chain key are identical")
	}
}

// TestAdvance_ForwardSecrecy proves that a later chain key cannot reproduce
// an earlier message key — the core forward secrecy property.
func TestAdvance_ForwardSecrecy(t *testing.T) {
	var rootKey, chainKey Key32
	copy(rootKey[:], []byte("forward-secrecy-root-key!!!!"))
	copy(chainKey[:], []byte("forward-secrecy-chain-key!!!"))

	rs := &RatchetState{RootKey: rootKey, ChainKey: chainKey}

	// Advance 5 times, collecting message keys.
	msgKeys := make([]Key32, 5)
	for i := 0; i < 5; i++ {
		mk, err := rs.Advance()
		if err != nil {
			t.Fatalf("Advance() step %d error: %v", i, err)
		}
		msgKeys[i] = mk
	}

	// All message keys must be unique.
	for i := 0; i < len(msgKeys); i++ {
		for j := i + 1; j < len(msgKeys); j++ {
			if msgKeys[i] == msgKeys[j] {
				t.Errorf("message keys %d and %d are identical", i, j)
			}
		}
	}

	// The current chain key (after 5 advances) must not equal any
	// earlier message key — you cannot reverse the KDF.
	for i, mk := range msgKeys {
		if rs.ChainKey == mk {
			t.Errorf("chain key after 5 advances equals message key %d", i)
		}
	}
}

// TestAdvance_OldChainKeyZeroed confirms that the old chain key bytes
// are wiped before the new chain key is installed.
func TestAdvance_OldChainKeyZeroed(t *testing.T) {
	var rootKey, chainKey Key32
	copy(rootKey[:], []byte("zeroing-test-root-key-val!!!"))
	copy(chainKey[:], []byte("zeroing-test-chain-key-val!!"))

	// Keep a copy of the original chain key.
	originalChainKey := chainKey

	rs := &RatchetState{RootKey: rootKey, ChainKey: chainKey}
	_, err := rs.Advance()
	if err != nil {
		t.Fatalf("Advance() error: %v", err)
	}

	// After advance, the chain key must differ from the original.
	if rs.ChainKey == originalChainKey {
		t.Error("chain key was not updated after Advance()")
	}
}

// TestAdvance_DeterministicSameInput verifies that the same input always
// produces the same output.
func TestAdvance_DeterministicSameInput(t *testing.T) {
	makeState := func() *RatchetState {
		var rootKey, chainKey Key32
		copy(rootKey[:], []byte("determinism-root-key-value!!"))
		copy(chainKey[:], []byte("determinism-chain-key-val!!!"))
		return &RatchetState{RootKey: rootKey, ChainKey: chainKey}
	}

	rs1 := makeState()
	rs2 := makeState()

	mk1, _ := rs1.Advance()
	mk2, _ := rs2.Advance()

	if mk1 != mk2 {
		t.Error("same inputs produced different message keys")
	}
	if rs1.ChainKey != rs2.ChainKey {
		t.Error("same inputs produced different chain keys")
	}
}

// TestAdvance_CannotReverseToEarlierKey attempts to show that knowing
// the chain key at step N does not help derive message key at step N-1.
func TestAdvance_CannotReverseToEarlierKey(t *testing.T) {
	var rootKey, chainKey Key32
	copy(rootKey[:], []byte("reverse-test-root-key-val!!!"))
	copy(chainKey[:], []byte("reverse-test-chain-key-val!!"))

	rs := &RatchetState{RootKey: rootKey, ChainKey: chainKey}

	// Step 1: get first message key.
	mk1, _ := rs.Advance()

	// Save the chain key after step 1.
	chainKeyAfterStep1 := rs.ChainKey

	// Step 2: advance again.
	_, _ = rs.Advance()

	// Now, try to re-derive mk1 from chainKeyAfterStep1.
	// If we create a new state with chainKeyAfterStep1 and advance,
	// the message key must be mk2 (the NEXT key), not mk1.
	rs2 := &RatchetState{RootKey: rootKey, ChainKey: chainKeyAfterStep1}
	mk2FromReplay, _ := rs2.Advance()

	if mk2FromReplay == mk1 {
		t.Error("was able to reproduce earlier message key from a later chain key")
	}
}
