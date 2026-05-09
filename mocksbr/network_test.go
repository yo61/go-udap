package mocksbr

import (
	"testing"

	"go-udap/udap"
)

func TestNewNetworkAutoGeneratesNDevices(t *testing.T) {
	net := NewNetwork(3, udap.NewNoOpLogger())
	if got, want := len(net.devices), 3; got != want {
		t.Errorf("expected %d devices, got %d", want, got)
	}
	if len(net.order) != 3 {
		t.Errorf("expected 3 entries in order, got %d", len(net.order))
	}
}

func TestNetworkAddInsertsDevice(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:ff"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if mac != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected returned MAC to match input, got %q", mac)
	}
	if len(net.devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(net.devices))
	}
}

func TestNetworkAddRejectsBadMAC(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{MAC: "not-a-mac"}); err == nil {
		t.Fatalf("expected error for invalid MAC")
	}
}

func TestNetworkAddRejectsDuplicateMAC(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:ff"}); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:ff"}); err == nil {
		t.Fatalf("expected error for duplicate MAC")
	}
}

func TestNetworkReceiveDiscoveryFansOut(t *testing.T) {
	net := NewNetwork(3, udap.NewNoOpLogger())

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	disc := c.CreateAdvancedDiscoveryPacket()

	replies := net.Receive(disc)
	if len(replies) != 3 {
		t.Fatalf("expected 3 replies (one per device), got %d", len(replies))
	}
	for i, reply := range replies {
		pkt, _, err := udap.ParsePacket(reply)
		if err != nil {
			t.Fatalf("reply %d: parse: %v", i, err)
		}
		if pkt.UCPMethod != udap.MethodAdvDisc {
			t.Errorf("reply %d: UCPMethod=0x%04x, want 0x%04x", i, pkt.UCPMethod, udap.MethodAdvDisc)
		}
		if pkt.UCPFlags != 0x00 {
			t.Errorf("reply %d: UCPFlags=0x%02x, want 0x00 (response)", i, pkt.UCPFlags)
		}
	}
}

func TestNetworkReceiveUnicastByMAC(t *testing.T) {
	net := NewNetwork(3, udap.NewNoOpLogger())
	mac := "00:04:20:00:00:02"

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	dev := &udap.Device{MAC: mac}
	getPkt := c.CreateGetDataPacket(dev, []string{"hostname"})

	replies := net.Receive(getPkt)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	pkt, _, err := udap.ParsePacket(replies[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	gotMAC := formatMAC(pkt.SrcAddress)
	if gotMAC != mac {
		t.Errorf("reply src MAC=%q, want %q", gotMAC, mac)
	}
}

func TestNetworkReceiveUnknownMACReturnsNoReply(t *testing.T) {
	net := NewNetwork(1, udap.NewNoOpLogger())

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	dev := &udap.Device{MAC: "ff:ff:ff:ff:ff:ff"}
	getPkt := c.CreateGetDataPacket(dev, []string{"hostname"})

	replies := net.Receive(getPkt)
	if len(replies) != 0 {
		t.Errorf("expected 0 replies for unknown MAC, got %d", len(replies))
	}
}
