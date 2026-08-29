// Package transport provides length-prefixed framing over a net.Conn for
// sending and receiving binary payloads.
//
// Contract (do not change without updating all dependent phases):
//
//	func SendFrame(conn net.Conn, payload []byte) error
//	func RecvFrame(conn net.Conn) ([]byte, error)
//
// Phase 4 — full implementation.
package transport

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

// maxFrameSize is a safety limit to prevent allocating absurd amounts of
// memory on a corrupt or malicious length prefix. 16 MiB should be more
// than enough for any ratchet message.
const maxFrameSize = 16 << 20 // 16 MiB

// SendFrame writes a length-prefixed frame containing payload to conn.
//
// Wire format: [4-byte big-endian length][payload bytes]
//
// The length prefix encodes the number of payload bytes that follow.
// Both the prefix and payload are written in a single sequence to minimise
// the number of syscalls.
//
// NOTE: the payload here is opaque bytes. In the full MiniRatchet pipeline,
// this is where ciphertext from internal/aead will be plugged in — the
// transport layer does not know or care whether the bytes are encrypted.
func SendFrame(conn net.Conn, payload []byte) error {
	// Write the 4-byte big-endian length prefix.
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, uint32(len(payload)))

	// Write prefix.
	if _, err := conn.Write(lengthBuf); err != nil {
		return fmt.Errorf("transport: write length prefix: %w", err)
	}

	// Write payload.
	if _, err := conn.Write(payload); err != nil {
		return fmt.Errorf("transport: write payload: %w", err)
	}

	return nil
}

// RecvFrame reads a length-prefixed frame from conn and returns the payload.
//
// It reads exactly 4 bytes for the length prefix, then reads exactly that
// many bytes of payload. Returns an error if the connection is dropped
// mid-frame or if the frame size exceeds the safety limit.
//
// NOTE: the returned bytes are opaque. In the full MiniRatchet pipeline,
// these would be passed to internal/aead.Decrypt — the transport layer
// does not interpret the payload.
func RecvFrame(conn net.Conn) ([]byte, error) {
	// Read exactly 4 bytes for the length prefix.
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, fmt.Errorf("transport: read length prefix: %w", err)
	}

	payloadLen := binary.BigEndian.Uint32(lengthBuf)

	// Safety check: reject absurdly large frames.
	if payloadLen > uint32(maxFrameSize) {
		return nil, fmt.Errorf("transport: frame size %d exceeds maximum %d", payloadLen, maxFrameSize)
	}

	// Read exactly payloadLen bytes.
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("transport: read payload (%d bytes): %w", payloadLen, err)
	}

	return payload, nil
}
