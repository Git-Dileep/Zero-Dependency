# Phase 5 — Attack, Compromise & Visual Ratchet Demo

**Owner:** Person B (Transport/Demo/Docs)  
**Status:** ✅ Complete

## What This Phase Does

The highest-value phase for judges. Implements four CLI-flag-driven demo modes and a visual key chain renderer, all using plain stdout formatting.

## Files Produced

| File | Purpose |
|------|---------|
| `internal/demo/demo.go` | All 4 demo modes + visual ratchet renderer + session wiring |
| `cmd/miniratchet/main.go` | Updated CLI with `--demo` and `--role` flags |

## Demo Modes

### 1. `steal-key`
Captures the current message key mid-session, then attempts `Decrypt()` on one earlier and one later ciphertext — both produce `AuthenticationError`.

### 2. `compromise`
Pauses the chat loop, prints "Bob's current key is compromised", resumes, then triggers a new DH ratchet epoch. Shows the session automatically re-secured.

### 3. `two-panel`
Split stdout so legitimate plaintext prints on one side and the attacker's raw ciphertext-only view prints on the other (box-drawing table).

### 4. `destroy`
Zeros out old key byte slices in memory and prints "There is nothing to recover."

### Visual Ratchet Renderer
Prints the key chain evolving as `K1 → K2 → K3 → K4`, visually marking prior keys as cleared (`████████`) once ratcheted past.

## Usage

```bash
./miniratchet --demo steal-key
./miniratchet --demo compromise
./miniratchet --demo two-panel
./miniratchet --demo destroy
./miniratchet --demo all
./miniratchet --role alice      # runs all modes
./miniratchet --role attacker   # runs steal-key + two-panel
```
