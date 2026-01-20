package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/whywaita/shoes-vz/internal/agent/vm"
)

func runListCommand() {
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	runnersPath := listFlags.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")

	// Parse flags from os.Args[2:] (skip program name and "list" subcommand)
	if err := listFlags.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// List VMs
	vms, err := vm.ListVMs(*runnersPath)
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found")
		return
	}

	// Print VMs in a table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUNNER ID\tIP ADDRESS\tSTATE\tCREATED AT\tUPDATED AT\tBUNDLE PATH")
	fmt.Fprintln(w, "---------\t----------\t-----\t----------\t----------\t-----------")

	for _, v := range vms {
		ipAddr := v.IPAddress
		if ipAddr == "" {
			ipAddr = "<not set>"
		}

		state := v.State
		if state == "" {
			state = "<unknown>"
		}

		updatedAt := v.UpdatedAt
		if updatedAt == "" {
			updatedAt = "<not set>"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			v.RunnerID,
			ipAddr,
			state,
			v.CreatedAt,
			updatedAt,
			v.BundlePath,
		)
	}

	w.Flush()
}
