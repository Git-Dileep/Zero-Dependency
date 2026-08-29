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
	"errors"
	"io"
	"net"
)

// MaxPayloadSize is the maximum allowed payload size (10 MB) to prevent
// malformed length prefixes from causing out-of-memory allocations.
const MaxPayloadSize = 10 * 1024 * 1024

// SendFrame writes a length-prefixed frame containing payload to conn.
// Format: 4-byte big-endian length prefix followed by the raw payload bytes.
// The prefix and payload are written in a single buffered write to minimise
// syscalls and partial-write edge cases.
func SendFrame(conn net.Conn, payload []byte) error {
	length := uint32(len(payload))

	// Build a single buffer: 4-byte length prefix + payload.
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	copy(buf[4:], payload)

	_, err := conn.Write(buf)
	return err
}

// RecvFrame reads a length-prefixed frame from conn and returns the payload.
// It reads exactly the 4-byte prefix, then exactly that many payload bytes,
// returning an error on a truncated or dropped connection.
func RecvFrame(conn net.Conn) ([]byte, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(conn, lengthBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])

	if length > MaxPayloadSize {
		return nil, errors.New("transport: payload too large")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
