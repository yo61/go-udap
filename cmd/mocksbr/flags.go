package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go-udap/mocksbr"
	"go-udap/udap"
)

// deviceOverride holds a parsed --device flag.
type deviceOverride struct {
	idx int
	cfg mocksbr.DeviceConfig
}

// parseDeviceFlags parses a list of --device specs (e.g.
// "idx=2,mac=aa:bb:..,name=foo") into deviceOverride structs.
//
// idx must be in [1, nDevices] and unique across the list. Unrecognized
// keys produce an error.
func parseDeviceFlags(specs []string, nDevices int) ([]deviceOverride, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	seenIdx := make(map[int]bool, len(specs))
	out := make([]deviceOverride, 0, len(specs))
	for _, spec := range specs {
		ov, err := parseOneDeviceSpec(spec, nDevices)
		if err != nil {
			return nil, err
		}
		if seenIdx[ov.idx] {
			return nil, fmt.Errorf("duplicate idx=%d in --device flags", ov.idx)
		}
		seenIdx[ov.idx] = true
		out = append(out, ov)
	}
	return out, nil
}

func parseOneDeviceSpec(spec string, nDevices int) (deviceOverride, error) {
	var ov deviceOverride
	idxSeen := false

	for _, kv := range strings.Split(spec, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			return ov, fmt.Errorf("--device %q: missing '=' in %q", spec, kv)
		}
		key := strings.ToLower(strings.TrimSpace(kv[:eq]))
		val := strings.TrimSpace(kv[eq+1:])
		switch key {
		case "idx":
			n, err := strconv.Atoi(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: idx=%q is not a number", spec, val)
			}
			if n < 1 || n > nDevices {
				return ov, fmt.Errorf("--device %q: idx=%d out of range [1,%d]", spec, n, nDevices)
			}
			ov.idx = n
			idxSeen = true
		case "mac":
			ov.cfg.MAC = val
		case "name":
			ov.cfg.Name = val
		case "model":
			ov.cfg.Model = val
		case "firmware":
			ov.cfg.Firmware = val
		case "hardware":
			ov.cfg.Hardware = val
		case "device-id", "deviceid":
			ov.cfg.DeviceID = val
		case "uuid":
			ov.cfg.UUID = val
		case "reboot":
			d, err := time.ParseDuration(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: reboot=%q is not a duration", spec, val)
			}
			ov.cfg.RebootDelay = d
		case "slow":
			d, err := time.ParseDuration(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: slow=%q is not a duration", spec, val)
			}
			ov.cfg.Slow = d
		case "unreachable":
			b, err := strconv.ParseBool(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: unreachable=%q is not a bool", spec, val)
			}
			ov.cfg.Unreachable = b
		case "fail-on", "failon":
			ops, err := parseFailOn(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: %w", spec, err)
			}
			ov.cfg.FailOn = ops
		case "malformed":
			m, err := parseMalformed(val)
			if err != nil {
				return ov, fmt.Errorf("--device %q: %w", spec, err)
			}
			ov.cfg.Malformed = m
		default:
			return ov, fmt.Errorf("--device %q: unknown key %q", spec, key)
		}
	}
	if !idxSeen {
		return ov, fmt.Errorf("--device %q: idx is required", spec)
	}
	return ov, nil
}

// parseFailOn decodes a fail-on= value into a list of mocksbr.Op. The
// value is a colon-separated list (slashes and pipes work too) of op
// names: discover, get, set, save, reset.
func parseFailOn(val string) ([]mocksbr.Op, error) {
	if val == "" {
		return nil, nil
	}
	splitters := func(r rune) bool { return r == ':' || r == '/' || r == '|' }
	parts := strings.FieldsFunc(val, splitters)
	out := make([]mocksbr.Op, 0, len(parts))
	for _, p := range parts {
		switch strings.ToLower(strings.TrimSpace(p)) {
		case "discover":
			out = append(out, mocksbr.OpDiscover)
		case "get":
			out = append(out, mocksbr.OpGet)
		case "set":
			out = append(out, mocksbr.OpSet)
		case "save":
			out = append(out, mocksbr.OpSave)
		case "reset":
			out = append(out, mocksbr.OpReset)
		default:
			return nil, fmt.Errorf("fail-on: unknown op %q (want discover|get|set|save|reset)", p)
		}
	}
	return out, nil
}

// parseMalformed decodes a malformed= value into a mocksbr.MalformedMode.
func parseMalformed(val string) (mocksbr.MalformedMode, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "", "none":
		return mocksbr.MalformedNone, nil
	case "oversized-count", "oversized_count", "oversizedcount":
		return mocksbr.MalformedOversizedCount, nil
	case "length-exceeds-payload", "length_exceeds_payload", "lengthexceedspayload":
		return mocksbr.MalformedLengthExceedsPayload, nil
	case "unknown-method", "unknown_method", "unknownmethod":
		return mocksbr.MalformedUnknownMethod, nil
	default:
		return mocksbr.MalformedNone, fmt.Errorf("malformed: unknown mode %q", val)
	}
}

// replaceAutoDevice replaces the auto-generated device at position idx
// (1-indexed) with one built from cfg. Identity fields the override
// didn't set inherit the auto-generated values.
func replaceAutoDevice(net *mocksbr.Network, idx int, cfg mocksbr.DeviceConfig) error {
	// Reach into the network through a small public-shaped helper:
	// remove the existing entry at order[idx-1] and Add() the new one.
	mac := net.RemoveAuto(idx)
	if mac == "" {
		return fmt.Errorf("no auto device at idx=%d", idx)
	}
	if cfg.MAC == "" {
		cfg.MAC = mac
	}
	if cfg.UUID == "" {
		cfg.UUID = fmt.Sprintf("mock-sbr-%03d", idx)
	}
	_, err := net.Add(cfg)
	return err
}

// resolveUDPAddr resolves a "host:port" string. Defaults host to
// 0.0.0.0 if the user passes ":17784" or just a port number.
func resolveUDPAddr(s string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp4", s)
}

// bindUDP binds the socket and enables broadcast.
func bindUDP(addr *net.UDPAddr, logger udap.Logger) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, err
	}
	udap.EnableBroadcast(conn, logger)
	return conn, nil
}

// deadlineFromContext returns a near-future deadline (200ms) bounded by
// ctx if ctx has its own. Used to keep the read loop responsive to
// cancellation without busy-waiting.
func deadlineFromContext(ctx interface{ Deadline() (time.Time, bool) }) time.Time {
	d := time.Now().Add(200 * time.Millisecond)
	if ctxD, ok := ctx.Deadline(); ok && ctxD.Before(d) {
		return ctxD
	}
	return d
}

// _ ensures atomic.Int64 isn't dropped by the compiler if added later.
var _ atomic.Int64
