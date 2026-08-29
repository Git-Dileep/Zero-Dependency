# Phase 2 — Core Symmetric Ratchet + AEAD Encryption

**Owner:** Person A (Crypto/Ratchet)  
**Status:** ⏳ Stub (awaiting Person A)

## What This Phase Does

Implements the hash ratchet and the AES-GCM encrypt/decrypt layer — the Must-have safety net.

## Files Produced

| File | Purpose |
|------|---------|
| `internal/ratchet/ratchet.go` | Full `Advance()` — HMAC-SHA256 KDF chain step |
| `internal/aead/aead.go` | Full `Encrypt()`/`Decrypt()` — AES-256-GCM with random 12-byte nonce |
| `internal/ratchet/ratchet_test.go` | Proves later chain key cannot reproduce earlier message key |
| `internal/aead/aead_test.go` | Proves tamper detection via `AuthenticationError` |

## Key Implementation Details

- `Advance()` derives `nextChainKey = HMAC-SHA256(ChainKey, 0x02)` and `messageKey = HMAC-SHA256(ChainKey, 0x01)`
- Old ChainKey bytes are overwritten before replacement (forward secrecy)
- `Encrypt` prepends a random 12-byte nonce to the AES-256-GCM ciphertext
- `Decrypt` returns a distinct `AuthenticationError` type on tamper/wrong-key

## Prompt

```
You are generating Go code for MiniRatchet (stdlib-only, no third-party
imports, Go 1.20+). Implement package internal/ratchet and internal/aead
using exactly this contract:

type Key32 = [32]byte
type RatchetState struct { RootKey, ChainKey Key32; Epoch uint32 }
func (rs *RatchetState) Advance() (messageKey Key32, err error)
func Encrypt(key Key32, plaintext, associatedData []byte) (ciphertext []byte, err error)
func Decrypt(key Key32, ciphertext, associatedData []byte) (plaintext []byte, err error)

Advance() derives nextChainKey = HMAC-SHA256(ChainKey, 0x02) and
messageKey = HMAC-SHA256(ChainKey, 0x01) using crypto/hmac + crypto/sha256,
then overwrites the old ChainKey bytes before replacing it. Encrypt/Decrypt
use AES-256-GCM (crypto/aes + crypto/cipher) with a random 12-byte nonce
from crypto/rand prepended to ciphertext. Decrypt must return a distinct
AuthenticationError type on tamper/wrong-key. Include unit tests proving a
later chain key cannot reproduce an earlier message key.
```
