// Package main provides the CLI entrypoint for MiniRatchet, a zero-dependency
// Signal-style ratchet messenger.
//
// Usage:
//
//	miniratchet                           # (default) run the evaluator guided tour & interactive chat
//	miniratchet --role alice              # run all demos as Alice (initiator)
//	miniratchet --role bob                # start as Bob   (responder)
//	miniratchet --role attacker           # run attacker-focused demos
//	miniratchet --demo steal-key          # run a single demo mode
//	miniratchet --demo all                # run all demo modes in sequence
//
// Phase 5 — wired to internal/demo.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/miniratchet/internal/demo"
)

func main() {
	role := flag.String("role", "", "role to assume: alice, bob, or attacker")
	demoMode := flag.String("demo", "", "demo mode: steal-key, compromise, two-panel, destroy, all")
	tour := flag.Bool("tour", false, "run the guided evaluator tour (default if no args)")
	addr := flag.String("addr", "localhost:9000", "address for TCP connection")
	flag.Parse()

	// If no flags provided, or explicit --tour, run the guided evaluator experience.
	if flag.NFlag() == 0 || *tour {
		if err := demo.RunGuidedTour(); err != nil {
			fmt.Fprintf(os.Stderr, "tour error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Print("\n\033[1;37mEnter Interactive Chat Mode? [Y/n]: \033[0m")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "n" && input != "no" {
			if err := demo.RunInteractiveChat(); err != nil {
				fmt.Fprintf(os.Stderr, "chat error: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	// If --demo is specified, run that specific demo mode directly.
	if *demoMode != "" {
		if err := demo.RunDemo(*demoMode); err != nil {
			fmt.Fprintf(os.Stderr, "demo error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Otherwise dispatch by role.
	switch *role {
	case "alice":
		if err := demo.RunAlice(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "alice error: %v\n", err)
			os.Exit(1)
		}
	case "bob":
		if err := demo.RunBob(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "bob error: %v\n", err)
			os.Exit(1)
		}
	case "attacker":
		if err := demo.RunAttacker(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "attacker error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "usage: miniratchet [no args] (runs the guided tour)\n")
		fmt.Fprintf(os.Stderr, "       miniratchet --role alice|bob|attacker\n")
		fmt.Fprintf(os.Stderr, "       miniratchet --demo steal-key|compromise|two-panel|destroy|all\n")
		os.Exit(1)
	}
}
