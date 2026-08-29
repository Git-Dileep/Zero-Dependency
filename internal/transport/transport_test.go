package transport

import (
	"bytes"
	"net"
	"sync"
	"testing"
)

// TestSendRecv_Roundtrip verifies basic frame send and receive over net.Pipe.
func TestSendRecv_Roundtrip(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	payload := []byte("hello MiniRatchet transport")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := SendFrame(client, payload); err != nil {
			t.Errorf("SendFrame failed: %v", err)
		}
	}()

	received, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame failed: %v", err)
	}
	if !bytes.Equal(received, payload) {
		t.Fatalf("Payload mismatch: got %q, want %q", received, payload)
	}

	wg.Wait()
}

// TestSendRecv_MultipleFrames verifies that multiple sequential frames
// are correctly framed and deframed without bleeding into each other.
func TestSendRecv_MultipleFrames(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	payloads := [][]byte{
		[]byte("frame-one"),
		[]byte("frame-two-longer-payload"),
		[]byte("f3"),
		[]byte(""),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, p := range payloads {
			if err := SendFrame(client, p); err != nil {
				t.Errorf("SendFrame failed: %v", err)
				return
			}
		}
	}()

	for i, want := range payloads {
		got, err := RecvFrame(server)
		if err != nil {
			t.Fatalf("RecvFrame %d failed: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("Frame %d mismatch: got %q, want %q", i, got, want)
		}
	}

	wg.Wait()
}

// TestRecvFrame_DroppedConnection verifies that RecvFrame returns an error
// when the connection is closed mid-read.
func TestRecvFrame_DroppedConnection(t *testing.T) {
	client, server := net.Pipe()

	// Close server side immediately — client RecvFrame should fail.
	server.Close()

	_, err := RecvFrame(client)
	if err == nil {
		t.Fatal("Expected error when reading from a closed connection")
	}
	client.Close()
}

// TestSendRecv_EmptyPayload verifies that a zero-length payload works correctly.
func TestSendRecv_EmptyPayload(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := SendFrame(client, []byte{}); err != nil {
			t.Errorf("SendFrame empty failed: %v", err)
		}
	}()

	got, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("RecvFrame empty failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Expected empty payload, got %d bytes", len(got))
	}

	wg.Wait()
}
