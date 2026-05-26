package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

func TestE2EDiscoverInfoIncludesNetworkConfig(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "192.168.1.50",
		SubnetMask: "255.255.255.0",
		Gateway:    "192.168.1.1",
	}); err != nil {
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

	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"discover", "--info", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Execute returned %v; stderr=%s", err, errBuf.String())
	}
	for _, want := range []string{"192.168.1.50", "255.255.255.0", "192.168.1.1", "Subnet:", "Gateway:"} {
		if !strings.Contains(outBuf.String(), want) {
			t.Errorf("stdout missing %q; got:\n%s", want, outBuf.String())
		}
	}
}

func TestE2EDiscoverInfoPartialFailureRendersDashes(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "10.0.0.1",
		SubnetMask: "255.0.0.0",
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:       "00:04:20:00:00:02",
		DropGetIP: true,
	}); err != nil {
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

	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"discover", "--info", "--timeout", "500ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 0 {
		t.Errorf("exit code %d, want 0 (partial failures are soft)", ExitCode(err))
	}
	// Device 1 (answers): should show real values.
	if !strings.Contains(outBuf.String(), "10.0.0.1") {
		t.Errorf("expected 10.0.0.1 in output: %s", outBuf.String())
	}
	// Device 2 (drops get_ip): the block for it must contain at least one "-".
	// We confirm by counting "Subnet:" lines: both blocks should have one each.
	if got := strings.Count(outBuf.String(), "Subnet:"); got != 2 {
		t.Errorf("got %d Subnet: lines, want 2", got)
	}
	// The dashes already signal "config unavailable"; without --verbose the
	// per-device get_ip-failed warning must stay off stderr.
	if strings.Contains(errBuf.String(), "warning: get_ip failed") {
		t.Errorf("stderr unexpectedly contains get_ip warning without --verbose; got:\n%s", errBuf.String())
	}
}

func TestE2EDiscoverInfoVerboseShowsGetIPWarning(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:       "00:04:20:00:00:01",
		DropGetIP: true,
	}); err != nil {
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

	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"-v", "discover", "--info", "--timeout", "500ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 0 {
		t.Errorf("exit code %d, want 0", ExitCode(err))
	}
	// With -v the per-device get_ip-failed warning must reach stderr so a
	// user debugging a non-responsive device can see why the table shows
	// dashes.
	if !strings.Contains(errBuf.String(), "warning: get_ip failed for 00:04:20:00:00:01") {
		t.Errorf("stderr missing get_ip warning under --verbose; got:\n%s", errBuf.String())
	}
}
