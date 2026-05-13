package udap

import (
	"net"
	"testing"
)

func TestComputeDirectedBroadcast(t *testing.T) {
	cases := []struct {
		name string
		addr net.IP
		mask net.IPMask
		want net.IP
	}{
		{"slash24", net.IPv4(192, 168, 1, 50), net.CIDRMask(24, 32), net.IPv4(192, 168, 1, 255)},
		{"slash16", net.IPv4(10, 1, 2, 3), net.CIDRMask(16, 32), net.IPv4(10, 1, 255, 255)},
		{"slash22", net.IPv4(172, 16, 5, 7), net.CIDRMask(22, 32), net.IPv4(172, 16, 7, 255)},
		{"slash30", net.IPv4(192, 168, 1, 5), net.CIDRMask(30, 32), net.IPv4(192, 168, 1, 7)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeDirectedBroadcast(tc.addr, tc.mask)
			if !got.Equal(tc.want) {
				t.Errorf("computeDirectedBroadcast = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnumerateInterfacesAllPassFilter(t *testing.T) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		t.Skipf("EnumerateInterfaces error (likely permissions/CI): %v", err)
	}
	if len(ifs) == 0 {
		t.Skip("no usable interfaces on this host")
	}
	for _, ni := range ifs {
		if ni.Name == "" {
			t.Errorf("Name is empty")
		}
		if ni.Addr == nil {
			t.Errorf("%s has nil Addr", ni.Name)
		}
		if ni.Addr.To4() == nil {
			t.Errorf("%s Addr is not IPv4: %v", ni.Name, ni.Addr)
		}
		if ni.Broadcast == nil {
			t.Errorf("%s has nil Broadcast", ni.Name)
		}
	}
}
