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
func ParseINI(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
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
		if _, known := udap.ParameterByName(key); !known {
			return nil, fmt.Errorf("line %d: unknown parameter %q", lineNum, key)
		}
		if err := udap.ValidateParameter(key, value); err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", lineNum, key, err)
		}
		out[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return out, nil
}
