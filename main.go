package main

import (
	"fmt"
	"os"

	"go-udap/cli"
)

func main() {
	err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(cli.ExitCode(err))
}
