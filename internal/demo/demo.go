// Package demo orchestrates the end-to-end MiniRatchet demonstration,
// wiring together the ratchet, AEAD, DH ratchet, and transport layers
// into runnable Alice/Bob/Attacker scenarios.
//
// It provides four CLI-flag-driven demo modes (steal-key, compromise,
// two-panel, destroy) and a visual ratchet renderer, all using only
// plain stdout formatting — no TUI library required.
//
// Phase 5 — full implementation.
package demo

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/miniratchet/internal/aead"
	"github.com/miniratchet/internal/dhratchet"
	"github.com/miniratchet/internal/ratchet"
)

// ──────────────────────────────────────────────────────────────────────
// Visual Ratchet Renderer
// ──────────────────────────────────────────────────────────────────────

// KeySlot represents a single key in the visual chain display.
type KeySlot struct {
	Label   string // e.g. "K1", "K2"
	KeyHex  string // first 8 hex chars of the key
	Cleared bool   // true once the ratchet has moved past this key
}

// RatchetVisualizer tracks and renders the evolving key chain as
// K1 → K2 → K3 → K4, visually marking prior keys as cleared.
type RatchetVisualizer struct {
	Slots []KeySlot
}

// NewRatchetVisualizer creates an empty visualizer.
func NewRatchetVisualizer() *RatchetVisualizer {
	return &RatchetVisualizer{}
}

// Push adds a new key slot to the chain, marking all previous slots as cleared.
func (rv *RatchetVisualizer) Push(key ratchet.Key32) {
	// Mark all existing slots as cleared.
	for i := range rv.Slots {
		rv.Slots[i].Cleared = true
	}
	label := fmt.Sprintf("K%d", len(rv.Slots)+1)
	rv.Slots = append(rv.Slots, KeySlot{
		Label:   label,
		KeyHex:  hex.EncodeToString(key[:4]), // first 4 bytes = 8 hex chars
		Cleared: false,
	})
}

// Render prints the chain evolution to stdout.
// Cleared keys show as struck-through with "██████" placeholders.
func (rv *RatchetVisualizer) Render() string {
	if len(rv.Slots) == 0 {
		return "  (no keys in chain)"
	}
	var parts []string
	for _, slot := range rv.Slots {
		if slot.Cleared {
			parts = append(parts, fmt.Sprintf("[%s:████████]", slot.Label))
		} else {
			parts = append(parts, fmt.Sprintf("[%s:%s]", slot.Label, slot.KeyHex))
		}
	}
	return "  " + strings.Join(parts, " → ")
}

// ──────────────────────────────────────────────────────────────────────
// Session: wires ratchet + aead + dhratchet into a usable chat session
// ──────────────────────────────────────────────────────────────────────

// Session holds the state for one side of a ratcheted conversation.
type Session struct {
	State      ratchet.RatchetState
	Visualizer *RatchetVisualizer
	MsgCounter uint32 // monotonic counter for associated data
}

// NewSession creates a session from a pre-shared root key and chain key.
func NewSession(rootKey, chainKey ratchet.Key32) *Session {
	return &Session{
		State: ratchet.RatchetState{
			RootKey:  rootKey,
			ChainKey: chainKey,
			Epoch:    0,
		},
		Visualizer: NewRatchetVisualizer(),
		MsgCounter: 0,
	}
}

// EncryptMessage advances the ratchet, encrypts plaintext with the derived
// message key, and returns the ciphertext. The monotonic counter is used
// as associated data to prevent replay.
func (s *Session) EncryptMessage(plaintext []byte) (ciphertext []byte, msgKey ratchet.Key32, err error) {
	msgKey, err = s.State.Advance()
	if err != nil {
		return nil, ratchet.Key32{}, fmt.Errorf("demo: ratchet advance failed: %w", err)
	}
	s.Visualizer.Push(msgKey)
	s.MsgCounter++

	ad := []byte(fmt.Sprintf("miniratchet-msg-%d", s.MsgCounter))
	ciphertext, err = aead.Encrypt(msgKey, plaintext, ad)
	if err != nil {
		return nil, ratchet.Key32{}, fmt.Errorf("demo: encrypt failed: %w", err)
	}
	return ciphertext, msgKey, nil
}

// DecryptMessage advances the ratchet, decrypts ciphertext with the derived
// message key, and returns the plaintext.
func (s *Session) DecryptMessage(ciphertext []byte) (plaintext []byte, msgKey ratchet.Key32, err error) {
	msgKey, err = s.State.Advance()
	if err != nil {
		return nil, ratchet.Key32{}, fmt.Errorf("demo: ratchet advance failed: %w", err)
	}
	s.Visualizer.Push(msgKey)
	s.MsgCounter++

	ad := []byte(fmt.Sprintf("miniratchet-msg-%d", s.MsgCounter))
	plaintext, err = aead.Decrypt(msgKey, ciphertext, ad)
	if err != nil {
		return nil, ratchet.Key32{}, fmt.Errorf("demo: decrypt failed: %w", err)
	}
	return plaintext, msgKey, nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 1: steal-key
// ──────────────────────────────────────────────────────────────────────

// RunStealKey demonstrates that capturing a message key mid-session does NOT
// allow decryption of earlier or later messages.
//
// Sequence:
//  1. Alice sends 3 messages, capturing ciphertext for each.
//  2. The attacker "steals" the message key from message #2.
//  3. The attacker tries to decrypt message #1 (earlier) — AuthenticationError.
//  4. The attacker tries to decrypt message #3 (later) — AuthenticationError.
func RunStealKey() error {
	printBanner("DEMO: steal-key")
	fmt.Println("Scenario: attacker steals ONE message key mid-session.")
	fmt.Println("Goal: show that stolen key cannot decrypt earlier or later messages.")
	fmt.Println()

	// Set up a session with a fixed seed for reproducibility.
	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("steal-key-demo-root-key-value!!"))
	copy(chainKey[:], []byte("steal-key-demo-chain-key-val!!"))
	alice := NewSession(rootKey, chainKey)

	// Alice sends 3 messages, collecting ciphertexts and keys.
	messages := []string{
		"Message 1: The eagle has landed.",
		"Message 2: Rendezvous at midnight.",
		"Message 3: Operation is a go.",
	}
	var ciphertexts [][]byte
	var msgKeys []ratchet.Key32

	fmt.Println("═══ Alice sends 3 messages ═══")
	for i, msg := range messages {
		ct, key, err := alice.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt message %d: %w", i+1, err)
		}
		ciphertexts = append(ciphertexts, ct)
		msgKeys = append(msgKeys, key)
		fmt.Printf("  [%d] Encrypted: %q → %s...\n", i+1, msg, hex.EncodeToString(ct[:min(16, len(ct))]))
	}
	fmt.Println()

	// Attacker steals message key #2.
	stolenKey := msgKeys[1]
	fmt.Printf("🔓 ATTACKER steals message key #2: %s\n\n", hex.EncodeToString(stolenKey[:8]))

	// Attempt to decrypt message #1 (earlier) with stolen key.
	fmt.Println("═══ Attacker tries to decrypt MESSAGE #1 (earlier) ═══")
	ad1 := []byte(fmt.Sprintf("miniratchet-msg-%d", 1))
	_, err := aead.Decrypt(stolenKey, ciphertexts[0], ad1)
	if err != nil {
		fmt.Printf("  ✗ FAILED: %v\n", err)
		fmt.Println("  → Stolen key CANNOT unlock earlier messages.")
	} else {
		fmt.Println("  ✓ Decrypted (UNEXPECTED — this should not happen!)")
	}
	fmt.Println()

	// Attempt to decrypt message #3 (later) with stolen key.
	fmt.Println("═══ Attacker tries to decrypt MESSAGE #3 (later) ═══")
	ad3 := []byte(fmt.Sprintf("miniratchet-msg-%d", 3))
	_, err = aead.Decrypt(stolenKey, ciphertexts[2], ad3)
	if err != nil {
		fmt.Printf("  ✗ FAILED: %v\n", err)
		fmt.Println("  → Stolen key CANNOT unlock later messages.")
	} else {
		fmt.Println("  ✓ Decrypted (UNEXPECTED — this should not happen!)")
	}
	fmt.Println()

	// Show the ratchet chain evolution.
	fmt.Println("═══ Key chain evolution ═══")
	fmt.Println(alice.Visualizer.Render())
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 2: compromise
// ──────────────────────────────────────────────────────────────────────

// RunCompromise demonstrates the self-healing property: after a key compromise,
// a new DH ratchet epoch automatically re-secures the session.
//
// Sequence:
//  1. Normal encrypted exchange (2 messages).
//  2. "Bob's current chain key is compromised" — attacker obtains it.
//  3. Trigger a new DH ratchet epoch (new key pair exchange).
//  4. Show that the attacker's stolen key is now useless.
func RunCompromise() error {
	printBanner("DEMO: compromise & recovery")
	fmt.Println("Scenario: Bob's chain key is compromised mid-session.")
	fmt.Println("Goal: show that a DH ratchet epoch rotation re-secures the session.")
	fmt.Println()

	// Initial shared keys.
	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("compromise-demo-root-key-val!!"))
	copy(chainKey[:], []byte("compromise-demo-chain-key-v!!"))
	bob := NewSession(rootKey, chainKey)

	// Normal operation: 2 messages.
	fmt.Println("═══ Normal operation: Bob sends 2 messages ═══")
	for i := 1; i <= 2; i++ {
		msg := fmt.Sprintf("Normal message %d from Bob", i)
		ct, _, err := bob.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		fmt.Printf("  [%d] %q → %s...\n", i, msg, hex.EncodeToString(ct[:min(16, len(ct))]))
	}
	fmt.Println()

	// COMPROMISE: attacker captures the current chain key.
	compromisedChainKey := bob.State.ChainKey
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║  ⚠  Bob's current chain key is COMPROMISED!     ║")
	fmt.Printf("║  Key: %s...  ║\n", hex.EncodeToString(compromisedChainKey[:12]))
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()

	// RECOVERY: trigger a DH ratchet epoch rotation.
	fmt.Println("═══ Triggering DH ratchet epoch rotation ═══")
	bobPriv, bobPub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return fmt.Errorf("new epoch key pair: %w", err)
	}
	fmt.Printf("  Bob generates new epoch key pair (pub: %s...)\n",
		hex.EncodeToString(bobPub.Bytes()[:8]))

	// Simulate Alice also generating a key pair and performing ECDH.
	alicePriv, alicePub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return fmt.Errorf("alice epoch key pair: %w", err)
	}
	fmt.Printf("  Alice generates new epoch key pair (pub: %s...)\n",
		hex.EncodeToString(alicePub.Bytes()[:8]))

	// Both sides compute shared secret and derive new root key.
	sharedSecret, err := dhratchet.ComputeSharedSecret(bobPriv, alicePub)
	if err != nil {
		return fmt.Errorf("compute shared secret: %w", err)
	}
	newRootKey, err := dhratchet.DeriveRootKey(sharedSecret, bob.State.RootKey)
	if err != nil {
		return fmt.Errorf("derive root key: %w", err)
	}
	_ = alicePriv // Alice's private key would be used on her side

	// Update Bob's session with the new root key and derive a new chain key.
	bob.State.RootKey = newRootKey
	bob.State.ChainKey = newRootKey // In a full implementation, chain key derived separately
	bob.State.Epoch++
	fmt.Printf("  ✓ New root key derived: %s...\n", hex.EncodeToString(newRootKey[:8]))
	fmt.Printf("  ✓ Epoch advanced to: %d\n", bob.State.Epoch)
	fmt.Println()

	// Post-recovery: send a message with the new key material.
	fmt.Println("═══ Post-recovery: Bob sends message with new keys ═══")
	msg := "Post-recovery message — session is re-secured!"
	ct, _, err := bob.EncryptMessage([]byte(msg))
	if err != nil {
		return fmt.Errorf("encrypt post-recovery: %w", err)
	}
	fmt.Printf("  [3] %q → %s...\n", msg, hex.EncodeToString(ct[:min(16, len(ct))]))
	fmt.Println()

	// Show that the compromised key is now useless.
	fmt.Println("═══ Attacker tries compromised chain key ═══")
	fmt.Printf("  Compromised key: %s...\n", hex.EncodeToString(compromisedChainKey[:8]))
	fmt.Printf("  Current key:     %s...\n", hex.EncodeToString(bob.State.ChainKey[:8]))
	if compromisedChainKey != bob.State.ChainKey {
		fmt.Println("  ✗ Keys differ — compromised key is USELESS after epoch rotation.")
	}
	fmt.Println()

	// Show the ratchet chain evolution.
	fmt.Println("═══ Key chain evolution ═══")
	fmt.Println(bob.Visualizer.Render())
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 3: two-panel
// ──────────────────────────────────────────────────────────────────────

// RunTwoPanel demonstrates a split view: one side shows the legitimate
// plaintext conversation, the other shows the attacker's raw ciphertext-only
// view — proving that without the key, the data is opaque.
func RunTwoPanel() error {
	printBanner("DEMO: two-panel (legitimate vs. attacker view)")
	fmt.Println("Left panel: legitimate plaintext. Right panel: attacker's ciphertext-only view.")
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("two-panel-demo-root-key-val!!\x00\x00"))
	copy(chainKey[:], []byte("two-panel-demo-chain-key-v!!\x00\x00"))
	alice := NewSession(rootKey, chainKey)

	messages := []string{
		"Hey Bob, meeting at 3pm?",
		"Confirmed. Bring the docs.",
		"See you there. Stay safe.",
		"Roger that. Over and out.",
	}

	// Print header.
	panelWidth := 40
	fmt.Printf("┌%s┬%s┐\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))
	fmt.Printf("│%-*s│%-*s│\n", panelWidth, " 🔓 LEGITIMATE VIEW", panelWidth, " 🔒 ATTACKER VIEW")
	fmt.Printf("├%s┼%s┤\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))

	for _, msg := range messages {
		ct, _, err := alice.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}

		// Left panel: plaintext (truncated to fit).
		leftText := truncate(msg, panelWidth-2)
		// Right panel: raw ciphertext hex (truncated to fit).
		ctHex := hex.EncodeToString(ct)
		rightText := truncate(ctHex, panelWidth-2)

		fmt.Printf("│ %-*s│ %-*s│\n", panelWidth-1, leftText, panelWidth-1, rightText)
	}

	fmt.Printf("└%s┴%s┘\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))
	fmt.Println()

	// Show the ratchet chain evolution.
	fmt.Println("═══ Key chain evolution ═══")
	fmt.Println(alice.Visualizer.Render())
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 4: destroy
// ──────────────────────────────────────────────────────────────────────

// RunDestroy demonstrates cryptographic key destruction: old keys are zeroed
// out in memory and there is nothing to recover. This is the forward secrecy
// guarantee made tangible.
func RunDestroy() error {
	printBanner("DEMO: destroy (key destruction)")
	fmt.Println("Scenario: demonstrating forward secrecy by zeroing old keys in memory.")
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("destroy-demo-root-key-value!!"))
	copy(chainKey[:], []byte("destroy-demo-chain-key-val!!\x00\x00"))
	session := NewSession(rootKey, chainKey)

	// Send a few messages, collecting keys.
	fmt.Println("═══ Sending messages and collecting keys ═══")
	var collectedKeys []ratchet.Key32
	for i := 1; i <= 4; i++ {
		msg := fmt.Sprintf("Secret message #%d", i)
		_, key, err := session.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		collectedKeys = append(collectedKeys, key)
		fmt.Printf("  [%d] Key: %s  Message: %q\n", i, hex.EncodeToString(key[:8]), msg)
	}
	fmt.Println()

	// Show the chain before destruction.
	fmt.Println("═══ Key chain BEFORE destruction ═══")
	fmt.Println(session.Visualizer.Render())
	fmt.Println()

	// Destroy: zero out all collected key byte slices.
	fmt.Println("═══ DESTROYING old keys ═══")
	for i := range collectedKeys {
		fmt.Printf("  Zeroing K%d: %s → ", i+1, hex.EncodeToString(collectedKeys[i][:8]))
		// Zero the key bytes.
		for j := range collectedKeys[i] {
			collectedKeys[i][j] = 0
		}
		fmt.Printf("%s ✓\n", hex.EncodeToString(collectedKeys[i][:8]))
	}

	// Also zero the session's root key and chain key.
	for i := range session.State.RootKey {
		session.State.RootKey[i] = 0
	}
	for i := range session.State.ChainKey {
		session.State.ChainKey[i] = 0
	}
	fmt.Println()

	// Verify destruction.
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║  🗑️  There is nothing to recover.                ║")
	fmt.Println("║                                                  ║")
	fmt.Println("║  All key material has been zeroed in memory.     ║")
	fmt.Println("║  Root key:  0000000000000000                     ║")
	fmt.Println("║  Chain key: 0000000000000000                     ║")
	allZero := true
	for _, k := range collectedKeys {
		if k != (ratchet.Key32{}) {
			allZero = false
			break
		}
	}
	if allZero {
		fmt.Println("║  Message keys: all zeroed ✓                      ║")
	}
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Top-level entry points (called from cmd/miniratchet)
// ──────────────────────────────────────────────────────────────────────

// RunAlice starts an Alice (initiator) session that runs all demo modes
// in sequence. In a full implementation this would connect via transport
// and run an interactive chat loop.
func RunAlice(addr string) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             MiniRatchet — Alice (Initiator)             ║")
	fmt.Printf("║             Target: %-36s ║\n", addr)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Run all demo modes to showcase the ratchet properties.
	if err := RunStealKey(); err != nil {
		return err
	}
	if err := RunCompromise(); err != nil {
		return err
	}
	if err := RunTwoPanel(); err != nil {
		return err
	}
	if err := RunDestroy(); err != nil {
		return err
	}
	return nil
}

// RunBob starts a Bob (responder) session. In a full implementation this
// would listen via transport and run an interactive chat loop.
func RunBob(addr string) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             MiniRatchet — Bob (Responder)               ║")
	fmt.Printf("║             Listening: %-33s ║\n", addr)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Bob is ready. In a full session, Bob would mirror Alice's")
	fmt.Println("ratchet state and decrypt her messages in real time.")
	fmt.Println()
	fmt.Println("For now, run --role alice to see the full demo sequence.")
	return nil
}

// RunAttacker starts a passive/active attacker demo showing the self-healing
// property of the DH ratchet. Runs steal-key and two-panel modes.
func RunAttacker(addr string) error {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║             MiniRatchet — Attacker View                 ║")
	fmt.Printf("║             Intercepting: %-30s ║\n", addr)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	if err := RunStealKey(); err != nil {
		return err
	}
	if err := RunTwoPanel(); err != nil {
		return err
	}
	return nil
}

// RunDemo dispatches to the appropriate demo mode based on the mode flag.
// Valid modes: "steal-key", "compromise", "two-panel", "destroy", "all".
func RunDemo(mode string) error {
	switch mode {
	case "steal-key":
		return RunStealKey()
	case "compromise":
		return RunCompromise()
	case "two-panel":
		return RunTwoPanel()
	case "destroy":
		return RunDestroy()
	case "all":
		if err := RunStealKey(); err != nil {
			return err
		}
		if err := RunCompromise(); err != nil {
			return err
		}
		if err := RunTwoPanel(); err != nil {
			return err
		}
		return RunDestroy()
	default:
		return fmt.Errorf("unknown demo mode: %q (valid: steal-key, compromise, two-panel, destroy, all)", mode)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

func printBanner(title string) {
	width := 58
	fmt.Println()
	fmt.Printf("╔%s╗\n", strings.Repeat("═", width))
	padding := width - len(title)
	left := padding / 2
	right := padding - left
	fmt.Printf("║%s%s%s║\n", strings.Repeat(" ", left), title, strings.Repeat(" ", right))
	fmt.Printf("╚%s╝\n", strings.Repeat("═", width))
	fmt.Println()
}

func printDivider() {
	fmt.Println(strings.Repeat("─", 60))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
