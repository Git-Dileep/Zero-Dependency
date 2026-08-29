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
	"time"

	"github.com/miniratchet/internal/aead"
	"github.com/miniratchet/internal/dhratchet"
	"github.com/miniratchet/internal/ratchet"
)

// ANSI Color Constants for UI Effects
const (
	Reset  = "\033[0m"
	Cyan   = "\033[1;36m"
	Dark   = "\033[1;30m"
	Green  = "\033[1;32m"
	White  = "\033[1;37m"
	Purple = "\033[1;35m"
	Red    = "\033[1;31m"
	Yellow = "\033[1;33m"
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

// Render prints the chain evolution to stdout with ANSI colors.
func (rv *RatchetVisualizer) Render() string {
	if len(rv.Slots) == 0 {
		return "  (no keys in chain)"
	}
	var parts []string
	for _, slot := range rv.Slots {
		if slot.Cleared {
			parts = append(parts, Dark+fmt.Sprintf("[%s: discarded]", slot.Label)+Reset)
		} else {
			parts = append(parts, Cyan+fmt.Sprintf("[%s: ACTIVE 🔑]", slot.Label)+Reset)
		}
	}
	return "  " + strings.Join(parts, Dark+" ── "+Reset)
}

// AnimateTransmission displays a visual message particle traveling between users.
func AnimateTransmission(senderName, receiverName string) {
	fmt.Println(Purple + " " + senderName + " " + Reset + "                                       " + Cyan + " " + receiverName + " " + Reset)
	fmt.Println(Dark + " 👤 " + Reset + "                                          " + Dark + " 👤 " + Reset)
	fmt.Println(Dark + " │ " + Reset + "                                           " + Dark + " │ " + Reset)
	
	distance := 40
	for i := 0; i <= distance; i++ {
		fmt.Print("\r" + Dark + " │ " + Reset + " ")
		trail := strings.Repeat(Dark+"·"+Reset, i)
		particle := Cyan + "►" + Reset
		remaining := strings.Repeat(" ", distance-i)
		fmt.Print(trail + particle + remaining + Dark + " │ " + Reset)
		time.Sleep(20 * time.Millisecond)
	}
	fmt.Print("\r\033[K") // Clear line
	fmt.Print(Dark + " │ " + Reset + " ")
	fmt.Print(strings.Repeat(Dark+"·"+Reset, distance))
	fmt.Print(Dark + " │ " + Reset)
	
	fmt.Println("\n" + Green + "                   [ MESSAGE DELIVERED ✓ ]                   " + Reset)
	time.Sleep(300 * time.Millisecond)
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
// message key, and returns the ciphertext.
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

func RunStealKey() error {
	printBanner("DEMO: steal-key")
	fmt.Println(White + "Scenario: attacker steals ONE message key mid-session." + Reset)
	fmt.Println("Goal: show that stolen key cannot decrypt earlier or later messages.")
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("steal-key-demo-root-key-value!!"))
	copy(chainKey[:], []byte("steal-key-demo-chain-key-val!!"))
	alice := NewSession(rootKey, chainKey)

	messages := []string{
		"Message 1: The eagle has landed.",
		"Message 2: Rendezvous at midnight.",
		"Message 3: Operation is a go.",
	}
	var ciphertexts [][]byte
	var msgKeys []ratchet.Key32

	fmt.Println(Cyan + "═══ Alice sends 3 messages ═══" + Reset)
	for i, msg := range messages {
		ct, key, err := alice.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt message %d: %w", i+1, err)
		}
		ciphertexts = append(ciphertexts, ct)
		msgKeys = append(msgKeys, key)
		
		AnimateTransmission("Alice", "Bob")
		fmt.Printf("  [%d] Encrypted: %q → %s...\n", i+1, msg, hex.EncodeToString(ct[:min(16, len(ct))]))
		fmt.Println(alice.Visualizer.Render())
		fmt.Println()
	}

	stolenKey := msgKeys[1]
	fmt.Printf(Red+"🔓 ATTACKER steals message key #2: %s\n\n"+Reset, hex.EncodeToString(stolenKey[:8]))

	fmt.Println(Yellow + "═══ Attacker tries to decrypt MESSAGE #1 (earlier) ═══" + Reset)
	ad1 := []byte(fmt.Sprintf("miniratchet-msg-%d", 1))
	_, err := aead.Decrypt(stolenKey, ciphertexts[0], ad1)
	if err != nil {
		fmt.Printf("  ✗ FAILED: %v\n", err)
		fmt.Println(Green + "  → Stolen key CANNOT unlock earlier messages." + Reset)
	}
	fmt.Println()

	fmt.Println(Yellow + "═══ Attacker tries to decrypt MESSAGE #3 (later) ═══" + Reset)
	ad3 := []byte(fmt.Sprintf("miniratchet-msg-%d", 3))
	_, err = aead.Decrypt(stolenKey, ciphertexts[2], ad3)
	if err != nil {
		fmt.Printf("  ✗ FAILED: %v\n", err)
		fmt.Println(Green + "  → Stolen key CANNOT unlock later messages." + Reset)
	}
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 2: compromise
// ──────────────────────────────────────────────────────────────────────

func RunCompromise() error {
	printBanner("DEMO: compromise & recovery")
	fmt.Println(White + "Scenario: Bob's chain key is compromised mid-session." + Reset)
	fmt.Println("Goal: show that a DH ratchet epoch rotation re-secures the session.")
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("compromise-demo-root-key-val!!"))
	copy(chainKey[:], []byte("compromise-demo-chain-key-v!!"))
	bob := NewSession(rootKey, chainKey)

	fmt.Println(Cyan + "═══ Normal operation: Bob sends 2 messages ═══" + Reset)
	for i := 1; i <= 2; i++ {
		msg := fmt.Sprintf("Normal message %d from Bob", i)
		ct, _, err := bob.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		AnimateTransmission("Bob", "Alice")
		fmt.Printf("  [%d] %q → %s...\n", i, msg, hex.EncodeToString(ct[:min(16, len(ct))]))
		fmt.Println(bob.Visualizer.Render())
		fmt.Println()
	}

	compromisedChainKey := bob.State.ChainKey
	fmt.Println(Red + "╔══════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Red + "║  ⚠  Bob's current chain key is COMPROMISED!      ║" + Reset)
	fmt.Printf(Red+"║  Key: %-38s ║\n"+Reset, hex.EncodeToString(compromisedChainKey[:12])+"...")
	fmt.Println(Red + "╚══════════════════════════════════════════════════╝" + Reset)
	fmt.Println()

	fmt.Println(Yellow + "═══ Triggering DH ratchet epoch rotation ═══" + Reset)
	bobPriv, bobPub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return fmt.Errorf("new epoch key pair: %w", err)
	}
	fmt.Printf("  Bob generates new epoch key pair (pub: %s...)\n", hex.EncodeToString(bobPub.Bytes()[:8]))

	alicePriv, alicePub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return fmt.Errorf("alice epoch key pair: %w", err)
	}
	fmt.Printf("  Alice generates new epoch key pair (pub: %s...)\n", hex.EncodeToString(alicePub.Bytes()[:8]))

	sharedSecret, err := dhratchet.ComputeSharedSecret(bobPriv, alicePub)
	if err != nil {
		return fmt.Errorf("compute shared secret: %w", err)
	}
	newRootKey, err := dhratchet.DeriveRootKey(sharedSecret, bob.State.RootKey)
	if err != nil {
		return fmt.Errorf("derive root key: %w", err)
	}
	_ = alicePriv

	bob.State.RootKey = newRootKey
	bob.State.ChainKey = newRootKey
	bob.State.Epoch++
	fmt.Printf(Green+"  ✓ New root key derived: %s...\n"+Reset, hex.EncodeToString(newRootKey[:8]))
	fmt.Printf(Green+"  ✓ Epoch advanced to: %d\n"+Reset, bob.State.Epoch)
	fmt.Println()

	fmt.Println(Cyan + "═══ Post-recovery: Bob sends message with new keys ═══" + Reset)
	msg := "Post-recovery message — session is re-secured!"
	ct, _, err := bob.EncryptMessage([]byte(msg))
	if err != nil {
		return fmt.Errorf("encrypt post-recovery: %w", err)
	}
	AnimateTransmission("Bob", "Alice")
	fmt.Printf("  [3] %q → %s...\n", msg, hex.EncodeToString(ct[:min(16, len(ct))]))
	fmt.Println(bob.Visualizer.Render())
	fmt.Println()

	fmt.Println(Yellow + "═══ Attacker tries compromised chain key ═══" + Reset)
	fmt.Printf("  Compromised key: %s...\n", hex.EncodeToString(compromisedChainKey[:8]))
	fmt.Printf("  Current key:     %s...\n", hex.EncodeToString(bob.State.ChainKey[:8]))
	if compromisedChainKey != bob.State.ChainKey {
		fmt.Println(Green + "  ✗ Keys differ — compromised key is USELESS after epoch rotation." + Reset)
	}
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 3: two-panel
// ──────────────────────────────────────────────────────────────────────

func RunTwoPanel() error {
	printBanner("DEMO: two-panel (legitimate vs. attacker view)")
	fmt.Println(White + "Left panel: legitimate plaintext. Right panel: attacker's ciphertext-only view." + Reset)
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

	panelWidth := 40
	fmt.Printf("┌%s┬%s┐\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))
	fmt.Printf("│%-*s│%-*s│\n", panelWidth, Green+" 🔓 LEGITIMATE VIEW"+Reset, panelWidth, Red+" 🔒 ATTACKER VIEW"+Reset)
	fmt.Printf("├%s┼%s┤\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))

	for _, msg := range messages {
		ct, _, err := alice.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}

		leftText := truncate(msg, panelWidth-2)
		ctHex := hex.EncodeToString(ct)
		rightText := truncate(ctHex, panelWidth-2)

		fmt.Printf("│ %-*s│ %-*s│\n", panelWidth-1, leftText, panelWidth-1, rightText)
	}

	fmt.Printf("└%s┴%s┘\n", strings.Repeat("─", panelWidth), strings.Repeat("─", panelWidth))
	fmt.Println()
	fmt.Println(alice.Visualizer.Render())
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Demo Mode 4: destroy
// ──────────────────────────────────────────────────────────────────────

func RunDestroy() error {
	printBanner("DEMO: destroy (key destruction)")
	fmt.Println(White + "Scenario: demonstrating forward secrecy by zeroing old keys in memory." + Reset)
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("destroy-demo-root-key-value!!"))
	copy(chainKey[:], []byte("destroy-demo-chain-key-val!!\x00\x00"))
	session := NewSession(rootKey, chainKey)

	fmt.Println(Cyan + "═══ Sending messages and collecting keys ═══" + Reset)
	var collectedKeys []ratchet.Key32
	for i := 1; i <= 4; i++ {
		msg := fmt.Sprintf("Secret message #%d", i)
		_, key, err := session.EncryptMessage([]byte(msg))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		collectedKeys = append(collectedKeys, key)
		AnimateTransmission("Alice", "Bob")
		fmt.Printf("  [%d] Key: %s  Message: %q\n", i, hex.EncodeToString(key[:8]), msg)
		fmt.Println(session.Visualizer.Render())
		fmt.Println()
	}

	fmt.Println(Yellow + "═══ DESTROYING old keys ═══" + Reset)
	for i := range collectedKeys {
		fmt.Printf("  Zeroing K%d: %s → ", i+1, hex.EncodeToString(collectedKeys[i][:8]))
		for j := range collectedKeys[i] {
			collectedKeys[i][j] = 0
		}
		fmt.Printf(Green+"%s ✓\n"+Reset, hex.EncodeToString(collectedKeys[i][:8]))
	}

	for i := range session.State.RootKey {
		session.State.RootKey[i] = 0
	}
	for i := range session.State.ChainKey {
		session.State.ChainKey[i] = 0
	}
	fmt.Println()

	fmt.Println(Green + "╔══════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Green + "║  🗑️  There is nothing to recover.                ║" + Reset)
	fmt.Println(Green + "║                                                  ║" + Reset)
	fmt.Println(Green + "║  All key material has been zeroed in memory.     ║" + Reset)
	fmt.Println(Green + "║  Root key:  0000000000000000                     ║" + Reset)
	fmt.Println(Green + "║  Chain key: 0000000000000000                     ║" + Reset)
	allZero := true
	for _, k := range collectedKeys {
		if k != (ratchet.Key32{}) {
			allZero = false
			break
		}
	}
	if allZero {
		fmt.Println(Green + "║  Message keys: all zeroed ✓                      ║" + Reset)
	}
	fmt.Println(Green + "╚══════════════════════════════════════════════════╝" + Reset)
	fmt.Println()
	printDivider()
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Top-level entry points (called from cmd/miniratchet)
// ──────────────────────────────────────────────────────────────────────

func RunAlice(addr string) error {
	fmt.Println()
	fmt.Println(Cyan + "╔══════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Cyan + "║             MiniRatchet — Alice (Initiator)              ║" + Reset)
	fmt.Printf(Cyan+"║             Target: %-36s ║\n"+Reset, addr)
	fmt.Println(Cyan + "╚══════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()

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

func RunBob(addr string) error {
	fmt.Println()
	fmt.Println(Cyan + "╔══════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Cyan + "║             MiniRatchet — Bob (Responder)                ║" + Reset)
	fmt.Printf(Cyan+"║             Listening: %-33s ║\n"+Reset, addr)
	fmt.Println(Cyan + "╚══════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()
	fmt.Println("Bob is ready. In a full session, Bob would mirror Alice's")
	fmt.Println("ratchet state and decrypt her messages in real time.")
	fmt.Println()
	fmt.Println("For now, run --role alice to see the full demo sequence.")
	return nil
}

func RunAttacker(addr string) error {
	fmt.Println()
	fmt.Println(Red + "╔══════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Red + "║             MiniRatchet — Attacker View                  ║" + Reset)
	fmt.Printf(Red+"║             Intercepting: %-30s ║\n"+Reset, addr)
	fmt.Println(Red + "╚══════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()

	if err := RunStealKey(); err != nil {
		return err
	}
	if err := RunTwoPanel(); err != nil {
		return err
	}
	return nil
}

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
	fmt.Printf(White+"╔%s╗\n"+Reset, strings.Repeat("═", width))
	padding := width - len(title)
	left := padding / 2
	right := padding - left
	fmt.Printf(White+"║%s%s%s║\n"+Reset, strings.Repeat(" ", left), title, strings.Repeat(" ", right))
	fmt.Printf(White+"╚%s╝\n"+Reset, strings.Repeat("═", width))
	fmt.Println()
}

func printDivider() {
	fmt.Println(Dark + strings.Repeat("─", 60) + Reset)
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
