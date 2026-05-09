// Package testhelper provides test-only helpers for spawning a
// mocksbr subprocess on real loopback UDP. Use this when you need to
// exercise the binary's read/dispatch loop end-to-end; tests that only
// need the device state machine or response wire format should use the
// in-process MockTransport via mocksbr.NewMockTransport instead.
package testhelper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// MockHandle is a running mocksbr subprocess. Test cleanup happens via
// t.Cleanup; callers don't need to defer anything.
type MockHandle struct {
	cmd *exec.Cmd
	// Port is the loopback UDP port the mock is listening on.
	Port int
	// Stderr is the buffered stderr output captured up to the
	// "listening" line. Not updated after the handshake completes.
	StderrPrefix string
}

// SpawnMock builds and runs cmd/mocksbr on an OS-picked loopback port,
// blocks until the binary logs "mocksbr listening" with the bound port,
// and returns a handle. The subprocess is killed on test cleanup.
//
// Pass extra flags via args; --listen is supplied by SpawnMock and
// must not be in args.
func SpawnMock(t *testing.T, args ...string) *MockHandle {
	t.Helper()
	bin := buildBinary(t)

	// Listen on 127.0.0.1:0 so the OS picks a free port. Mocksbr's
	// "listening" log line includes the bound address.
	full := append([]string{"--listen", "127.0.0.1:0", "-v"}, args...)
	cmd := exec.Command(bin, full...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mocksbr: %v", err)
	}

	port, prefix, err := readListenPort(stderr, 5*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("waiting for mocksbr listen line: %v", err)
	}

	// Drain stderr in the background so the pipe doesn't fill.
	go io.Copy(io.Discard, stderr)

	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	return &MockHandle{cmd: cmd, Port: port, StderrPrefix: prefix}
}

var listenRE = regexp.MustCompile(`addr=127\.0\.0\.1:(\d+)`)

// readListenPort reads stderr looking for the "mocksbr listening"
// log line and returns the bound port.
func readListenPort(r io.Reader, timeout time.Duration) (int, string, error) {
	ch := make(chan struct {
		port int
		buf  string
		err  error
	}, 1)
	go func() {
		buf := new(strings.Builder)
		s := bufio.NewScanner(r)
		for s.Scan() {
			line := s.Text()
			buf.WriteString(line)
			buf.WriteByte('\n')
			if !strings.Contains(line, "mocksbr listening") {
				continue
			}
			m := listenRE.FindStringSubmatch(line)
			if len(m) != 2 {
				continue
			}
			port, err := strconv.Atoi(m[1])
			ch <- struct {
				port int
				buf  string
				err  error
			}{port, buf.String(), err}
			return
		}
		ch <- struct {
			port int
			buf  string
			err  error
		}{0, buf.String(), fmt.Errorf("mocksbr exited before logging listen line")}
	}()

	select {
	case res := <-ch:
		return res.port, res.buf, res.err
	case <-time.After(timeout):
		return 0, "", fmt.Errorf("timeout waiting for listen line")
	}
}

// buildBinary compiles cmd/mocksbr to a tempfile and returns the path.
// The build is cached per-test-binary via go's test cache; calling this
// multiple times in one test process is cheap after the first.
func buildBinary(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "mocksbr")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	// Find the module root: walk up from this file's location.
	pkgDir, err := moduleRoot()
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", out, "./cmd/mocksbr")
	cmd.Dir = pkgDir
	cmd.Env = nil // inherit
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build cmd/mocksbr: %v\n%s", err, output)
	}
	return out
}

// moduleRoot returns the path to the go-udap module root, derived from
// $GOPATH-independent module-relative paths.
func moduleRoot() (string, error) {
	cmd := exec.CommandContext(context.Background(), "go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		return "", fmt.Errorf("no go.mod in module path")
	}
	return filepath.Dir(gomod), nil
}
