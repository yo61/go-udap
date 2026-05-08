package cli

import (
	"fmt"
	"io"
)

// These stubs are replaced one per task in the following tasks.

func runRead(args []string, stdout, stderr io.Writer) error {
	return notImplemented("read")
}
func runGet(args []string, stdout, stderr io.Writer) error {
	return notImplemented("get")
}
func runSet(args []string, stdout, stderr io.Writer) error {
	return notImplemented("set")
}
func runSave(args []string, stdout, stderr io.Writer) error {
	return notImplemented("save")
}
func runReset(args []string, stdout, stderr io.Writer) error {
	return notImplemented("reset")
}
func runCommit(args []string, stdout, stderr io.Writer) error {
	return notImplemented("commit")
}

func notImplemented(name string) error {
	return &ExitError{Code: 2, Err: fmt.Errorf("%s: not implemented yet", name)}
}
