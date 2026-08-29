# Phase 1 — Shared Architecture & Repo Skeleton

**Owner:** Both (contract lock-in)  
**Status:** ✅ Complete

## What This Phase Does

Locks the module layout and the shared contract. This is fired first on hackathon morning to generate the repo scaffold before any other phase starts.

## Files Produced

| File | Purpose |
|------|---------|
| `go.mod` | Module definition with empty require block |
| `cmd/miniratchet/main.go` | CLI entrypoint with `--role` flag |
| `internal/ratchet/ratchet.go` | Package stub with `RatchetState` and `Advance()` |
| `internal/aead/aead.go` | Package stub with `Encrypt()` and `Decrypt()` |
| `internal/dhratchet/dhratchet.go` | Package stub with `NewEpochKeyPair()` and `DeriveRootKey()` |
| `internal/transport/transport.go` | Package stub with `SendFrame()` and `RecvFrame()` |
| `internal/demo/demo.go` | Package stub with `RunAlice()`, `RunBob()`, `RunAttacker()` |

## Shared Contract

```go
type Key32 = [32]byte

// internal/ratchet
type RatchetState struct {
    RootKey  Key32
    ChainKey Key32
    Epoch    uint32
}
func (rs *RatchetState) Advance() (messageKey Key32, err error)

// internal/aead
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error)
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error)

// internal/dhratchet
func NewEpochKeyPair() (priv *ecdh.PrivateKey, pub *ecdh.PublicKey, err error)
func DeriveRootKey(sharedSecret []byte, oldRootKey Key32) (newRootKey Key32, err error)

// internal/transport
func SendFrame(conn net.Conn, payload []byte) error
func RecvFrame(conn net.Conn) ([]byte, error)
```
