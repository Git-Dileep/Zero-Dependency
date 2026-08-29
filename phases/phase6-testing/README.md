# Phase 6 — Testing & Hardening

**Owner:** Person A (Crypto/Ratchet)  
**Status:** ⏳ Awaiting Phases 2–5

## What This Phase Does

Fired after Phases 2–5 exist, to close gaps the individual phase prompts wouldn't catch on their own (cross-module edge cases).

## Tests to Write

| Test | Package | What It Validates |
|------|---------|-------------------|
| AES-GCM tamper detection | `aead` | Flip one ciphertext byte → `Decrypt` must fail |
| Root-key rotation | `dhratchet` | Correct key derivation across two DH ratchet epochs |
| Transport out-of-order | `transport` | Behavior when frames arrive out of sequence |
| Transport mid-frame drop | `transport` | Error handling when connection drops mid-frame |
| Replay rejection | `aead` | Replayed ciphertext rejected by associated-data counter check |

## Prompt

```
You are generating Go tests for MiniRatchet (stdlib-only, Go 1.20+, using
package "testing" only). Assume internal/ratchet, internal/aead,
internal/dhratchet, and internal/transport already exist matching this
contract:

type Key32 = [32]byte
func (rs *RatchetState) Advance() (messageKey Key32, err error)
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error)
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error)
func SendFrame(conn net.Conn, payload []byte) error
func RecvFrame(conn net.Conn) ([]byte, error)

Write tests for: AES-GCM tamper detection (flip one ciphertext byte,
expect Decrypt to fail), correct root-key rotation across two dhratchet
epochs, transport behavior on an out-of-order frame and on a connection
dropped mid-frame, and a replayed ciphertext being rejected by the
associated-data counter check.
```
