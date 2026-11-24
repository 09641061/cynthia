---
name: go-testing
description: Design, implement, and independently review behavior-focused Go tests and architectural guardrails in Akame.
---

# Go testing

Follow `go-gin-ddd`, repository instructions, and `permissions` when applicable.
Derive expected behavior from requirements and contracts, not solely implementation.

- Test real domain objects, invariants, invalid input, boundaries, and unchanged state on rejection.
- Use narrow, snapshot-safe fakes for application boundaries; assert outcomes and state, not only calls or nil errors.
- Use real PostgreSQL and production migrations for SQL, constraints, concurrency, and transactions; use real Redis for its scripts.
- Test HTTP binding, payloads, failures, authentication and authorization with `httptest`; reserve E2E for critical journeys.
- Keep tests independent, deterministic and readable. Never use `time.Sleep` in tests or parallelize shared mutable fixtures.
- External Redis expiration uses real server time: observe expiry with a bounded deadline; do not disguise fixed sleeps as synchronization.
- Do not weaken assertions, hide failures with skips, manufacture coverage, or change production behavior to fit incorrect tests.
- Never use real secrets in fixtures. Report infrastructure skips and unexecuted checks accurately.
- Prefer explicit synchronization, clocks, and `testing/synctest` when applicable; Go 1.22+ range variables need no defensive copies.
- A separate reviewer or tester must independently compare tests with domain requirements.
- All `.go` files, including tests and generated files, are limited to 150 physical lines without exceptions.
- Generate UUIDs in tests; never hardcode UUID literals. Keep procedural repository rules in Markdown and executable guardrails in native Go tests, not shell scripts.
- Avoid obvious narration and decorative comments. Preserve build directives, licenses, exported API contracts, and non-obvious invariants or constraints.
- Run Go tests and verification only via `nix-shell` as directed by `AGENTS.md`.
