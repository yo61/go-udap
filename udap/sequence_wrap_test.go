package udap

import (
	"testing"
)

// TestSequenceWrapsAtUint16Boundary locks in the wire-field wrap
// behaviour for review finding #7. The internal sequence counter is
// uint32 (so it never overflows in any realistic Client lifetime),
// but the on-the-wire field is uint16 (per the UDAP packet header
// definition). The cast at packet build time must therefore wrap
// modulo 65536.
//
// Pre-fix the wrap was correct but only by virtue of an implicit
// uint32→uint16 truncation, with no test or comment to lock the
// behaviour in. A regression that switched the field type or
// dropped the cast would silently break interop with any device or
// peer expecting the canonical uint16 representation.
func TestSequenceWrapsAtUint16Boundary(t *testing.T) {
	cases := []struct {
		name      string
		startFrom uint32
		want      []uint16
	}{
		{
			name:      "fresh client",
			startFrom: 0,
			want:      []uint16{1, 2, 3},
		},
		{
			name:      "approaching wrap",
			startFrom: 65533,
			want:      []uint16{65534, 65535, 0, 1},
		},
		{
			name:      "after wrap",
			startFrom: 0x10000,
			want:      []uint16{1, 2, 3},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{logger: NewNoOpLogger(), sequence: tc.startFrom}
			for i, want := range tc.want {
				raw := c.CreateAdvancedDiscoveryPacket()
				parsed, _, err := ParsePacket(raw)
				if err != nil {
					t.Fatalf("call %d: ParsePacket: %v", i, err)
				}
				if parsed.Sequence != want {
					t.Errorf("call %d: Sequence=%d, want %d", i, parsed.Sequence, want)
				}
			}
		})
	}
}
