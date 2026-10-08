# Bootstrap policy

Write the TGo compiler in pure Go for now. This keeps bootstrap and implementation work
simple. It also removes the need for TGo backward compatibility during early language work.

Tools that do not affect generated program runtime code can use TGo. This lets tools such as
`tgolint` test TGo on repository code.

Commit each generated `*_tgo.go` file beside its `.tgo` source file.
