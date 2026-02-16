// Package main is the entry point for the tfe-run application.
package main

import (
	"os"

	"github.com/thulasirajkomminar/tfe-run/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
