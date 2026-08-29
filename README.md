# MiniRatchet

> **A stolen key does not unlock your conversation history.**

MiniRatchet is a zero-dependency, stdlib-only Go implementation of a Signal-style double ratchet messenger, built for **Track E: Security & Crypto Utilities**. It demonstrates forward secrecy and post-compromise security using only Go's standard library — no `golang.org/x/crypto`, no third-party packages, ever.

## What It Does

MiniRatchet implements the core primitives of the [Double Ratchet Algorithm](https://signal.org/docs/specifications/doubleratchet/):

- **Symmetric ratchet** — HMAC-SHA256 KDF chain that derives a unique encryption key per message and immediately destroys old key material
- **AEAD encryption** — AES-256-GCM authenticated encryption with random nonces and associated-data binding
- **DH ratchet** — X25519 ephemeral key exchange that rotates the root key, providing self-healing after compromise
- **Length-prefixed transport** — Simple TCP framing for multi-process Alice/Bob/Attacker communication

## Build

```bash
go build -o miniratchet ./cmd/miniratchet
```

Single command. No `go mod download`. No dependencies to fetch. That's the point.

## Run the Demo

### Full demo (all modes)

```bash
./miniratchet --demo all
```

### Individual demo modes

```bash
# Steal a message key — prove it can't decrypt earlier or later messages
./miniratchet --demo steal-key

# Compromise a chain key — watch the DH ratchet automatically re-secure
./miniratchet --demo compromise

# Side-by-side: legitimate plaintext vs. attacker's ciphertext view
./miniratchet --demo two-panel

# Zero all key material in memory — "There is nothing to recover."
./miniratchet --demo destroy
```

### Role-based sessions

```bash
# Terminal 1: start Bob (responder)
./miniratchet --role bob

# Terminal 2: start Alice (initiator) — runs the full demo sequence
./miniratchet --role alice

# Terminal 3: start an attacker — shows steal-key + two-panel views
./miniratchet --role attacker
```

Use `--addr` to change the default address (default: `localhost:9000`).

## Demo Walkthrough (5-minute video script)

| Step | Mode | What the audience sees |
|------|------|----------------------|
| 1 | `steal-key` | Attacker captures one message key mid-session, tries to decrypt an earlier and a later message — both fail with `AuthenticationError`. |
| 2 | `compromise` | Bob's chain key is leaked. A new DH epoch triggers automatically. The stolen key is now useless — new root key differs completely. |
| 3 | `two-panel` | Split-screen: left shows readable plaintext, right shows opaque ciphertext hex. Same data, different perspectives. |
| 4 | `destroy` | All keys zeroed in memory. Hex dump confirms `0000000000000000`. "There is nothing to recover." |

## Threat Model

### What an attacker with one stolen message key **can** do:

- Decrypt the **single message** that was encrypted with that specific key
- Confirm the existence of a conversation at that point in time

### What an attacker with one stolen message key **cannot** do:

- **Decrypt any earlier messages** — the KDF chain is one-way; you cannot reverse HMAC-SHA256 to recover a parent key
- **Decrypt any later messages** — the chain key advances irreversibly after each message; the stolen key is never used again
- **Derive the root key** — message keys are derived from the chain key via a separate HMAC path (`0x01`), not from the root key directly
- **Prevent session recovery** — the DH ratchet rotates the root key using fresh X25519 ephemeral keys, so even a compromised chain key becomes useless after the next epoch

### What an attacker with a stolen **chain key** can do:

- Derive all **future** message keys in the current epoch (until the next DH ratchet step)

### What a DH ratchet epoch rotation guarantees:

- The attacker's stolen chain key is **completely disconnected** from the new root key
- Recovery is **automatic** — no user action required, no "reset session" button
- This is the **self-healing** property: compromise is temporary, not permanent

### What MiniRatchet does **not** protect against:

- **Active MITM during initial key exchange** — there is no identity verification or certificate pinning (out of scope for this hackathon)
- **Endpoint compromise** — if the attacker owns the device (full memory read), all bets are off; the ratchet protects the *wire*, not the *endpoint*
- **Side-channel attacks** — no constant-time guarantees beyond what Go's `crypto/*` packages provide
- **Message ordering attacks** — out-of-order delivery is not handled by the current symmetric ratchet (would require a message index + skipped-key cache)

## Architecture

```
cmd/miniratchet/         CLI entrypoint (--role, --demo flags)
internal/
  ratchet/               Symmetric KDF ratchet (HMAC-SHA256 chain)
  aead/                  AES-256-GCM encrypt/decrypt
  dhratchet/             X25519 DH ratchet + HKDF-style key derivation
  transport/             Length-prefixed TCP framing
  demo/                  steal-key, compromise, two-panel, destroy modes
```

## License

Hackathon project — MIT.
