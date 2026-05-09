package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"go-udap/udap"
)

// ParseINI parses an INI-style stream of key=value lines.
// Lines starting with # or ; are comments; blank lines are ignored.
// Whitespace around keys and values is trimmed. Values are not quoted.
// Unknown parameter names and values failing udap.ValidateParameter
// produce errors.
//
// If two distinct keys resolve (via udap.ParameterByName, including
// aliases) to the same canonical parameter — for example
// `slimserver_address` and `squeezecenter_address`, both aliases for
// `server_address` — the parser rejects the input rather than letting
// last-write-win on the silently-overlapping NVRAM offset.
func ParseINI(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	canonicalSeenAt := make(map[string]string) // canonical name → first key that produced it
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' separator", lineNum)
		}
		key := strings.TrimSpace(trimmed[:eq])
		value := strings.TrimSpace(trimmed[eq+1:])
		canonical, known := udap.ParameterByName(key)
		if !known {
			return nil, fmt.Errorf("line %d: unknown parameter %q", lineNum, key)
		}
		if err := udap.ValidateParameter(key, value); err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", lineNum, key, err)
		}
		if prev, dup := canonicalSeenAt[canonical.Name]; dup && prev != key {
			return nil, fmt.Errorf("line %d: %q and %q both set the same NVRAM parameter (%s); pick one",
				lineNum, prev, key, canonical.Name)
		}
		canonicalSeenAt[canonical.Name] = key
		out[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return out, nil
}
