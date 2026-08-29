package transport

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// testPipe creates a connected pair of net.Conn for testing.
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

// TestSendRecvFrame_Roundtrip verifies that a payload survives a send/recv cycle.
func TestSendRecvFrame_Roundtrip(t *testing.T) {
	client, server := testPipe(t)

	payload := []byte("Hello, MiniRatchet transport layer!")

	// Send from client.
	if err := SendFrame(client, payload); err != nil {
		t.Fatalf("SendFrame error: %v", err)
	}

	// Receive on server.
	got, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame error: %v", err)
	}

	if !bytes.Equal(got, payload) {
		t.Errorf("RecvFrame = %q, want %q", got, payload)
	}
}

// TestSendRecvFrame_MultipleFrames verifies that multiple frames can be sent
// and received in sequence on the same connection.
func TestSendRecvFrame_MultipleFrames(t *testing.T) {
	client, server := testPipe(t)

	messages := []string{
		"first frame",
		"second frame with more data",
		"",
		"fourth frame after empty",
	}

	// Send all frames.
	for _, msg := range messages {
		if err := SendFrame(client, []byte(msg)); err != nil {
			t.Fatalf("SendFrame(%q) error: %v", msg, err)
		}
	}

	// Receive all frames.
	for _, want := range messages {
		got, err := RecvFrame(server)
		if err != nil {
			t.Fatalf("RecvFrame error: %v", err)
		}
		if string(got) != want {
			t.Errorf("RecvFrame = %q, want %q", got, want)
		}
	}
}

// TestSendRecvFrame_EmptyPayload verifies that an empty payload works correctly.
func TestSendRecvFrame_EmptyPayload(t *testing.T) {
	client, server := testPipe(t)

	if err := SendFrame(client, []byte{}); err != nil {
		t.Fatalf("SendFrame empty error: %v", err)
	}

	got, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("RecvFrame returned %d bytes for empty payload", len(got))
	}
}

// TestRecvFrame_ConnectionDroppedMidPrefix verifies that RecvFrame returns
// an error if the connection is closed before the full length prefix is read.
func TestRecvFrame_ConnectionDroppedMidPrefix(t *testing.T) {
	client, server := testPipe(t)

	// Write only 2 bytes of a 4-byte prefix, then close.
	client.Write([]byte{0x00, 0x05})
	client.Close()

	_, err := RecvFrame(server)
	if err == nil {
		t.Fatal("RecvFrame succeeded on truncated prefix; expected error")
	}
}

// TestRecvFrame_ConnectionDroppedMidPayload verifies that RecvFrame returns
// an error if the connection is closed before the full payload is read.
func TestRecvFrame_ConnectionDroppedMidPayload(t *testing.T) {
	client, server := testPipe(t)

	// Write a length prefix indicating 100 bytes, but only send 10.
	prefix := make([]byte, 4)
	binary.BigEndian.PutUint32(prefix, 100)
	client.Write(prefix)
	client.Write([]byte("only10byte"))
	client.Close()

	_, err := RecvFrame(server)
	if err == nil {
		t.Fatal("RecvFrame succeeded on truncated payload; expected error")
	}
}

// TestSendRecvFrame_LargePayload verifies that large payloads are handled correctly.
func TestSendRecvFrame_LargePayload(t *testing.T) {
	client, server := testPipe(t)

	// Create a 64KB payload.
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	if err := SendFrame(client, payload); err != nil {
		t.Fatalf("SendFrame large payload error: %v", err)
	}

	got, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame error: %v", err)
	}

	if !bytes.Equal(got, payload) {
		t.Error("large payload round-trip mismatch")
	}
}
