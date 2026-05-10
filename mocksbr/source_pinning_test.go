package mocksbr

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"go-udap/udap"
)

// TestClientRejectsForgedReplyFromUnexpectedSrc and
// TestClientAcceptsReplyFromExpectedSrc lock in the fix for review
// finding #6 — udap.Client.waitForDeviceReply used to discard the
// transport-provided source identifier (an IP for UDPTransport, an
// arbitrary string for MockTransport) and trust the in-payload
// SrcAddress alone. A LAN attacker who spoofed the device's MAC in a
// forged UCP reply would have it accepted.
//
// Post-fix, the source from Recv is matched against device.IP; only
// matching replies are forwarded.

func TestClientRejectsForgedReplyFromUnexpectedSrc(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	transport := NewMockTransport(net)
	client := udap.NewClientWithTransport(transport, udap.NewNoOpLogger())
	defer client.Close()

	const mac = "aa:bb:cc:dd:ee:01"
	const expectedSrc = "src-of-real-device"
	const spoofedSrc = "src-of-attacker"
	dev := &udap.Device{MAC: mac, IP: expectedSrc}

	transport.InjectReply(forgeGetDataReply(t, mac), spoofedSrc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})

	if err == nil {
		t.Fatalf("expected error from spoofed-src reply, got success")
	}
}

func TestClientAcceptsReplyFromExpectedSrc(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	transport := NewMockTransport(net)
	client := udap.NewClientWithTransport(transport, udap.NewNoOpLogger())
	defer client.Close()

	const mac = "aa:bb:cc:dd:ee:01"
	const expectedSrc = "src-of-real-device"
	dev := &udap.Device{MAC: mac, IP: expectedSrc}

	transport.InjectReply(forgeGetDataReply(t, mac), expectedSrc)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})

	if err != nil {
		t.Fatalf("expected reply accepted, got error: %v", err)
	}
}

// forgeGetDataReply constructs a minimal valid GetData response: a
// 27-byte UDAP header naming srcMAC as the source, plus a 2-byte
// uint16 BE count of zero items. Enough to pass ParsePacket and the
// MethodGetData branch of GetDeviceConfigWithContext.
func forgeGetDataReply(t *testing.T, srcMAC string) []byte {
	t.Helper()
	var src [6]byte
	if _, err := fmt.Sscanf(srcMAC, "%02x:%02x:%02x:%02x:%02x:%02x",
		&src[0], &src[1], &src[2], &src[3], &src[4], &src[5]); err != nil {
		t.Fatalf("parse MAC: %v", err)
	}
	pkt := udap.Packet{
		DstType:    udap.AddrTypeETH,
		SrcType:    udap.AddrTypeETH,
		SrcAddress: src,
		UDAPType:   udap.TypeUCP,
		UAPClass:   [4]byte{0x00, 0x01, 0x00, 0x01},
		UCPMethod:  udap.MethodGetData,
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, pkt); err != nil {
		t.Fatalf("encode header: %v", err)
	}
	if err := binary.Write(&buf, binary.BigEndian, uint16(0)); err != nil {
		t.Fatalf("encode count: %v", err)
	}
	return buf.Bytes()
}
