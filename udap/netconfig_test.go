package udap

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
)

func TestNetworkConfigString(t *testing.T) {
	nc := NetworkConfig{
		IP:         net.IPv4(192, 168, 1, 50),
		SubnetMask: net.IPv4(255, 255, 255, 0),
		Gateway:    net.IPv4(192, 168, 1, 1),
	}
	got := nc.String()
	for _, want := range []string{"192.168.1.50", "255.255.255.0", "192.168.1.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q; got %q", want, got)
		}
	}
}

func TestNetworkConfigJSONRoundtrip(t *testing.T) {
	in := NetworkConfig{
		IP:         net.IPv4(10, 0, 0, 1),
		SubnetMask: net.IPv4(255, 0, 0, 0),
		Gateway:    net.IPv4(10, 0, 0, 254),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out NetworkConfig
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !in.IP.Equal(out.IP) || !in.SubnetMask.Equal(out.SubnetMask) || !in.Gateway.Equal(out.Gateway) {
		t.Errorf("roundtrip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestNetworkConfigZeroValueStringIsSafe(t *testing.T) {
	var nc NetworkConfig
	// Must not panic; output content is implementation-defined.
	_ = nc.String()
}
