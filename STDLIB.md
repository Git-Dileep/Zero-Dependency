# STDLIB.md — Zero-Dependency Substitution Table

MiniRatchet uses **zero third-party imports**. Every cryptographic and networking primitive is built from Go's standard library. This table maps what you'd normally reach for to what we actually used.

## Substitution Table

| # | Normally | Instead | Phase |
|---|----------|---------|-------|
| 1 | `golang.org/x/crypto/hkdf` | `crypto/hmac` + `crypto/sha256` (HKDF extract-then-expand built from HMAC-SHA256) | Phase 3 |
| 2 | `golang.org/x/crypto/curve25519` | `crypto/ecdh` (X25519 via `ecdh.X25519()`) | Phase 3 |
| 3 | `golang.org/x/crypto/chacha20poly1305` | `crypto/aes` + `crypto/cipher` (AES-256-GCM) | Phase 2 |
| 4 | `golang.org/x/crypto/nacl/secretbox` | `crypto/aes` + `crypto/cipher` (AES-256-GCM with random nonce) | Phase 2 |
| 5 | `github.com/gorilla/websocket` | `net` (plain TCP with length-prefixed framing via `encoding/binary`) | Phase 4 |
| 6 | `google.golang.org/protobuf` | `encoding/binary` (4-byte big-endian length prefix + raw bytes) | Phase 4 |
| 7 | `github.com/charmbracelet/bubbletea` | `fmt` + `strings` (plain stdout box-drawing and Unicode formatting) | Phase 5 |
| 8 | `github.com/fatih/color` | `fmt.Printf` (Unicode symbols ✓ ✗ ⚠ 🔓 🔒 🗑️ and box-drawing chars) | Phase 5 |
| 9 | `github.com/stretchr/testify` | `testing` (stdlib `t.Fatal`, `t.Error`, `t.Fatalf` with manual assertions) | Phase 6 |
| 10 | `golang.org/x/crypto/argon2` or `golang.org/x/crypto/scrypt` | `crypto/hmac` + `crypto/sha256` (HMAC-based KDF chain for key derivation) | Phase 2 |
| 11 | `github.com/spf13/cobra` | `flag` (stdlib flag parsing with `--role` and `--demo`) | Phase 1 |
| 12 | `crypto/rand` (external CSPRNG wrappers) | `crypto/rand` (already stdlib — used directly for GCM nonce generation) | Phase 2 |

## Why This Matters

The Go standard library (as of Go 1.20+) ships production-grade implementations of:

- **X25519 ECDH** (`crypto/ecdh`) — the same curve Signal uses for key agreement
- **AES-256-GCM** (`crypto/aes` + `crypto/cipher`) — NIST-approved AEAD
- **HMAC-SHA256** (`crypto/hmac` + `crypto/sha256`) — the building block for HKDF and KDF chains
- **CSPRNG** (`crypto/rand`) — OS-backed randomness for nonces and key generation

By composing these trusted, audited primitives instead of importing third-party wrappers, MiniRatchet:

1. **Eliminates supply-chain risk** — no transitive dependencies to audit or worry about
2. **Builds reproducibly** — `go build` with an empty `require` block, every time
3. **Compiles to a static binary** — no shared libraries, no runtime dependencies

## Reproducible Build Bonus

To verify the build is reproducible (same source → same binary, byte-for-byte):

```bash
# Build twice with -trimpath to strip machine-specific paths
go build -trimpath -o miniratchet_1 ./cmd/miniratchet
go build -trimpath -o miniratchet_2 ./cmd/miniratchet

# Compare SHA-256 checksums — they must match
sha256sum miniratchet_1 miniratchet_2
```

The `-trimpath` flag removes all filesystem paths from the compiled binary, ensuring that two builds from the same source on different machines produce identical output. Go's deterministic compiler and linker handle the rest — no Bazel, no Nix, no Docker needed.

## Single-File Bonus

To consolidate MiniRatchet into a single `main.go`:

1. Copy the contents of each `internal/` package into one file, removing the `package` declarations
2. Prefix any name collisions (there are none by design — the contract enforces unique function names across packages)
3. Place everything under `package main` with a single `func main()` entrypoint

The result is a single `.go` file that can be built with `go build main.go` — no module system required, no directory structure needed. This is useful for sharing the project as a single gist or for pedagogical purposes.
