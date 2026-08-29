package main

import (
	"flag"
	"fmt"
	"os"

	"miniratchet/internal/demo"
)

func main() {
	roleFlag := flag.String("role", "", "Role to run as: alice, bob, attacker, or demo")
	demoFlag := flag.String("demo", "all", "Which demo to run: steal-key, compromise, two-panel, destroy, all")
	flag.Parse()

	if *roleFlag == "" {
		fmt.Println("Usage: miniratchet --role [alice|bob|attacker|demo] [--demo name]")
		os.Exit(1)
	}

	if *roleFlag == "demo" {
		switch *demoFlag {
		case "steal-key":
			demo.RunStealKeyDemo()
		case "compromise":
			demo.RunCompromiseDemo()
		case "two-panel":
			demo.RunTwoPanelDemo()
		case "destroy":
			demo.RunDestroyDemo()
		case "all":
			fmt.Println("=== RUNNING ALL DEMOS ===")
			demo.RunStealKeyDemo()
			fmt.Println("\n------------------------------------------------")
			demo.RunCompromiseDemo()
			fmt.Println("\n------------------------------------------------")
			demo.RunTwoPanelDemo()
			fmt.Println("\n------------------------------------------------")
			demo.RunDestroyDemo()
		default:
			fmt.Printf("Unknown demo mode: %s\n", *demoFlag)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("[+] Starting MiniRatchet in role: %s\n", *roleFlag)
	fmt.Println("[!] Network transport for alice/bob/attacker not fully wired in main.go yet.")
	fmt.Println("[*] Use '--role demo --demo all' to see the core project deliverables.")
}
