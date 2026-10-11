# Check source before lowering

The [driver](../../arch/transpiler/readme.md) writes original TGo at final
generated Go paths and asks Go to select packages. The checker then:

1. Reads selected files. The frontend parses TGo and records operation markers;
   ordinary Go uses Go's parser.
2. Checks source packages with the private checker, checking dependencies as
   needed and caching each package ID. Test variants have separate IDs.
3. Reads external dependency types from compiled export data with the matching
   private importer.
4. Gives propagation its success types. Checks final error results and the
   enclosing function's error return. Uses the original error type for `!` and
   wrapped `error` for `!!`.
5. Checks if branches in the enclosing source function, including surrounding
   types, scope, and jumps.

No helper declaration or package reload is needed for each operator. The
frontend's placeholders do not introduce new source functions.

Pass checked syntax, types, bindings, and operation records to lowering. The
driver writes the final ordinary Go, then stock Go checks it. Compilation
returns no file contents.

Generate the checker and export reader from the same pinned Go version and
tracked patches. See [upstream preparation](../../../internal/checker/upstream.md).
Public `Load` keeps stock objects and checks the same ordinary syntax separately
with private types for lowering.

[WHAT](what.md) explains the decision. [PROOF](proof.md) defines its checks.
