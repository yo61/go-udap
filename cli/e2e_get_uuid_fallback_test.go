package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

// When discovery omits TLV 0x0d, the CLI should fall back to get_uuid
// to populate the UUID and surface it in `info` / `discover --info`.
func TestE2EInfoFallsBackToGetUUID(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:                   "00:04:20:00:00:01",
		UUID:                  "deadbeefcafebabe1122334455667788",
		SuppressDiscoveryUUID: true, // forces fallback
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

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"info", "00:04:20:00:00:01", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	if !strings.Contains(outBuf.String(), "deadbeefcafebabe1122334455667788") {
		t.Errorf("stdout missing UUID from get_uuid fallback; got:\n%s", outBuf.String())
	}
	if !strings.Contains(outBuf.String(), "UUID:") {
		t.Errorf("stdout missing UUID label; got:\n%s", outBuf.String())
	}
}

// When discovery omits TLV 0x0d AND the device drops get_uuid, the
// fallback fails silently (no warning without --verbose) and the UUID
// line is omitted from the info output.
func TestE2EInfoFallbackFailureLeavesUUIDEmpty(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:                   "00:04:20:00:00:01",
		UUID:                  "deadbeefcafebabe1122334455667788",
		SuppressDiscoveryUUID: true,
		DropGetUUID:           true, // fallback also fails
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

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"info", "00:04:20:00:00:01", "--timeout", "300ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	// UUID line must be absent (formatDeviceInfo skips empty UUID).
	if strings.Contains(outBuf.String(), "UUID:") {
		t.Errorf("stdout unexpectedly contains UUID label; got:\n%s", outBuf.String())
	}
	// And the fallback's warning must NOT appear without --verbose.
	if strings.Contains(errBuf.String(), "get_uuid fallback failed") {
		t.Errorf("stderr unexpectedly contains fallback warning without --verbose; got:\n%s", errBuf.String())
	}
}

// Under --verbose, the fallback's failure DOES surface the warning to stderr.
func TestE2EInfoFallbackFailureVerboseShowsWarning(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })
	if _, err := network.Add(mocksbr.DeviceConfig{
		MAC:                   "00:04:20:00:00:01",
		SuppressDiscoveryUUID: true,
		DropGetUUID:           true,
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

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"-v", "info", "00:04:20:00:00:01", "--timeout", "300ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "warning: get_uuid fallback failed for 00:04:20:00:00:01") {
		t.Errorf("stderr missing fallback warning under --verbose; got:\n%s", errBuf.String())
	}
}
