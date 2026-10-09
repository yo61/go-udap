package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var getCmd = &cobra.Command{
	Use:   "get MAC PARAM [PARAM...]",
	Short: "Read specific parameters",
	Long: `Read one or more named NVRAM parameters from a device. Unlike "read",
get only fetches the parameters you ask for and rejects unknown names
up front (exit 2), as it does a parameter named twice.

One parameter prints its bare value; several print "name=value" lines
in request order. --format json writes one object, in request order
even for one parameter, and csv writes name,value rows.

Use the canonical wire name (e.g. lan_ip_mode, server_address). Aliases
such as squeezecenter_address are also accepted. Run "go-udap read --all"
to see the full list of parameters a device understands.`,
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeParameterNames,
	RunE:              runGet,
	Annotations:       map[string]string{annotationResult: ""},
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: exitUsage, Err: err}
	}
	params := args[1:]
	canonical, err := canonicalParamNames(params)
	if err != nil {
		return &ExitError{Code: exitUsage, Err: err}
	}

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: exitFailure, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stop := startProgress(stderr, "get", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	values, err := client.GetDeviceConfigWithContext(ctx, device, params)
	if err != nil {
		return deviceOpError("get", mac, timeout, err)
	}
	stop()
	result := getResult{make(paramList, 0, len(params))}
	for _, p := range params {
		result.paramList = append(result.paramList, paramValue{name: p, value: values[canonical[p]]})
	}
	return renderResult(cmd, result)
}

// canonicalParamNames maps each requested name to its canonical
// parameter name. Unknown names, and two names for one parameter, are
// errors.
func canonicalParamNames(params []string) (map[string]string, error) {
	canonical := make(map[string]string, len(params))
	requestedAs := make(map[string]string, len(params))
	for _, p := range params {
		param, ok := udap.ParameterByName(p)
		if !ok {
			return nil, fmt.Errorf("get: unknown parameter %q", p)
		}
		if first, seen := requestedAs[param.Name]; seen {
			if first == p {
				return nil, fmt.Errorf("get: parameter %q requested twice", p)
			}
			return nil, fmt.Errorf("get: %q and %q are the same parameter", first, p)
		}
		requestedAs[param.Name] = p
		canonical[p] = param.Name
	}
	return canonical, nil
}

// getResult is the Result of `get`: the requested parameters in
// request order. One parameter's text is its bare value.
type getResult struct{ paramList }

func (r getResult) WriteText(w io.Writer) error {
	if len(r.paramList) == 1 {
		_, err := fmt.Fprintln(w, r.paramList[0].value)
		return err
	}
	return r.paramList.WriteText(w)
}
