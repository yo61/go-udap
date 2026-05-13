package udap

import (
	"fmt"
	"net"
)

// NetworkConfig is the result of a UCP_METHOD_GET_IP (0x0002) query.
// Returned by Client.GetDeviceNetworkConfigWithContext. Distinct from
// the Device aggregate: Device reflects what discovery passively
// observed; NetworkConfig is what the device reports actively when
// asked.
//
// All fields are optional — devices may omit TLVs (notably Gateway on
// static-IP-without-gateway configurations). Missing fields are zero
// IPs.
type NetworkConfig struct {
	IP         net.IP `json:"ip,omitempty"`
	SubnetMask net.IP `json:"subnet_mask,omitempty"`
	Gateway    net.IP `json:"gateway,omitempty"`
}

// String returns a multi-line representation suitable for CLI output.
func (n NetworkConfig) String() string {
	return fmt.Sprintf("IP:      %s\nSubnet:  %s\nGateway: %s",
		ipOrDash(n.IP), ipOrDash(n.SubnetMask), ipOrDash(n.Gateway))
}

func ipOrDash(ip net.IP) string {
	if len(ip) == 0 || ip.IsUnspecified() {
		return "-"
	}
	return ip.String()
}
