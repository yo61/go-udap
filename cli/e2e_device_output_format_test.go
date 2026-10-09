package cli

import (
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go-udap/mocksbr"
)

// configuredDevice answers every query; partialDevice has no UUID and
// drops get_ip, so its uuid and network are absent; noGatewayDevice
// answers get_ip without a gateway.
var (
	configuredDevice = mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "192.168.1.50",
		SubnetMask: "255.255.255.0",
		Gateway:    "192.168.1.1",
		UUID:       "deadbeefcafebabe1122334455667788",
	}
	partialDevice   = mocksbr.DeviceConfig{MAC: "00:04:20:00:00:02", DropGetIP: true}
	noGatewayDevice = mocksbr.DeviceConfig{
		MAC: "00:04:20:00:00:03", IP: "10.0.0.7", SubnetMask: "255.0.0.0",
		UUID: "00112233445566778899aabbccddeeff",
	}
)

func runOK(t *testing.T, env *e2eEnv, args ...string) string {
	t.Helper()
	stdout, stderr, exitCode := env.runCLI(t, append(args, "--timeout", "300ms")...)
	if exitCode != 0 {
		t.Fatalf("%v: exit code %d, want 0; stderr:\n%s", args, exitCode, stderr)
	}
	return stdout
}

func decodeJSON(t *testing.T, stdout string, into any) {
	t.Helper()
	if !strings.HasSuffix(stdout, "}\n") && !strings.HasSuffix(stdout, "]\n") {
		t.Errorf("JSON should be one compact value with a trailing newline; got %q", stdout)
	}
	if err := json.Unmarshal([]byte(stdout), into); err != nil {
		t.Fatalf("decode JSON: %v; stdout:\n%s", err, stdout)
	}
}

func decodeCSV(t *testing.T, stdout string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
	if err != nil {
		t.Fatalf("decode CSV: %v; stdout:\n%s", err, stdout)
	}
	return records
}

func assertEqual(t *testing.T, what string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %#v\nwant %#v", what, got, want)
	}
}

// deviceJSON is a mock device's metadata. mocksbr reports the reply's
// source address, which is the device's MAC, as its IP.
func deviceJSON(mac, uuid any) map[string]any {
	return map[string]any{
		"mac": mac, "ip": mac, "name": "Squeezebox Device", "model": "Squeezebox Receiver",
		"firmware": "77", "hardware_rev": "0005", "uuid": uuid, "state": "init",
	}
}

func TestE2EDeviceCommandsTextIsByteIdentical(t *testing.T) {
	const configuredInfo = "MAC:      00:04:20:00:00:01\nIP:       00:04:20:00:00:01\n" +
		"Name:     Squeezebox Device\nModel:    Squeezebox Receiver\nFirmware: 77\n" +
		"HW Rev:   0005\nUUID:     deadbeefcafebabe1122334455667788\nState:    init\n"
	const partialInfo = "MAC:      00:04:20:00:00:02\nIP:       00:04:20:00:00:02\n" +
		"Name:     Squeezebox Device\nModel:    Squeezebox Receiver\nFirmware: 77\n" +
		"HW Rev:   0005\nState:    init\n"
	const configuredNetwork = "IP:      192.168.1.50\nSubnet:  255.255.255.0\nGateway: 192.168.1.1\n"
	const absentNetwork = "IP:      -\nSubnet:  -\nGateway: -\n"
	cases := map[string]struct {
		args []string
		want string
	}{
		"discover": {[]string{"discover"}, "00:04:20:00:00:01\n00:04:20:00:00:02\n"},
		"discover --info": {
			[]string{"discover", "--info"},
			configuredInfo + configuredNetwork + "\n" + partialInfo + absentNetwork,
		},
		"info":  {[]string{"info", "00:04:20:00:00:02"}, partialInfo},
		"getip": {[]string{"getip", "00:04:20:00:00:01"}, configuredNetwork},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env := startMockNetwork(t, configuredDevice, partialDevice)
			for _, args := range [][]string{tc.args, append(tc.args, "-o", "text")} {
				if got := runOK(t, env, args...); got != tc.want {
					t.Errorf("%v stdout:\n got %q\nwant %q", args, got, tc.want)
				}
			}
		})
	}
}

func TestE2EDiscoverJSONAndCSV(t *testing.T) {
	env := startMockNetwork(t, configuredDevice, partialDevice)

	var got []map[string]any
	decodeJSON(t, runOK(t, env, "discover", "--json"), &got)
	assertEqual(t, "json", got, []map[string]any{
		{"mac": "00:04:20:00:00:01"},
		{"mac": "00:04:20:00:00:02"},
	})

	assertEqual(t, "csv", decodeCSV(t, runOK(t, env, "discover", "-o", "csv")), [][]string{
		{"mac"},
		{"00:04:20:00:00:01"},
		{"00:04:20:00:00:02"},
	})
}

func TestE2EDiscoverInfoJSONNestsNetwork(t *testing.T) {
	env := startMockNetwork(t, configuredDevice, partialDevice, noGatewayDevice)
	var got []map[string]any
	decodeJSON(t, runOK(t, env, "discover", "--info", "--json"), &got)

	configured := deviceJSON("00:04:20:00:00:01", "deadbeefcafebabe1122334455667788")
	configured["network"] = map[string]any{
		"ip": "192.168.1.50", "subnet_mask": "255.255.255.0", "gateway": "192.168.1.1",
	}
	partial := deviceJSON("00:04:20:00:00:02", nil)
	partial["network"] = nil
	noGateway := deviceJSON("00:04:20:00:00:03", "00112233445566778899aabbccddeeff")
	noGateway["network"] = map[string]any{"ip": "10.0.0.7", "subnet_mask": "255.0.0.0", "gateway": nil}
	assertEqual(t, "json", got, []map[string]any{configured, partial, noGateway})
}

func TestE2EDiscoverInfoCSVFlattensNetwork(t *testing.T) {
	env := startMockNetwork(t, configuredDevice, partialDevice)
	got := decodeCSV(t, runOK(t, env, "discover", "--info", "-o", "csv"))
	assertEqual(t, "csv", got, [][]string{
		{"mac", "ip", "name", "model", "firmware", "hardware_rev", "uuid", "state",
			"network_ip", "network_subnet_mask", "network_gateway"},
		{"00:04:20:00:00:01", "00:04:20:00:00:01", "Squeezebox Device", "Squeezebox Receiver",
			"77", "0005", "deadbeefcafebabe1122334455667788", "init",
			"192.168.1.50", "255.255.255.0", "192.168.1.1"},
		{"00:04:20:00:00:02", "00:04:20:00:00:02", "Squeezebox Device", "Squeezebox Receiver",
			"77", "0005", "", "init", "", "", ""},
	})
}

func TestE2EInfoJSONAndCSV(t *testing.T) {
	env := startMockNetwork(t, configuredDevice, partialDevice)

	var got map[string]any
	decodeJSON(t, runOK(t, env, "info", "00:04:20:00:00:01", "--json"), &got)
	assertEqual(t, "json", got, deviceJSON("00:04:20:00:00:01", "deadbeefcafebabe1122334455667788"))

	got = nil
	decodeJSON(t, runOK(t, env, "info", "00:04:20:00:00:02", "--json"), &got)
	assertEqual(t, "json without uuid", got, deviceJSON("00:04:20:00:00:02", nil))

	csvOut := runOK(t, env, "info", "00:04:20:00:00:01", "-o", "csv")
	assertEqual(t, "csv", decodeCSV(t, csvOut), [][]string{
		{"mac", "ip", "name", "model", "firmware", "hardware_rev", "uuid", "state"},
		{"00:04:20:00:00:01", "00:04:20:00:00:01", "Squeezebox Device", "Squeezebox Receiver",
			"77", "0005", "deadbeefcafebabe1122334455667788", "init"},
	})
}

func TestE2EGetIPJSONAndCSV(t *testing.T) {
	env := startMockNetwork(t, configuredDevice, noGatewayDevice)

	var got map[string]any
	decodeJSON(t, runOK(t, env, "getip", "00:04:20:00:00:01", "--json"), &got)
	assertEqual(t, "json", got, map[string]any{
		"ip": "192.168.1.50", "subnet_mask": "255.255.255.0", "gateway": "192.168.1.1",
	})

	decodeJSON(t, runOK(t, env, "getip", "00:04:20:00:00:03", "-o", "json"), &got)
	assertEqual(t, "json without gateway", got, map[string]any{
		"ip": "10.0.0.7", "subnet_mask": "255.0.0.0", "gateway": nil,
	})

	csvOut := runOK(t, env, "getip", "00:04:20:00:00:03", "-o", "csv")
	assertEqual(t, "csv", decodeCSV(t, csvOut), [][]string{
		{"ip", "subnet_mask", "gateway"},
		{"10.0.0.7", "255.0.0.0", ""},
	})
}

func TestE2EDiscoverEmptyResult(t *testing.T) {
	const infoHeader = "mac,ip,name,model,firmware,hardware_rev,uuid,state," +
		"network_ip,network_subnet_mask,network_gateway\n"
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"discover"}, ""},
		{[]string{"discover", "--json"}, "[]\n"},
		{[]string{"discover", "-o", "csv"}, "mac\n"},
		{[]string{"discover", "--info"}, ""},
		{[]string{"discover", "--info", "--json"}, "[]\n"},
		{[]string{"discover", "--info", "-o", "csv"}, infoHeader},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			env := startMockNetwork(t)
			stdout, stderr, exitCode := env.runCLI(t, append(tc.args, "--timeout", "200ms")...)
			if exitCode != 0 {
				t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
			}
			if stdout != tc.want {
				t.Errorf("stdout: got %q, want %q", stdout, tc.want)
			}
			if stderr != "no devices found within 200ms\n" {
				t.Errorf("stderr: got %q", stderr)
			}
		})
	}
}

func TestE2EDeviceCommandFailureWritesNoStructuredOutput(t *testing.T) {
	env := startMockNetwork(t, partialDevice)
	cases := []struct {
		args  []string
		cause string
	}{
		{[]string{"info", "aa:bb:cc:dd:ee:ff", "--json"}, "device aa:bb:cc:dd:ee:ff not found"},
		{[]string{"getip", "00:04:20:00:00:02", "-o", "csv"}, "getip: no reply from 00:04:20:00:00:02"},
	}
	for _, tc := range cases {
		t.Run(tc.args[0], func(t *testing.T) {
			stdout, stderr, exitCode := env.runCLI(t, append(tc.args, "--timeout", "200ms")...)
			if exitCode != 1 {
				t.Errorf("exit code %d, want 1", exitCode)
			}
			if !strings.Contains(stderr, tc.cause) {
				t.Errorf("stderr missing %q; got:\n%s", tc.cause, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout should be empty; got %q", stdout)
			}
		})
	}
}
