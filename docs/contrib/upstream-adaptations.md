# Upstream adaptations

Keep the upstream copyright header on each adapted source file. Point its
license line to the repository copy of the upstream license. Add each adapted
file to `scripts/check_upstream_provenance.py` and record its source here or in
the linked implementation document.

Run `make upstream-provenance` after you add or move adapted source.

## Control-flow graph builder

`pkg/syntax/cfg` adapts `golang.org/x/tools/go/cfg` from
`golang.org/x/tools v0.51.0`. The repository stores the license at
`third_party/go/LICENSE`.

| Local file | `golang.org/x/tools v0.51.0` source |
| --- | --- |
| `pkg/syntax/cfg/cfg.tgo` | `go/cfg/cfg.go` |
| `pkg/syntax/cfg/builder.tgo` | `go/cfg/builder.go` |

The formatter source map is in
[Go printer port](../implementation/go-printer-port.md).
