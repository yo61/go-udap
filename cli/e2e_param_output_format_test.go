package cli

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"go-udap/mocksbr"
)

const paramMAC = "00:04:20:00:00:01"

// paramDevice has three values changed from the factory defaults and
// one NVRAM offset no known parameter covers.
var paramDevice = mocksbr.DeviceConfig{
	MAC: paramMAC,
	NVRAM: map[string]string{
		"hostname": "kitchen", "server_address": "10.0.0.7", "interface": "1",
	},
	UnknownOffsets: map[uint16][]byte{900: {0xde, 0xad}},
}

const readAllText = "bridging=0\nhostname=kitchen\ninterface=1\nlan_gateway=0.0.0.0\n" +
	"lan_ip_mode=1\nlan_network_address=0.0.0.0\nlan_subnet_mask=255.255.255.0\n" +
	"lms_address=0.0.0.0\noffset_900=dead\nprimary_dns=0.0.0.0\nsecondary_dns=0.0.0.0\n" +
	"server_address=10.0.0.7\nsqueezecenter_name=\nwireless_SSID=\nwireless_channel=6\n" +
	"wireless_keylen=0\nwireless_mode=0\nwireless_region_id=4\nwireless_wep_key=\n" +
	"wireless_wep_key_1=\nwireless_wep_key_2=\nwireless_wep_key_3=\nwireless_wep_on=0\n" +
	"wireless_wpa_cipher=3\nwireless_wpa_mode=1\nwireless_wpa_on=0\nwireless_wpa_psk=\n"

// jsonObjectKeys returns the keys of a JSON object in document order,
// which decoding into a map would lose.
func jsonObjectKeys(t *testing.T, stdout string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("want a JSON object; got %q (%v)", stdout, err)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("decode key: %v", err)
		}
		key, ok := tok.(string)
		if !ok {
			t.Fatalf("want a string key; got %v", tok)
		}
		keys = append(keys, key)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			t.Fatalf("decode value of %v: %v", tok, err)
		}
	}
	return keys
}

// csvParams turns long-form name,value CSV into its rows after
// checking the header.
func csvParams(t *testing.T, stdout string) [][]string {
	t.Helper()
	records := decodeCSV(t, stdout)
	if len(records) == 0 || strings.Join(records[0], ",") != "name,value" {
		t.Fatalf("want a name,value header; got %q", stdout)
	}
	return records[1:]
}

func TestE2EParamCommandsTextIsByteIdentical(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"read": {
			[]string{"read", paramMAC},
			"hostname=kitchen\ninterface=1\nserver_address=10.0.0.7\n",
		},
		"read --all": {[]string{"read", paramMAC, "--all"}, readAllText},
		"get one":    {[]string{"get", paramMAC, "hostname"}, "kitchen\n"},
		"get several": {
			[]string{"get", paramMAC, "server_address", "hostname"},
			"server_address=10.0.0.7\nhostname=kitchen\n",
		},
		"get alias": {[]string{"get", paramMAC, "squeezecenter_address"}, "10.0.0.7\n"},
		"set": {
			[]string{"set", paramMAC, "--lan-ip-mode", "0", "--hostname", "den"},
			"hostname=den\nlan_ip_mode=0\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{tc.args, append(tc.args, "-o", "text")} {
				env := startMockNetwork(t, paramDevice)
				if got := runOK(t, env, args...); got != tc.want {
					t.Errorf("%v stdout:\n got %q\nwant %q", args, got, tc.want)
				}
			}
		})
	}
}

func TestE2EReadOutputRoundTripsThroughSet(t *testing.T) {
	env := startMockNetwork(t, paramDevice)
	backup := runOK(t, env, "read", paramMAC)
	prevReader, prevPiped := stdinReader, stdinIsPiped
	stdinReader = strings.NewReader(backup)
	stdinIsPiped = func() bool { return true }
	t.Cleanup(func() { stdinReader, stdinIsPiped = prevReader, prevPiped })

	if got := runOK(t, env, "set", paramMAC, "--config", "-"); got != backup {
		t.Errorf("set echo:\n got %q\nwant the read output %q", got, backup)
	}
}

func TestE2EReadJSONAndCSV(t *testing.T) {
	env := startMockNetwork(t, paramDevice)

	var got map[string]string
	decodeJSON(t, runOK(t, env, "read", paramMAC, "--json"), &got)
	assertEqual(t, "json", got, map[string]string{
		"hostname": "kitchen", "interface": "1", "server_address": "10.0.0.7",
	})

	assertEqual(t, "csv", csvParams(t, runOK(t, env, "read", paramMAC, "-o", "csv")), [][]string{
		{"hostname", "kitchen"}, {"interface", "1"}, {"server_address", "10.0.0.7"},
	})
}

func TestE2EReadAllJSONAndCSVIncludeUnknownOffsets(t *testing.T) {
	env := startMockNetwork(t, paramDevice)
	var wantKeys []string
	var wantRows [][]string
	wantValues := map[string]string{}
	for _, line := range strings.Split(strings.TrimSuffix(readAllText, "\n"), "\n") {
		name, value, _ := strings.Cut(line, "=")
		wantKeys = append(wantKeys, name)
		wantRows = append(wantRows, []string{name, value})
		wantValues[name] = value
	}

	stdout := runOK(t, env, "read", paramMAC, "--all", "--json")
	var got map[string]string
	decodeJSON(t, stdout, &got)
	assertEqual(t, "json", got, wantValues)
	assertEqual(t, "json key order", jsonObjectKeys(t, stdout), wantKeys)

	assertEqual(t, "csv", csvParams(t, runOK(t, env, "read", paramMAC, "-a", "-o", "csv")), wantRows)
}

func TestE2EReadWithNothingChangedIsAnEmptyObject(t *testing.T) {
	env := startMockNetwork(t, mocksbr.DeviceConfig{MAC: paramMAC})
	for format, want := range map[string]string{"text": "", "json": "{}\n", "csv": "name,value\n"} {
		if got := runOK(t, env, "read", paramMAC, "-o", format); got != want {
			t.Errorf("%s stdout: got %q, want %q", format, got, want)
		}
	}
}

func TestE2EGetJSONAndCSVKeepRequestOrder(t *testing.T) {
	env := startMockNetwork(t, paramDevice)
	request := []string{"server_address", "hostname", "squeezecenter_name", "lan_ip_mode"}
	args := slices.Concat([]string{"get", paramMAC}, request)

	stdout := runOK(t, env, slices.Concat(args, []string{"--json"})...)
	var got map[string]string
	decodeJSON(t, stdout, &got)
	assertEqual(t, "json", got, map[string]string{
		"server_address": "10.0.0.7", "hostname": "kitchen",
		"squeezecenter_name": "", "lan_ip_mode": "1",
	})
	assertEqual(t, "json key order", jsonObjectKeys(t, stdout), request)

	csvOut := runOK(t, env, slices.Concat(args, []string{"-o", "csv"})...)
	assertEqual(t, "csv", csvParams(t, csvOut), [][]string{
		{"server_address", "10.0.0.7"}, {"hostname", "kitchen"},
		{"squeezecenter_name", ""}, {"lan_ip_mode", "1"},
	})
}

func TestE2EGetOneParamIsStillAnObject(t *testing.T) {
	env := startMockNetwork(t, paramDevice)
	got := runOK(t, env, "get", paramMAC, "hostname", "--json")
	if want := `{"hostname":"kitchen"}` + "\n"; got != want {
		t.Errorf("json: got %q, want %q", got, want)
	}
	got = runOK(t, env, "get", paramMAC, "squeezecenter_address", "-o", "csv")
	if want := "name,value\nsqueezecenter_address,10.0.0.7\n"; got != want {
		t.Errorf("csv: got %q, want %q", got, want)
	}
}

func TestE2EGetRepeatedParameterIsUsageError(t *testing.T) {
	cases := map[string]struct {
		params  []string
		message string
	}{
		"same name": {
			[]string{"hostname", "lan_ip_mode", "hostname"},
			`get: parameter "hostname" requested twice`,
		},
		"alias and canonical": {
			[]string{"squeezecenter_address", "server_address"},
			`get: "squeezecenter_address" and "server_address" are the same parameter`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env := startMockNetwork(t, paramDevice)
			args := append([]string{"get", paramMAC}, tc.params...)
			stdout, stderr, exitCode := env.runCLI(t, append(args, "--timeout", "200ms")...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2", exitCode)
			}
			if stderr != "error: "+tc.message+"\n" {
				t.Errorf("stderr:\n got %q\nwant %q", stderr, "error: "+tc.message+"\n")
			}
			if stdout != "" {
				t.Errorf("stdout should be empty; got %q", stdout)
			}
		})
	}
}

func TestE2ESetJSONAndCSVEchoTheWrittenParameters(t *testing.T) {
	args := []string{"set", paramMAC, "--lan-ip-mode", "0", "--hostname", "den"}

	env := startMockNetwork(t, paramDevice)
	var got map[string]string
	stdout := runOK(t, env, slices.Concat(args, []string{"--json"})...)
	decodeJSON(t, stdout, &got)
	assertEqual(t, "json", got, map[string]string{"hostname": "den", "lan_ip_mode": "0"})
	assertEqual(t, "json key order", jsonObjectKeys(t, stdout), []string{"hostname", "lan_ip_mode"})

	env = startMockNetwork(t, paramDevice)
	csvOut := runOK(t, env, slices.Concat(args, []string{"-o", "csv"})...)
	assertEqual(t, "csv", csvParams(t, csvOut), [][]string{
		{"hostname", "den"}, {"lan_ip_mode", "0"},
	})
}
