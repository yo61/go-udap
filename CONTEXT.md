# go-udap

A single-shot CLI that discovers and configures Squeezebox devices over
UDAP. This glossary covers how the CLI presents what it learns.

## Output

**Data command**:
A subcommand whose purpose is to write a Result to stdout (`discover`,
`info`, `getip`, `interfaces`, `read`, `get`, `set`), plus the
`--version` and `--build-info` queries.
_Avoid_: query command, read command (`read` is one specific command)

**Result**:
The information a Data command produced, independent of how it is
written out.
_Avoid_: response (a UDAP wire reply), output, payload

**Output format**:
The representation a Result is written in on stdout: `text`, `json`, or
`csv`. Stderr is never in an Output format.
_Avoid_: output mode, style, encoding

**Build info**:
Facts about how the binary was built (commit, commit time, whether the
working tree was modified, Go version, OS, architecture), reported by
`--build-info`.
_Avoid_: build date (the timestamp is the commit's, not the build's),
version info
