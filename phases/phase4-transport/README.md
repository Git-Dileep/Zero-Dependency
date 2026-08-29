# Phase 4 — Networking / Transport Layer

**Owner:** Person B (Transport/Demo/Docs)  
**Status:** ⏳ Stub (awaiting implementation)

## What This Phase Does

Plain TCP framing so Alice, Bob, and the Attacker can run as three separate processes. Fully independent of the crypto phases — it only moves opaque byte blobs.

## Files Produced

| File | Purpose |
|------|---------|
| `internal/transport/transport.go` | `SendFrame()` and `RecvFrame()` with 4-byte length prefix |
| `internal/transport/transport_test.go` | Tests for truncated connections, frame integrity |

## Key Implementation Details

- 4-byte big-endian length prefix (`encoding/binary`) followed by raw payload
- `SendFrame` writes prefix + payload in one buffered write
- `RecvFrame` reads exactly the prefix, then exactly that many payload bytes
- Returns error on truncated/dropped connection

## Prompt

```
You are generating Go code for MiniRatchet (stdlib-only, Go 1.20+, no
third-party imports). Implement package internal/transport using exactly
this contract:

func SendFrame(conn net.Conn, payload []byte) error
func RecvFrame(conn net.Conn) ([]byte, error)

Use a 4-byte big-endian length prefix (encoding/binary) followed by the raw
payload bytes. SendFrame writes the prefix then the payload in one buffered
write. RecvFrame reads exactly the prefix, then exactly that many payload
bytes, returning an error on a truncated/dropped connection. Also generate
a minimal cmd/miniratchet main() that, given --role alice or --role bob,
either net.Listen()s or net.Dial()s on localhost:9000 and exchanges raw
frames in a goroutine read/write loop. Include a comment marking where
ciphertext from internal/aead will be plugged in later.
```
