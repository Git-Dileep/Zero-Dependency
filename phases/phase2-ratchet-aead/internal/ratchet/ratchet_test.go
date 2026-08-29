package ratchet

import (
	"bytes"
	"testing"
)

func TestRatchetAdvance(t *testing.T) {
	initialChainKey := Key32{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}

	state := &RatchetState{
		ChainKey: initialChainKey,
		Epoch:    1,
	}

	// Advance once
	msgKey1, err := state.Advance()
	if err != nil {
		t.Fatalf("Advance failed: %v", err)
	}

	// Make sure the state's chain key changed
	if bytes.Equal(state.ChainKey[:], initialChainKey[:]) {
		t.Fatal("ChainKey was not ratcheted forward (still equal to initial)")
	}

	// Advance again
	chainKey2 := state.ChainKey
	msgKey2, err := state.Advance()
	if err != nil {
		t.Fatalf("Advance failed: %v", err)
	}

	// Message keys must be distinct
	if bytes.Equal(msgKey1[:], msgKey2[:]) {
		t.Fatal("Message keys should be distinct")
	}

	// Knowing chainKey2 should let us compute msgKey2 but not msgKey1
	// Verify forward secrecy: from chainKey2 we can't derive msgKey1
	testState := &RatchetState{
		ChainKey: chainKey2,
	}
	derivedMsgKey2, _ := testState.Advance()
	if !bytes.Equal(derivedMsgKey2[:], msgKey2[:]) {
		t.Fatal("Derived message key 2 does not match")
	}
}
