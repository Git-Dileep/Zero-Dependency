package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	Reset  = "\033[0m"
	Cyan   = "\033[1;36m"
	Dark   = "\033[1;30m"
	Green  = "\033[1;32m"
	White  = "\033[1;37m"
	Purple = "\033[1;35m"
	Clear  = "\033[H\033[2J" // Clear screen
)

func main() {
	fmt.Print(Clear)
	fmt.Println(White + "╔══════════════════════════════════════════════════╗" + Reset)
	fmt.Println(White + "║              RATCHET TIMELINE DEMO               ║" + Reset)
	fmt.Println(White + "╚══════════════════════════════════════════════════╝" + Reset)
	fmt.Println()

	// Simulate 5 messages being sent.
	totalKeys := 6
	
	for currentActive := 1; currentActive <= totalKeys; currentActive++ {
		fmt.Printf("\r\033[K") // Clear line
		
		var timeline []string
		
		for i := 1; i <= totalKeys; i++ {
			if i < currentActive {
				// Discarded key
				timeline = append(timeline, Dark+"[K"+fmt.Sprint(i)+": discarded]"+Reset)
			} else if i == currentActive {
				// Active key
				timeline = append(timeline, Cyan+"[K"+fmt.Sprint(i)+": ACTIVE 🔑]"+Reset)
			} else {
				// Future key (not generated yet)
				timeline = append(timeline, Dark+"[...........]"+Reset)
			}
		}
		
		fmt.Print("  " + strings.Join(timeline, Dark+" ── "+Reset))
		time.Sleep(800 * time.Millisecond)
	}
	fmt.Println("\n\n" + Green + "✓ Sequence Complete" + Reset)
}
