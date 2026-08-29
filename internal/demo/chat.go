package demo

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/miniratchet/internal/ratchet"
)

// RunInteractiveChat launches a single-terminal simulated ratcheting chat.
// It proves encryption by displaying the cryptographic lifecycle of each message.
func RunInteractiveChat() error {
	fmt.Print(Clear)
	fmt.Println(Cyan + "╔══════════════════════════════════════════════════════════════════╗" + Reset)
	fmt.Println(Cyan + "║                 INTERACTIVE CHAT MODE                            ║" + Reset)
	fmt.Println(Cyan + "║  Type a message and see the cryptography happen in real-time.    ║" + Reset)
	fmt.Println(Cyan + "║  (Type 'exit' or 'quit' to leave)                                ║" + Reset)
	fmt.Println(Cyan + "╚══════════════════════════════════════════════════════════════════╝" + Reset)
	fmt.Println()

	var rootKey, chainKey ratchet.Key32
	copy(rootKey[:], []byte("interactive-demo-root-key-val!!"))
	copy(chainKey[:], []byte("interactive-demo-chain-key-v!!"))
	
	// Create Alice (You) and Bob (Auto-reply)
	alice := NewSession(rootKey, chainKey)
	bob := NewSession(rootKey, chainKey)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(Purple + "Alice (You) > " + Reset)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" || input == "quit" {
			break
		}
		if input == "" {
			continue
		}

		// Alice encrypts
		ct, msgKey, err := alice.EncryptMessage([]byte(input))
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}

		fmt.Printf(Dark+"  [🔒 Ratchet] Key %s generated. Old keys zeroed.\n"+Reset, hex.EncodeToString(msgKey[:8]))
		fmt.Printf(Dark+"  [🔒 Encrypt] Plaintext encrypted with AES-256-GCM. AD: miniratchet-msg-%d\n"+Reset, alice.MsgCounter)
		fmt.Printf(Dark+"  [📡 Network] Transmitting %d bytes: 0x%s...\n"+Reset, len(ct), hex.EncodeToString(ct[:min(16, len(ct))]))
		
		AnimateTransmission("Alice", "Bob")
		
		// Bob decrypts
		pt, _, err := bob.DecryptMessage(ct)
		if err != nil {
			return fmt.Errorf("bob decrypt: %w", err)
		}
		fmt.Println(Cyan + "Bob < Received: " + Reset + "\"" + string(pt) + "\"")
		
		// Visualizer chain
		fmt.Println(alice.Visualizer.Render())
		fmt.Println()
		
		time.Sleep(500 * time.Millisecond)
		
		// Bob Auto-reply
		reply := fmt.Sprintf("Message received! (Ack msg %d)", bob.MsgCounter)
		fmt.Println(Cyan + "Bob (Auto) > " + Reset + reply)
		
		ctReply, msgKeyReply, err := bob.EncryptMessage([]byte(reply))
		if err != nil {
			return fmt.Errorf("bob encrypt: %w", err)
		}
		fmt.Printf(Dark+"  [🔒 Ratchet] Key %s generated. Old keys zeroed.\n"+Reset, hex.EncodeToString(msgKeyReply[:8]))
		fmt.Printf(Dark+"  [🔒 Encrypt] Plaintext encrypted with AES-256-GCM. AD: miniratchet-msg-%d\n"+Reset, bob.MsgCounter)
		fmt.Printf(Dark+"  [📡 Network] Transmitting %d bytes: 0x%s...\n"+Reset, len(ctReply), hex.EncodeToString(ctReply[:min(16, len(ctReply))]))
		
		AnimateTransmission("Bob", "Alice")
		
		ptReply, _, err := alice.DecryptMessage(ctReply)
		if err != nil {
			return fmt.Errorf("alice decrypt: %w", err)
		}
		
		fmt.Println(Purple + "Alice (You) < Received: " + Reset + "\"" + string(ptReply) + "\"")
		fmt.Println(alice.Visualizer.Render())
		fmt.Println()
	}
	
	return nil
}
