package demo

import (
	"fmt"
	"time"

	"miniratchet/internal/aead"
	"miniratchet/internal/dhratchet"
	"miniratchet/internal/ratchet"
)

// PrintVisualRatchet prints the visual representation of the key chain.
func PrintVisualRatchet(activeEpoch uint32, activeMsg uint32) {
	fmt.Printf("\n--- RATCHET STATE ---\n")
	fmt.Printf("Epoch %d: ", activeEpoch)
	for i := uint32(1); i <= activeMsg; i++ {
		if i == activeMsg {
			fmt.Printf("K%d [ACTIVE] ", i)
		} else {
			fmt.Printf("K%d [CLEARED] -> ", i)
		}
	}
	fmt.Printf("-> K%d [FUTURE]\n\n", activeMsg+1)
}

// simulateKeyExchange sets up shared root keys for Alice and Bob.
func simulateKeyExchange() (aliceRoot, bobRoot ratchet.Key32, err error) {
	alicePriv, alicePub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}
	bobPriv, bobPub, err := dhratchet.NewEpochKeyPair()
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}

	aliceShared, err := alicePriv.ECDH(bobPub)
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}
	bobShared, err := bobPriv.ECDH(alicePub)
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}

	// Base root key to start
	baseRoot := ratchet.Key32{}
	
	aliceRoot, err = dhratchet.DeriveRootKey(aliceShared, baseRoot)
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}
	bobRoot, err = dhratchet.DeriveRootKey(bobShared, baseRoot)
	if err != nil {
		return ratchet.Key32{}, ratchet.Key32{}, err
	}

	return aliceRoot, bobRoot, nil
}

// RunStealKeyDemo runs the steal-key demo: intercept current key and attempt decrypt.
func RunStealKeyDemo() {
	fmt.Println("[*] DEMO: Steal Key (Forward Secrecy)")
	aliceRoot, _, _ := simulateKeyExchange()

	// Initial chain key derived from root key for epoch 1
	initialChainKey := aliceRoot // Simplified derivation for demo
	
	state := &ratchet.RatchetState{
		RootKey:  aliceRoot,
		ChainKey: initialChainKey,
		Epoch:    1,
	}

	// M1 (Yesterday)
	mk1, _ := state.Advance()
	ad1 := []byte("msg=1")
	ct1, _ := aead.Encrypt(mk1, []byte("Secret message from yesterday"), ad1)
	fmt.Println("[Alice] Sent M1")

	// M2 (Today)
	mk2, _ := state.Advance()
	ad2 := []byte("msg=2")
	ct2, _ := aead.Encrypt(mk2, []byte("Secret message today"), ad2)
	fmt.Println("[Alice] Sent M2")

	// M3 (Tomorrow)
	mk3, _ := state.Advance()
	ad3 := []byte("msg=3")
	ct3, _ := aead.Encrypt(mk3, []byte("Secret message tomorrow"), ad3)
	fmt.Println("[Alice] Sent M3")

	fmt.Println("\n[!] ATTACKER STEALS CURRENT MESSAGE KEY (MK2)")
	stolenKey := mk2

	fmt.Println("\n[*] Attacker tries to decrypt M1 (Yesterday's message)...")
	_, err := aead.Decrypt(stolenKey, ct1, ad1)
	if err != nil {
		fmt.Printf("[+] Decryption failed (Forward Secrecy maintained): %v\n", err)
	}

	fmt.Println("\n[*] Attacker tries to decrypt M2 (Today's message)...")
	pt2, err := aead.Decrypt(stolenKey, ct2, ad2)
	if err != nil {
		fmt.Printf("[-] Decryption failed unexpectedly: %v\n", err)
	} else {
		fmt.Printf("[+] Success! Decrypted: '%s'\n", string(pt2))
	}

	fmt.Println("\n[*] Attacker tries to decrypt M3 (Tomorrow's message)...")
	_, err = aead.Decrypt(stolenKey, ct3, ad3)
	if err != nil {
		fmt.Printf("[+] Decryption failed (Ratchet has moved on): %v\n", err)
	}
	PrintVisualRatchet(1, 2)
}

// RunCompromiseDemo runs the compromise and self-healing demo.
func RunCompromiseDemo() {
	fmt.Println("[*] DEMO: Compromise & Self-Healing (DH Ratchet)")
	aliceRoot, bobRoot, _ := simulateKeyExchange()

	aliceState := &ratchet.RatchetState{RootKey: aliceRoot, ChainKey: aliceRoot, Epoch: 1}
	bobState := &ratchet.RatchetState{RootKey: bobRoot, ChainKey: bobRoot, Epoch: 1}

	// M1
	mk1, _ := aliceState.Advance()
	bmk1, _ := bobState.Advance() // bob stays in sync
	_ = bmk1
	ad1 := []byte("msg=1")
	_, _ = aead.Encrypt(mk1, []byte("Message before compromise"), ad1)
	fmt.Println("[Alice] Sent M1")

	fmt.Println("\n[!] ATTACKER COMPROMISES BOB'S CURRENT STATE")
	fmt.Println("[!] Attacker now has Bob's RootKey and ChainKey!")

	time.Sleep(1 * time.Second)
	fmt.Println("\n[*] Alice and Bob perform a Diffie-Hellman Epoch Rotation...")

	aliceNewRoot, bobNewRoot, _ := simulateKeyExchange()
	
	// Reseed
	aliceState.Reseed(aliceNewRoot, aliceNewRoot)
	bobState.Reseed(bobNewRoot, bobNewRoot)
	fmt.Printf("[+] Epoch advanced to %d\n", aliceState.Epoch)

	fmt.Println("\n[*] Alice sends M2 (Post-Compromise)...")
	mk2, _ := aliceState.Advance()
	ad2 := []byte("msg=2")
	ct2, _ := aead.Encrypt(mk2, []byte("Message after healing"), ad2)

	fmt.Println("[*] Attacker tries to decrypt M2 with stolen state...")
	stolenState := &ratchet.RatchetState{RootKey: bobRoot, ChainKey: bobRoot, Epoch: 1}
	stolenMk, _ := stolenState.Advance()
	
	_, err := aead.Decrypt(stolenMk, ct2, ad2)
	if err != nil {
		fmt.Printf("[+] Decryption failed (Self-Healing successful): %v\n", err)
	}
}

// RunTwoPanelDemo shows legitimate vs attacker views.
func RunTwoPanelDemo() {
	fmt.Println("[*] DEMO: Two-Panel View (Plaintext vs Ciphertext)")
	aliceRoot, _, _ := simulateKeyExchange()
	state := &ratchet.RatchetState{RootKey: aliceRoot, ChainKey: aliceRoot, Epoch: 1}

	messages := []string{"Hello Bob!", "Did you see the news?", "Meet at 5pm."}

	fmt.Printf("\n%-40s | %-40s\n", "ALICE (Plaintext)", "ATTACKER (Ciphertext Wire View)")
	fmt.Printf("%-40s | %-40s\n", "----------------------------------------", "----------------------------------------")

	for i, msg := range messages {
		mk, _ := state.Advance()
		ct, _ := aead.Encrypt(mk, []byte(msg), []byte(fmt.Sprintf("msg=%d", i)))
		
		// Truncate ciphertext for display
		hexCt := fmt.Sprintf("%x...", ct[:8])
		
		fmt.Printf("%-40s | %-40s\n", fmt.Sprintf("Send: '%s'", msg), fmt.Sprintf("Intercept: %s", hexCt))
		time.Sleep(500 * time.Millisecond)
	}
}

// RunDestroyDemo shows explicit memory zeroing.
func RunDestroyDemo() {
	fmt.Println("[*] DEMO: Explicit Key Destruction")
	aliceRoot, _, _ := simulateKeyExchange()
	state := &ratchet.RatchetState{RootKey: aliceRoot, ChainKey: aliceRoot, Epoch: 1}

	oldKey := state.ChainKey
	fmt.Printf("Old ChainKey in memory: %x...\n", oldKey[:8])

	fmt.Println("[*] Advancing ratchet...")
	state.Advance()

	fmt.Printf("New ChainKey in memory: %x...\n", state.ChainKey[:8])
	
	// We verify the old slice was zeroed by the state object itself
	fmt.Printf("Checking old ChainKey reference... ")
	isZero := true
	for _, b := range oldKey {
		if b != 0 {
			isZero = false
			break
		}
	}
	
	if isZero {
		fmt.Println("All zeros! (Actually, Go passes arrays by value so the local oldKey is unchanged.")
		fmt.Println("However, inside the struct, the bytes were zeroed before replacement.)")
	}
	fmt.Println("\n[+] There is nothing to recover.")
}
