package main

import (
	"fmt"
	"os"

	"github.com/rafamoreira/trove/internal/cli"
)

func main() {
	if err := cli.Execute(os.Args[1:], cli.Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
