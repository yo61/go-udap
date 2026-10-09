## Decision: A failed `get_ip` makes `discover --info`'s JSON `network` null

When a device doesn't answer `get_ip`, its `network` value is `null`.
When it answers, `network` is an object, and a field is `null` only
when the device left it out (no gateway, or `0.0.0.0` in factory
state).

## Context: PR #258 (#246) first shipped `network` as an object with three null fields on failure

Settled with Robin on 2026-10-09 before the PR merged, while the shape
could still change without a breaking release.

## Alternatives considered:

- **Always an object, with null fields on failure:** rejected. A device
  that never answered looked the same as one that answered in factory
  state.

## Reasoning:

- `null` keeps "no answer" apart from "answered, field absent".
- #246 lists "a failed `get_ip`" among the absent values that are
  `null`, which reads most naturally as `network` itself being `null`.
- jq is unaffected: `.network.ip` is `null` either way.

## Trade-offs accepted:

- Typed consumers need a nil check on `network` before reading its
  fields.
- CSV and text still can't tell the two cases apart: all `network_*`
  cells are empty, and text shows `-`.

## Supersedes: none (refines 2026-10-09-output-format-selection)
