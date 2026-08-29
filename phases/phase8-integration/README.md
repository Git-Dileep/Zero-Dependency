# Phase 8 — Integration, Rehearsal & Submission

**Owner:** Both (together)  
**Status:** ⏳ Awaiting all phases

## What This Phase Does

This is a **human phase**, not a chatbot phase. Both team members together, since this is the highest-risk point for silent contract drift between AI-generated modules.

## Checklist

- [ ] **Compile all packages together** — `go build ./...` — fix any signature mismatches against the shared contract
- [ ] **Run all tests** — `go test ./...` — confirm every test from Phase 6 passes
- [ ] **Run the full demo sequence live:**
  - [ ] `steal-key` → prove stolen key can't decrypt earlier/later messages
  - [ ] `compromise` → show DH ratchet auto-recovery
  - [ ] `two-panel` → side-by-side plaintext vs. ciphertext
  - [ ] `destroy` → zero all keys, "nothing to recover"
  - [ ] Time it against the 5-minute video script
- [ ] **Clean-clone test** — confirm the repo builds from a fresh `git clone` on a machine that isn't either team member's own
- [ ] **Record the demo video**
- [ ] **Verify the repo is public**
- [ ] **Submit**

## Quick Commands

```bash
# Full build
go build ./...

# Full test suite
go test ./...

# Run the complete demo
go run ./cmd/miniratchet --demo all

# Reproducible build check
go build -trimpath -o miniratchet_1 ./cmd/miniratchet
go build -trimpath -o miniratchet_2 ./cmd/miniratchet
sha256sum miniratchet_1 miniratchet_2
```
