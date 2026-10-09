# Nil analysis fuzz report

The fuzz tests compare `tgolint` with a small nil-flow oracle. The oracle generates
Boolean guards, early exits, loops, assignments, swaps, saved guards, and later writes.

## Found bugs

1. A contradictory nil guard does not make its branch unreachable. The last comparison
   can replace the first comparison and cause a false error.
2. A parallel assignment can lose a saved proof when it moves the proved value to a new
   variable. This causes a false error after a swap.
3. A parallel assignment rebuilds aliases while it changes assignment targets. A swap
   can make different values share one nil fact. This can hide an unsafe call.
4. A saved comparison loses its known Boolean result after its source variable changes.
   A branch known to be false can report an error.
5. A write to one alias removes a proof from copies that keep the old value. This causes
   a false error for a safe copy.
6. Alias removal changes its map while it searches that map. Map iteration order can
   split the wrong alias group.
7. Alternative branches use variable names instead of value identities. Equivalent
   proofs through two aliases do not survive the branch join.
8. An early branch join collapses related alternatives. A later nil check cannot remove
   an impossible alternative, so the remaining proof is lost.

Each fix has an independent branch from `main`. Its PR adds the smallest counterexample
as a permanent fuzz seed or focused property test.
