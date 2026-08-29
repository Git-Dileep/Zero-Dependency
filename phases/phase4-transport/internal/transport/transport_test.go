package transport

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestTransportSendRecv(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	payload1 := []byte("hello server")
	payload2 := []byte("hello client")

	// Send from client
	go func() {
		err := SendFrame(client, payload1)
		if err != nil {
			t.Errorf("Client SendFrame failed: %v", err)
		}

		received, err := RecvFrame(client)
		if err != nil {
			t.Errorf("Client RecvFrame failed: %v", err)
		}
		if !bytes.Equal(received, payload2) {
			t.Errorf("Client received wrong payload")
		}
	}()

	// Receive on server
	received, err := RecvFrame(server)
	if err != nil {
		t.Fatalf("Server RecvFrame failed: %v", err)
	}
	if !bytes.Equal(received, payload1) {
		t.Fatalf("Server received wrong payload")
	}

	// Send back to client
	err = SendFrame(server, payload2)
	if err != nil {
		t.Fatalf("Server SendFrame failed: %v", err)
	}

	// Wait briefly for goroutine to finish
	time.Sleep(50 * time.Millisecond)
}
