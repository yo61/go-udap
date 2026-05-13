package cli

import (
	"context"
	"fmt"
	"io"

	"go-udap/udap"
)

// maybeFillUUID populates device.UUID via the get_uuid UCP method
// when discovery didn't include TLV 0x0d. Older Squeezebox firmware
// omits UUID from adv_discover but answers get_uuid correctly.
//
// Soft-fails: a transport error, MethodError, or timeout just leaves
// device.UUID empty (formatDeviceInfo will then omit the UUID line).
// The diagnostic is gated behind verbose so the default --info output
// stays clean.
//
// Mutates device through its live pointer (see the live-pointer
// contract documented on udap.Client device accessors).
func maybeFillUUID(ctx context.Context, client *udap.Client, device *udap.Device, verbose bool, stderr io.Writer) {
	if device.UUID != "" {
		return
	}
	uuid, err := client.GetDeviceUUIDWithContext(ctx, device)
	if err != nil {
		if verbose {
			fmt.Fprintf(stderr, "warning: get_uuid fallback failed for %s: %v\n", device.MAC, err)
		}
		return
	}
	device.UUID = uuid
}
