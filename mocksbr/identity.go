package mocksbr

import "fmt"

// autoConfig returns the DeviceConfig for the idx'th auto-generated
// virtual device (1-indexed). Identities are deterministic so tests
// can target devices by hardcoded MAC without first calling discover.
//
// MACs follow the OUI 00:04:20 (Slim Devices, used by real Squeezebox
// hardware). The last byte is idx; idx > 255 panics — out-of-range
// at construction is a programmer error, not a runtime condition.
func autoConfig(idx int) DeviceConfig {
	if idx < 1 || idx > 255 {
		panic(fmt.Sprintf("mocksbr: idx %d out of range [1,255]", idx))
	}
	return DeviceConfig{
		MAC:  fmt.Sprintf("00:04:20:00:00:%02x", idx),
		Name: "",
		UUID: fmt.Sprintf("mock-sbr-%03d", idx),
	}
}
