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
6. Alternative branches use variable names instead of value identities. Equivalent
   proofs through two aliases do not survive the branch join.
7. An early branch join collapses related alternatives. A later nil check cannot remove
   an impossible alternative, so the remaining proof is lost.
8. A saved compound guard can lose its known false result after a source value changes.
   Its body then reports an error even though the body cannot run.
9. A saved non-nil proof can lose the old value when its checked variable is overwritten.
   A surviving alias then reports an error even though the saved guard proves it is non-nil.

The fixes use three independent branches from `main`. One fixes nil unions and Boolean
reachability. One fixes parallel assignment identities. One keeps relational branch
alternatives. Each PR adds its smallest counterexamples as permanent fuzz seeds.
