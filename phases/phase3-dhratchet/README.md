# Phase 3 — Diffie-Hellman Ratchet / Epoch Rotation

**Owner:** Person A (Crypto/Ratchet)  
**Status:** ✅ Complete

## What This Phase Does

Implements the DH ratchet that provides the "self-healing after compromise" property. Uses X25519 for ephemeral key agreement and HKDF-style extract-then-expand built from HMAC-SHA256.

## Files Produced

| File | Purpose |
|------|---------|
| `internal/dhratchet/dhratchet.go` | `NewEpochKeyPair()`, `DeriveRootKey()`, `ComputeSharedSecret()` |
| `internal/dhratchet/dhratchet_test.go` | 6 tests covering ECDH symmetry, key derivation, determinism, nil rejection |

## Key Implementation Details

- `NewEpochKeyPair()` uses `crypto/ecdh` X25519 curve
- `DeriveRootKey()` implements HKDF extract-then-expand from HMAC-SHA256 (RFC 5869):
  - Extract: `PRK = HMAC-SHA256(salt=oldRootKey, IKM=sharedSecret)`
  - Expand: `OKM = HMAC-SHA256(PRK, "MiniRatchet-RootKey" || 0x01)`
- `ComputeSharedSecret()` zeros the private key bytes immediately after ECDH
- This is composing trusted primitives, NOT inventing a new cryptographic algorithm
