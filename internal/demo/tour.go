package demo

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const Clear = "\033[H\033[2J"

// RunGuidedTour orchestrates the sequential evaluator tour of all features.
func RunGuidedTour() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(Clear)
		fmt.Println(Cyan + `
╔══════════════════════════════════════════════════════════════════╗
║                                                                  ║
║                   WELCOME TO MINIRATCHET                         ║
║      Zero-Dependency Signal-Style Double Ratchet in Go           ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
` + Reset)
		fmt.Println(White + "This guided tour will demonstrate the core cryptographic properties." + Reset)
		fmt.Println(Dark + "Press Enter to start the tour..." + Reset)
		reader.ReadString('\n')

		if err := RunStealKey(); err != nil {
			return err
		}
		fmt.Println(Dark + "Press Enter to continue..." + Reset)
		reader.ReadString('\n')

		if err := RunCompromise(); err != nil {
			return err
		}
		fmt.Println(Dark + "Press Enter to continue..." + Reset)
		reader.ReadString('\n')

		if err := RunTwoPanel(); err != nil {
			return err
		}
		fmt.Println(Dark + "Press Enter to continue..." + Reset)
		reader.ReadString('\n')

		if err := RunDestroy(); err != nil {
			return err
		}

		fmt.Println()
		fmt.Print(White + "Repeat demonstration? [Y/n]: " + Reset)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "n" || input == "no" {
			break
		}
	}
	return nil
}
