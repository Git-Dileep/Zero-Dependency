package integration_test

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/miniratchet/internal/aead"
	"github.com/miniratchet/internal/dhratchet"
	"github.com/miniratchet/internal/ratchet"
	"github.com/miniratchet/internal/transport"
)

// TestEndToEnd_AliceBob verifies a full encrypted conversation between
// two parties sharing a DH-derived root key over a TCP transport.
func TestEndToEnd_AliceBob(t *testing.T) {
	// --- Key exchange ---
	alicePriv, alicePub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Alice keygen: %v", err)
	}
	bobPriv, bobPub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Bob keygen: %v", err)
	}

	aliceShared, err := dhratchet.ComputeSharedSecret(alicePriv, bobPub)
	if err != nil {
		t.Fatalf("Alice shared secret: %v", err)
	}
	bobShared, err := dhratchet.ComputeSharedSecret(bobPriv, alicePub)
	if err != nil {
		t.Fatalf("Bob shared secret: %v", err)
	}

	if !bytes.Equal(aliceShared, bobShared) {
		t.Fatal("Shared secrets do not match")
	}

	baseRoot := ratchet.Key32{}
	aliceRoot, err := dhratchet.DeriveRootKey(aliceShared, baseRoot)
	if err != nil {
		t.Fatalf("Alice DeriveRootKey: %v", err)
	}
	bobRoot, err := dhratchet.DeriveRootKey(bobShared, baseRoot)
	if err != nil {
		t.Fatalf("Bob DeriveRootKey: %v", err)
	}

	if aliceRoot != bobRoot {
		t.Fatal("Derived root keys do not match")
	}

	// --- Set up ratchet states ---
	aliceState := &ratchet.RatchetState{RootKey: aliceRoot, ChainKey: aliceRoot, Epoch: 1}
	bobState := &ratchet.RatchetState{RootKey: bobRoot, ChainKey: bobRoot, Epoch: 1}

	// --- Transport layer ---
	aliceConn, bobConn := net.Pipe()
	defer aliceConn.Close()
	defer bobConn.Close()

	messages := []string{
		"Hey Bob, the eagle has landed.",
		"Rendezvous at midnight.",
		"Operation is a go.",
		"Stay safe.",
		"Over and out.",
	}

	var wg sync.WaitGroup

	// Alice sends
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, msg := range messages {
			mk, err := aliceState.Advance()
			if err != nil {
				t.Errorf("Alice advance %d: %v", i, err)
				return
			}
			ad := []byte(fmt.Sprintf("msg=%d", i+1))
			ct, err := aead.Encrypt(mk, []byte(msg), ad)
			if err != nil {
				t.Errorf("Alice encrypt %d: %v", i, err)
				return
			}
			if err := transport.SendFrame(aliceConn, ct); err != nil {
				t.Errorf("Alice send %d: %v", i, err)
				return
			}
		}
	}()

	// Bob receives
	for i, want := range messages {
		frame, err := transport.RecvFrame(bobConn)
		if err != nil {
			t.Fatalf("Bob recv %d: %v", i, err)
		}
		mk, err := bobState.Advance()
		if err != nil {
			t.Fatalf("Bob advance %d: %v", i, err)
		}
		ad := []byte(fmt.Sprintf("msg=%d", i+1))
		pt, err := aead.Decrypt(mk, frame, ad)
		if err != nil {
			t.Fatalf("Bob decrypt %d: %v", i, err)
		}
		if string(pt) != want {
			t.Fatalf("Message %d mismatch: got %q, want %q", i, string(pt), want)
		}
	}

	wg.Wait()
}

// TestReplayRejection verifies that replaying a captured ciphertext is rejected
// because the associated-data counter will have moved on.
func TestReplayRejection(t *testing.T) {
	var root ratchet.Key32
	copy(root[:], []byte("replay-rejection-test-root-k"))

	aliceState := &ratchet.RatchetState{ChainKey: root, Epoch: 1}
	bobState := &ratchet.RatchetState{ChainKey: root, Epoch: 1}

	// Alice sends message 1
	mk1a, _ := aliceState.Advance()
	ad1 := []byte("msg=1")
	ct1, _ := aead.Encrypt(mk1a, []byte("first message"), ad1)

	// Bob decrypts message 1 successfully
	mk1b, _ := bobState.Advance()
	_, err := aead.Decrypt(mk1b, ct1, ad1)
	if err != nil {
		t.Fatalf("Legitimate decrypt failed: %v", err)
	}

	// Alice sends message 2
	mk2a, _ := aliceState.Advance()
	ad2 := []byte("msg=2")
	_, _ = aead.Encrypt(mk2a, []byte("second message"), ad2)

	// Attacker replays ct1 but with ad2 (the current counter)
	mk2b, _ := bobState.Advance()
	_, err = aead.Decrypt(mk2b, ct1, ad2)
	if err == nil {
		t.Fatal("Replay with different key should have failed")
	}
}
