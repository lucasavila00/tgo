# CFG source map

`pkg/syntax/cfg` adapts the control-flow graph implementation from
`golang.org/x/tools/go/cfg` at `golang.org/x/tools v0.51.0`. Adapted files keep
the Go copyright header. `third_party/go/LICENSE` contains the BSD license text.

| Local file | `golang.org/x/tools v0.51.0` source |
| --- | --- |
| `pkg/syntax/cfg/cfg.go` | `go/cfg/cfg.go` |
| `pkg/syntax/cfg/builder.tgo` | `go/cfg/builder.go` |

The local implementation uses the public TGo syntax tree. This source map
records provenance and does not change CFG behavior.
