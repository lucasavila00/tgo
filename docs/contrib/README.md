# Contributor documentation

- [Feature documentation](feature-documentation.md): feature inventory and the
  required documentation updates for each change.
- [Compiler lowering](compiler-lowering.md): semantic priorities, review rules,
  and the current missing-lowering audit.
- [Upstream adaptations](upstream-adaptations.md): attribution and source maps
  for code that TGo adapts from other projects.

Architecture-specific contributor guides belong in this directory. User
behavior belongs in the specification or user guides.

Run `make source-size` after you add or split handwritten Go or TGo source.
The check enforces the 700-line limit and validates each fixture exclusion.
