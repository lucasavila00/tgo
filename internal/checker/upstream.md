# Upstream source

The private checker and compiled-package reader use Go 1.27.1 source from
`go/types` and `go/internal/gcimporter`. Their required internal packages are
copied under `upstream/`. The Go license is in `LICENSE`.

Imports refer to the private packages. Upstream tests, generation entry points,
and public `go/types` compatibility link names are omitted. TGo changes are
checked in with this source. Keep the checker and reader on the same Go version.
