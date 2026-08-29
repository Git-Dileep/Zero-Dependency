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
	Purple = "\033[1;35m"
	Yellow = "\033[1;33m"
)

func main() {
	fmt.Println(Purple + " Alice " + Reset + "                                       " + Cyan + " Bob " + Reset)
	fmt.Println(Dark + " 👤 " + Reset + "                                          " + Dark + " 👤 " + Reset)
	fmt.Println(Dark + " │ " + Reset + "                                           " + Dark + " │ " + Reset)
	
	distance := 40

	// Animate a particle from Alice to Bob
	for i := 0; i <= distance; i++ {
		fmt.Print("\r" + Dark + " │ " + Reset + " ")
		
		// The trail
		trail := strings.Repeat(Dark+"·"+Reset, i)
		// The particle
		particle := Cyan + "►" + Reset
		// The remaining space
		remaining := strings.Repeat(" ", distance-i)
		
		fmt.Print(trail + particle + remaining + Dark + " │ " + Reset)
		time.Sleep(50 * time.Millisecond)
	}
	
	// Flash "MESSAGE DELIVERED"
	fmt.Print("\r\033[K") // Clear line
	fmt.Print(Dark + " │ " + Reset + " ")
	fmt.Print(strings.Repeat(Dark+"·"+Reset, distance))
	fmt.Print(Dark + " │ " + Reset)
	
	fmt.Println("\n" + Green + "                   [ MESSAGE DELIVERED ✓ ]                   " + Reset)
	time.Sleep(500 * time.Millisecond)
}
