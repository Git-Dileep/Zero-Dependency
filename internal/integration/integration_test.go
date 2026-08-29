// Package integration_test provides cross-module integration tests for
// MiniRatchet, validating that ratchet, aead, dhratchet, and transport
// work correctly together.
//
// Phase 6 — full implementation.
package integration_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miniratchet/internal/aead"
	"github.com/miniratchet/internal/dhratchet"
	"github.com/miniratchet/internal/ratchet"
	"github.com/miniratchet/internal/transport"
)

// testPipe creates a connected pair of net.Conn for integration tests.
func testPipe(t *testing.T) (client, server net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	connCh := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		connCh <- c
	}()

	client, err = net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	select {
	case server = <-connCh:
		t.Cleanup(func() { server.Close() })
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server connection")
	}

	return client, server
}

// ──────────────────────────────────────────────────────────────────────
// Test 1: AES-GCM tamper detection (flip one ciphertext byte)
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_AESGCMTamperDetection(t *testing.T) {
	// Derive a message key from the ratchet.
	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("integration-tamper-root-key!"))
	copy(chainKey[:], []byte("integration-tamper-chain-key"))

	rs := &ratchet.RatchetState{RootKey: rootKey, ChainKey: chainKey}
	msgKey, err := rs.Advance()
	if err != nil {
		t.Fatalf("Advance: %v", err)
	}

	// Encrypt a message with the derived key.
	plaintext := []byte("tamper detection integration test")
	ad := []byte("counter-1")
	ct, err := aead.Encrypt(msgKey, plaintext, ad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Verify normal decryption works.
	pt, err := aead.Decrypt(msgKey, ct, ad)
	if err != nil {
		t.Fatalf("Decrypt (unmodified): %v", err)
	}
	if string(pt) != string(plaintext) {
		t.Fatalf("Decrypt returned wrong plaintext: %q", pt)
	}

	// Flip one byte in the ciphertext body (past the nonce).
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	if len(tampered) > 13 {
		tampered[13] ^= 0xFF
	}

	_, err = aead.Decrypt(msgKey, tampered, ad)
	if err == nil {
		t.Fatal("Decrypt succeeded on tampered ciphertext; expected AuthenticationError")
	}

	var authErr *aead.AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T: %v", err, err)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Test 2: Root-key rotation across two DH ratchet epochs
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_RootKeyRotationTwoEpochs(t *testing.T) {
	// Initial root key.
	var rootKey ratchet.Key32
	copy(rootKey[:], []byte("integration-rotation-root-k!"))

	// ── Epoch 1 ──
	alicePriv1, _, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Alice epoch 1 keygen: %v", err)
	}
	bobPriv1, _, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Bob epoch 1 keygen: %v", err)
	}

	secret1, err := dhratchet.ComputeSharedSecret(alicePriv1, bobPriv1.PublicKey())
	if err != nil {
		t.Fatalf("Epoch 1 shared secret: %v", err)
	}

	rootKey1, err := dhratchet.DeriveRootKey(secret1, rootKey)
	if err != nil {
		t.Fatalf("Epoch 1 DeriveRootKey: %v", err)
	}

	// ── Epoch 2 ──
	alicePriv2, _, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Alice epoch 2 keygen: %v", err)
	}
	bobPriv2, _, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		t.Fatalf("Bob epoch 2 keygen: %v", err)
	}

	secret2, err := dhratchet.ComputeSharedSecret(alicePriv2, bobPriv2.PublicKey())
	if err != nil {
		t.Fatalf("Epoch 2 shared secret: %v", err)
	}

	rootKey2, err := dhratchet.DeriveRootKey(secret2, rootKey1)
	if err != nil {
		t.Fatalf("Epoch 2 DeriveRootKey: %v", err)
	}

	// All three root keys must be different.
	if rootKey == rootKey1 {
		t.Error("root key unchanged after epoch 1")
	}
	if rootKey1 == rootKey2 {
		t.Error("root key unchanged after epoch 2")
	}
	if rootKey == rootKey2 {
		t.Error("root key after epoch 2 equals initial root key")
	}

	// Verify that the epoch-2 root key produces working encryption keys.
	rs := &ratchet.RatchetState{RootKey: rootKey2, ChainKey: rootKey2}
	msgKey, err := rs.Advance()
	if err != nil {
		t.Fatalf("Advance after epoch 2: %v", err)
	}

	ct, err := aead.Encrypt(msgKey, []byte("post-rotation message"), []byte("epoch-2"))
	if err != nil {
		t.Fatalf("Encrypt after rotation: %v", err)
	}

	pt, err := aead.Decrypt(msgKey, ct, []byte("epoch-2"))
	if err != nil {
		t.Fatalf("Decrypt after rotation: %v", err)
	}

	if string(pt) != "post-rotation message" {
		t.Errorf("wrong plaintext: %q", pt)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Test 3: Transport — connection dropped mid-frame
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_TransportDroppedMidFrame(t *testing.T) {
	client, server := testPipe(t)

	// Write a valid length prefix claiming 500 bytes, but only send 10.
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, 500)
	client.Write(prefix)
	client.Write([]byte("short-data"))
	client.Close()

	_, err := transport.RecvFrame(server)
	if err == nil {
		t.Fatal("RecvFrame succeeded on truncated frame; expected error")
	}
}

// ──────────────────────────────────────────────────────────────────────
// Test 4: Replayed ciphertext rejected by associated-data counter
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_ReplayRejectedByCounter(t *testing.T) {
	// Simulate a sender with a monotonic counter as associated data.
	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("integration-replay-root-key!"))
	copy(chainKey[:], []byte("integration-replay-chain-key"))

	senderState := &ratchet.RatchetState{RootKey: rootKey, ChainKey: chainKey}
	receiverState := &ratchet.RatchetState{RootKey: rootKey, ChainKey: chainKey}

	// Sender sends message #1.
	senderKey1, _ := senderState.Advance()
	ad1 := []byte("msg-counter-1")
	ct1, err := aead.Encrypt(senderKey1, []byte("first message"), ad1)
	if err != nil {
		t.Fatalf("Encrypt msg 1: %v", err)
	}

	// Receiver decrypts message #1 normally.
	receiverKey1, _ := receiverState.Advance()
	pt1, err := aead.Decrypt(receiverKey1, ct1, ad1)
	if err != nil {
		t.Fatalf("Decrypt msg 1: %v", err)
	}
	if string(pt1) != "first message" {
		t.Fatalf("wrong plaintext: %q", pt1)
	}

	// Sender sends message #2.
	senderKey2, _ := senderState.Advance()
	ad2 := []byte("msg-counter-2")
	ct2, err := aead.Encrypt(senderKey2, []byte("second message"), ad2)
	if err != nil {
		t.Fatalf("Encrypt msg 2: %v", err)
	}

	// Receiver advances to message #2.
	receiverKey2, _ := receiverState.Advance()

	// REPLAY ATTACK: attacker replays ct1 but receiver expects counter-2.
	// Decrypting ct1 with receiverKey2 should fail (different key).
	_, err = aead.Decrypt(receiverKey2, ct1, ad1)
	if err == nil {
		t.Fatal("Replay of message #1 with key #2 succeeded; expected failure")
	}

	// Also: replaying ct1 with the correct key but wrong counter should fail.
	_, err = aead.Decrypt(receiverKey1, ct1, ad2)
	if err == nil {
		t.Fatal("Replay of message #1 with wrong counter succeeded; expected failure")
	}

	// Verify message #2 decrypts normally with correct key and counter.
	pt2, err := aead.Decrypt(receiverKey2, ct2, ad2)
	if err != nil {
		t.Fatalf("Decrypt msg 2: %v", err)
	}
	if string(pt2) != "second message" {
		t.Fatalf("wrong plaintext for msg 2: %q", pt2)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Test 5: Full pipeline — ratchet → encrypt → transport → decrypt
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_FullPipeline(t *testing.T) {
	client, server := testPipe(t)

	// Shared initial state.
	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("full-pipeline-root-key-val!!"))
	copy(chainKey[:], []byte("full-pipeline-chain-key-val!"))

	aliceState := &ratchet.RatchetState{RootKey: rootKey, ChainKey: chainKey}
	bobState := &ratchet.RatchetState{RootKey: rootKey, ChainKey: chainKey}

	messages := []string{
		"Hello Bob!",
		"Meeting at noon.",
		"Bring the USB drive.",
	}

	// Alice encrypts and sends.
	for i, msg := range messages {
		msgKey, err := aliceState.Advance()
		if err != nil {
			t.Fatalf("Alice Advance %d: %v", i, err)
		}

		ad := []byte(fmt.Sprintf("msg-%d", i))
		ct, err := aead.Encrypt(msgKey, []byte(msg), ad)
		if err != nil {
			t.Fatalf("Alice Encrypt %d: %v", i, err)
		}

		if err := transport.SendFrame(client, ct); err != nil {
			t.Fatalf("SendFrame %d: %v", i, err)
		}
	}

	// Bob receives and decrypts.
	for i, want := range messages {
		ct, err := transport.RecvFrame(server)
		if err != nil {
			t.Fatalf("RecvFrame %d: %v", i, err)
		}

		msgKey, err := bobState.Advance()
		if err != nil {
			t.Fatalf("Bob Advance %d: %v", i, err)
		}

		ad := []byte(fmt.Sprintf("msg-%d", i))
		pt, err := aead.Decrypt(msgKey, ct, ad)
		if err != nil {
			t.Fatalf("Bob Decrypt %d: %v", i, err)
		}

		if string(pt) != want {
			t.Errorf("message %d: got %q, want %q", i, pt, want)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────
// Test 6: Transport round-trip preserves ciphertext integrity
// ──────────────────────────────────────────────────────────────────────

func TestIntegration_TransportPreservesCiphertext(t *testing.T) {
	client, server := testPipe(t)

	var key ratchet.Key32
	copy(key[:], []byte("transport-integrity-test-key!"))

	plaintext := []byte("integrity check payload")
	ad := []byte("integrity-ad")

	ct, err := aead.Encrypt(key, plaintext, ad)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Send ciphertext through transport.
	if err := transport.SendFrame(client, ct); err != nil {
		t.Fatalf("SendFrame: %v", err)
	}

	// Receive on the other side.
	received, err := transport.RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame: %v", err)
	}

	// The bytes must be identical.
	if !bytes.Equal(ct, received) {
		t.Error("transport altered the ciphertext bytes")
	}

	// Decrypt must succeed.
	pt, err := aead.Decrypt(key, received, ad)
	if err != nil {
		t.Fatalf("Decrypt after transport: %v", err)
	}

	if string(pt) != string(plaintext) {
		t.Errorf("got %q, want %q", pt, plaintext)
	}
}
