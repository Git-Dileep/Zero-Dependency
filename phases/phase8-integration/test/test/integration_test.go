package test

import (
	"bytes"
	"testing"
	
	"miniratchet/internal/aead"
	"miniratchet/internal/dhratchet"
	"miniratchet/internal/ratchet"
)

func TestIntegrationEndToEnd(t *testing.T) {
	// Initialize Epoch 1 Keys
	alicePriv, alicePub, err := dhratchet.NewEpochKeyPair()
	if err != nil { t.Fatalf("Alice keys failed: %v", err) }
	
	bobPriv, bobPub, err := dhratchet.NewEpochKeyPair()
	if err != nil { t.Fatalf("Bob keys failed: %v", err) }
	
	aliceShared, _ := alicePriv.ECDH(bobPub)
	bobShared, _ := bobPriv.ECDH(alicePub)
	
	baseRoot := ratchet.Key32{}
	aliceRoot, _ := dhratchet.DeriveRootKey(aliceShared, baseRoot)
	bobRoot, _ := dhratchet.DeriveRootKey(bobShared, baseRoot)
	
	if !bytes.Equal(aliceRoot[:], bobRoot[:]) {
		t.Fatal("Root keys mismatch")
	}
	
	aliceState := &ratchet.RatchetState{RootKey: aliceRoot, ChainKey: aliceRoot, Epoch: 1}
	bobState := &ratchet.RatchetState{RootKey: bobRoot, ChainKey: bobRoot, Epoch: 1}
	
	// Send 5 messages Alice -> Bob
	for i := 1; i <= 5; i++ {
		msgKeyA, _ := aliceState.Advance()
		ad := []byte("msg=" + string(rune(i)))
		
		pt := []byte("Integration test message")
		ct, err := aead.Encrypt(msgKeyA, pt, ad)
		if err != nil { t.Fatalf("Encryption failed: %v", err) }
		
		// Bob decrypts
		msgKeyB, _ := bobState.Advance()
		decrypted, err := aead.Decrypt(msgKeyB, ct, ad)
		if err != nil { t.Fatalf("Decryption failed: %v", err) }
		
		if !bytes.Equal(pt, decrypted) {
			t.Fatal("Decrypted mismatch")
		}
	}
}
