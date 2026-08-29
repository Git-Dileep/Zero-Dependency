# Phase 7 — README, STDLIB.md & Bonus Challenges

**Owner:** Person B (Transport/Demo/Docs)  
**Status:** ✅ Complete

## What This Phase Does

Documentation for judges: project pitch, demo instructions, threat model, and the stdlib substitution table proving zero-dependency compliance.

## Files Produced

| File | Purpose |
|------|---------|
| `README.md` | Project pitch, build, demo walkthrough, threat model, architecture |
| `STDLIB.md` | 12-row substitution table, reproducible-build bonus, single-file bonus |

## README Highlights

- **Pitch:** "A stolen key does not unlock your conversation history"
- **Build:** Single `go build` command, no dependencies
- **Demo:** Full walkthrough table for 5-minute video
- **Threat Model:** What attackers can/cannot do with stolen message keys vs. chain keys
- **Architecture:** Package-level diagram

## STDLIB.md Highlights

- 12 substitutions (x/crypto/hkdf → hmac+sha256, gorilla/websocket → net, bubbletea → fmt, etc.)
- Reproducible build: `go build -trimpath` + `sha256sum` comparison
- Single-file bonus: consolidation instructions
