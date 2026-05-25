package cli

import (
	"strings"
	"testing"

	"go-udap/mocksbr"
)

// TestE2ESetFactoryFreshDefaultsToWired covers the auto-default behavior
// for the NVRAM interface byte. A freshly-reset Squeezebox reports
// interface=128 — a "not configured" sentinel that the device firmware
// refuses to act on (network never comes up after the post-set reboot).
// Plain RMW would faithfully write 128 back, so the CLI substitutes a
// concrete value: wired (1) by default, surfaced on stderr so the user
// sees the substitution.
func TestE2ESetFactoryFreshDefaultsToWired(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, stderr, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.20.30.40", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "interface=1") {
		t.Errorf("stdout should echo injected interface=1; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "server_address=10.20.30.40") {
		t.Errorf("stdout should still echo the user's --server-address; got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "device interface is unset") {
		t.Errorf("stderr should announce the inference; got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "defaulting to wired") {
		t.Errorf("stderr should name the chosen default; got:\n%s", stderr)
	}
}

// TestE2ESetFactoryFreshWithWirelessSSIDInfersWireless covers the
// wireless-intent inference. If the user supplies --wireless-ssid (the
// unambiguous "join this network now" signal), the CLI flips the default
// from wired to wireless.
func TestE2ESetFactoryFreshWithWirelessSSIDInfersWireless(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, stderr, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--wireless-ssid", "my-net",
		"--wireless-wpa-psk", "eight-chars",
		"--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "interface=0") {
		t.Errorf("stdout should echo injected interface=0; got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "inferred wireless from --wireless-ssid") {
		t.Errorf("stderr should announce wireless inference; got:\n%s", stderr)
	}
}

// TestE2ESetExplicitInterfaceSilencesNotice ensures the user can opt out
// of the inference noise by being explicit. If --interface is supplied,
// no substitution happens and no notice is printed.
func TestE2ESetExplicitInterfaceSilencesNotice(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, stderr, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--interface", "1",
		"--server-address", "10.20.30.40",
		"--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "interface=1") {
		t.Errorf("stdout should echo the user's --interface 1; got:\n%s", stdout)
	}
	if strings.Contains(stderr, "device interface is unset") {
		t.Errorf("explicit --interface should silence the notice; got stderr:\n%s", stderr)
	}
}

// TestE2ESetAlreadyConfiguredInterfacePreserved ensures the inference
// only fires when the device's current value is the unset sentinel. A
// device that already reports interface=0 (configured wireless) keeps
// that value through RMW with no notice.
func TestE2ESetAlreadyConfiguredInterfacePreserved(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:   "aa:bb:cc:dd:ee:01",
		NVRAM: map[string]string{"interface": "0"},
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	stdout, stderr, exitCode := env.runCLI(t, "set", "aa:bb:cc:dd:ee:01",
		"--server-address", "10.20.30.40", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	// The CLI must not inject interface= into the user's merged map
	// when the device already reports a configured value. The wire-
	// side RMW preserves the device's interface=0 (exercised by
	// existing set/RMW tests); here we lock in the inverse — no
	// inference fires, so stdout's echo of the user's merged map
	// contains no interface= line at all.
	if strings.Contains(stdout, "interface=") {
		t.Errorf("no interface injection expected; got stdout:\n%s", stdout)
	}
	if strings.Contains(stderr, "device interface is unset") {
		t.Errorf("no notice expected when device already has a configured value; got stderr:\n%s", stderr)
	}
}
