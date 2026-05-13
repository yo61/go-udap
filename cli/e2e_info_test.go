package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

func TestE2EInfoPrintsDeviceMetadata(t *testing.T) {
	env := startMockEnv(t, 2)
	stdout, _, exitCode := env.runCLI(t, "info", "00:04:20:00:00:02", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	for _, want := range []string{
		"00:04:20:00:00:02",
		"Squeezebox Receiver",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q; got:\n%s", want, stdout)
		}
	}
}

func TestE2EInfoMissingMACIsExitCodeTwo(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "info", "aa:bb:cc:dd:ee:ff", "--timeout", "200ms")
	if exitCode != 2 {
		t.Errorf("exit code %d, want 2 (device not found)", exitCode)
	}
}

func TestE2EInfoPrintsHardwareRevAndUUID(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	_, err := network.Add(mocksbr.DeviceConfig{
		MAC:      "00:04:20:00:00:01",
		Hardware: "0005",
		UUID:     "deadbeefcafebabe1122334455667788",
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err = Run([]string{"info", "00:04:20:00:00:01", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	for _, want := range []string{"HW Rev:", "0005", "UUID:", "deadbeefcafebabe1122334455667788"} {
		if !strings.Contains(outBuf.String(), want) {
			t.Errorf("stdout missing %q; got:\n%s", want, outBuf.String())
		}
	}
}
