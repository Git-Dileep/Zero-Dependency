// Package transport provides length-prefixed framing over a net.Conn for
// sending and receiving binary payloads.
//
// Contract (do not change without updating all dependent phases):
//
//	func SendFrame(conn net.Conn, payload []byte) error
//	func RecvFrame(conn net.Conn) ([]byte, error)
//
// Phase 4 scaffold — implementation will be filled in by Phase 4.
package transport

import "net"

// SendFrame writes a length-prefixed frame containing payload to conn.
//
// TODO(phase4): implement length-prefixed framing.
func SendFrame(conn net.Conn, payload []byte) error {
	return nil // placeholder
}

// RecvFrame reads a length-prefixed frame from conn and returns the payload.
//
// TODO(phase4): implement length-prefixed framing.
func RecvFrame(conn net.Conn) ([]byte, error) {
	return nil, nil // placeholder
}
