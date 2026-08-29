package transport

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

// SendFrame writes a 4-byte big-endian length prefix followed by the payload.
func SendFrame(conn net.Conn, payload []byte) error {
	length := uint32(len(payload))
	
	// Create a single buffer for length prefix + payload to minimize writes.
	// This ensures we do one buffered write as required by the spec.
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	copy(buf[4:], payload)

	_, err := conn.Write(buf)
	return err
}

// RecvFrame reads exactly the 4-byte length prefix, then exactly that many bytes.
func RecvFrame(conn net.Conn) ([]byte, error) {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(conn, lengthBuf[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])
	
	// Optional: add a sane upper bound check to prevent OOM
	if length > 1024*1024*10 { // 10MB max
		return nil, errors.New("payload too large")
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
