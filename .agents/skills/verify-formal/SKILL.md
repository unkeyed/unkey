---
name: verify-formal
description: "Runs and extends Unkey's bounded invariant checks across repository areas. Use for formal verification, concurrent state machines, lifecycle guards, changes to modeled source, or /verify-formal. Reports counterexamples, source drift, bounds, and mutation evidence."
---

# Verify formal properties

Use this as the repository-wide verification entry point, not as a skill for one
product feature. Select the relevant scope from
[the model inventory](reference/models.md). The suite includes executable
specifications of deployment admission and circuit-breaker recovery. It does
not include credits.

## Run the suite

From the repository root, run:

```bash
mise run verify-formal
```

The command uses the Go version pinned by mise. It needs no database, network
service, additional dependency, or verifier installation. It enumerates finite
input classes and request schedules against the real Go implementations, not
random samples. It is bounded verification, not a TLA+ proof or a proof of the
whole application.

Use a scope when investigating one contract:

```bash
mise run verify-formal --check deploygate
mise run verify-formal --check circuitbreaker-limit
mise run verify-formal --check circuitbreaker-recovery
```

Add `--json` for case counts and the first counterexample in structured form.
An unknown scope is an error, never an empty successful run.

## Interpret results

Report each scope separately. Zero violations covers only its documented domain
and assumptions. A counterexample is a finding, not permission to fix product
code. Read the referenced implementation and explain the failing sequence before
proposing a fix. Do not weaken a property or hide a failing scope to get green.

The checked-in baseline has an open circuit-breaker probe-limit finding. The
default command therefore exits nonzero. The inventory records the trace and
exclusions. Passing deployment checks do not cancel that finding.

Source fingerprints stop a run when reviewed implementation, contracts, enums,
clock behavior, dependencies, or toolchain configuration change. This is a
review alarm, not a proof. On drift:

1. Read the source diff and the scope's model and assumptions.
2. Update the model only for an intended contract change. Preserve real bugs as
   findings.
3. Update only the reviewed fingerprints in
   `tools/formalverify/sources.sha256`, using `sha256sum <path>`.
4. Run the affected scope, the mutation checks, and then the full verifier.
5. Report any uncovered behavior. Never refresh hashes solely to bypass drift.

## Check that verification can fail

Run the negative controls:

```bash
mise run verify-formal --self-test
```

This compiles temporary Go overlays that alter real product source. It reverses
the deployment readiness guard and delays circuit-breaker recovery by one
success. Each selected scope must pass without the overlay, report a semantic
counterexample with it, and pass again without it. Compile errors, fingerprint
failures, and empty reports do not count as caught mutations. Temporary files
are removed; the checkout is never rewritten.

The probe-limit property already fails on the real implementation. Its failing
schedules remain visible in the normal command. A passing `--self-test` means
the negative controls work, not that all product invariants hold.

## Add a scope

When the requested area has no model, say it is uncovered and create a focused
scope rather than treating existing passes as coverage for it.

1. Read the implementation, closest tests, and applicable repository guidance.
   State one invariant and the user harm if it fails.
2. Choose the tool from the property. Prefer explicit-state checking for small
   finite protocols, TLA+/TLC for distributed designs, or a code-level verifier
   when memory or arithmetic safety is the concern. Follow installation and
   dependency approval rules. Do not replace an unavailable proof tool with
   ordinary tests and call them a proof.
3. Specify bounds, initial states, atomic steps, input classes, and exclusions.
   Derive the expected contract independently of the implementation. Map every
   modeled action to its source boundary.
4. Add the executable check under `tools/formalverify/` and register its name in
   the command. Add a source map and a small real-source mutation. Preserve the
   single command and structured counterexample output.
5. Execute the scope, the broken variant, and the restored source. Update the
   inventory and fingerprints only after reviewing that evidence.

Keep this entry point generic as scopes grow. Keep domain details in the
inventory or a linked scope document, not in the skill name.

## Hand off

Give the command, explored cases, violations, first counterexample, mutation
results, assumptions, and exclusions. Distinguish a product violation from a
stale model or a tool failure. State whether product code changed and whether
the result is committed and pushed. Never claim unbounded correctness from a
bounded run.
