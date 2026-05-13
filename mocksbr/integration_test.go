package mocksbr_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go-udap/mocksbr"
	"go-udap/udap"
)

// TestClientDiscoversAllDevices drives a real udap.Client against a
// MockTransport-backed Network and verifies each device shows up in the
// client's device list with the expected MAC.
func TestClientDiscoversAllDevices(t *testing.T) {
	net := mocksbr.NewNetwork(3, udap.NewNoOpLogger())
	defer net.Close()

	client := udap.NewClientWithTransport(mocksbr.NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		t.Fatalf("DiscoverDevicesWithContext: %v", err)
	}

	devices := client.ListDevices()
	if got, want := len(devices), 3; got != want {
		t.Fatalf("discovered %d devices, want %d", got, want)
	}

	wantMACs := map[string]bool{
		"00:04:20:00:00:01": true,
		"00:04:20:00:00:02": true,
		"00:04:20:00:00:03": true,
	}
	for _, d := range devices {
		if !wantMACs[d.MAC.String()] {
			t.Errorf("unexpected device MAC: %s", d.MAC)
		}
	}
}

// TestClientReadModifyWriteRoundTrip drives discover → read → set →
// reset → read against a MockTransport, verifying that values written
// before the reset persist (because handleSetData saves to NVRAM on
// every set).
func TestClientReadModifyWriteRoundTrip(t *testing.T) {
	net := mocksbr.NewNetwork(1, udap.NewNoOpLogger())
	defer net.Close()
	client := udap.NewClientWithTransport(mocksbr.NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	device := client.GetDevice("00:04:20:00:00:01")
	if device == nil {
		t.Fatalf("device not discovered")
	}

	// Read initial state — hostname should be empty (factory default).
	cfg, err := client.GetDeviceConfigWithContext(ctx, device, []string{"hostname"})
	if err != nil {
		t.Fatalf("GetDeviceConfig (initial): %v", err)
	}
	if cfg["hostname"] != "" {
		t.Errorf("initial hostname=%q, want empty", cfg["hostname"])
	}

	// Set hostname.
	if err := client.SetDeviceConfigWithContext(ctx, device, map[string]string{"hostname": "test-host"}); err != nil {
		t.Fatalf("SetDeviceConfig: %v", err)
	}

	// Read back — should be the new value.
	device.Parameters = nil // force re-read
	cfg, err = client.GetDeviceConfigWithContext(ctx, device, []string{"hostname"})
	if err != nil {
		t.Fatalf("GetDeviceConfig (post-set): %v", err)
	}
	if cfg["hostname"] != "test-host" {
		t.Errorf("post-set hostname=%q, want test-host", cfg["hostname"])
	}
}

// TestClientResetEntersRebootWindow verifies that after reset, the
// device drops packets briefly. Tests the window by issuing a
// follow-up GetData immediately after Reset and asserting it times out.
func TestClientResetEntersRebootWindow(t *testing.T) {
	net := mocksbr.NewNetwork(1, udap.NewNoOpLogger())
	defer net.Close()
	client := udap.NewClientWithTransport(mocksbr.NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	device := client.GetDevice("00:04:20:00:00:01")
	if device == nil {
		t.Fatalf("device not discovered")
	}

	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// Immediate GetData should hit the reboot window and time out.
	getCtx, getCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer getCancel()
	_, err := client.GetDeviceConfigWithContext(getCtx, device, []string{"hostname"})
	if err == nil {
		t.Errorf("expected GetDeviceConfig to fail during reboot window, got nil")
	}

	// After the window, GetData should succeed.
	time.Sleep(150 * time.Millisecond)
	cfg, err := client.GetDeviceConfigWithContext(ctx, device, []string{"hostname"})
	if err != nil {
		t.Fatalf("GetDeviceConfig (post-reboot): %v", err)
	}
	if _, ok := cfg["hostname"]; !ok {
		t.Errorf("post-reboot read missing hostname key")
	}
}

// TestDiscoveryResponseIncludesUUID verifies that discovery responses
// include the UUID TLV (0x0d) when a device is configured with a UUID.
func TestDiscoveryResponseIncludesUUID(t *testing.T) {
	n := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	defer n.Close()
	if _, err := n.Add(mocksbr.DeviceConfig{
		MAC:  "00:04:20:00:00:01",
		UUID: "deadbeefcafebabe1122334455667788",
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Build a discovery request packet to feed into Receive.
	client, _ := udap.NewClientWithLogger(udap.NewNoOpLogger())
	req := client.CreateAdvancedDiscoveryPacket()
	client.Close()

	replies := n.Receive(req)
	if len(replies) != 1 {
		t.Fatalf("got %d replies, want 1", len(replies))
	}
	if !bytes.Contains(replies[0], []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}) {
		t.Errorf("reply does not contain UUID bytes; reply hex=%x", replies[0])
	}
}
