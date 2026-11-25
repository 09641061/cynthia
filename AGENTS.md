# Cynthia (Takodu)

Cynthia is the implementation of **Takodu** (Takodu Industries), an appointment scheduling and customer management platform for small service businesses (salons, barbershops, spas, clinics, advisory).

## Target Segments & Access Model

- **End Customers:** external users (not IAM members) who check services/availability and book, reschedule, or cancel via WhatsApp without installing apps.
- **Owners & Administrators / Collaborators (Staff):** internal users modeled in `permissions` with **one `Owner`** and all other business users as **`Member`s** with customizable group permissions (e.g., admin members managing services, schedules, and bookings; collaborators viewing their daily schedule, updating appointment status, and appearing in `scheduling`).

## Skills

Always read and follow the relevant skill in `.agents/skills/` before designing, implementing, reviewing, or testing:

- `scheduling` — calendars, availability, conflict prevention, appointments, WhatsApp bookings.
- `crm` — customers, phone matching, conversations, WhatsApp integration.
- `catalog` — categories, services, ordering.
- `organization` — business settings, integrations, AI agent.
- `permissions` — `Owner`, `Member`, customizable groups/permissions, and scheduling authorization.
- `go-testing` — Go tests and repository verification guardrails.

Keep domain boundaries separate, verify against actual code/tests, and keep solutions simple for the MVP.

## Permanent rules

- **Do not run the application** or leave it running. Only execute tests and verification; never use `go run` or `make run`.
- **Migrations:** use versioned SQL migration files with format `<number>_<timestamp>_<verb>_<entity>.sql` (e.g. `0001_20261105204851_create_schema.sql`). Create a new migration file for each schema change; do not edit migrations that have already been applied.
- **Seeds:** keep one SQL seed file per feature. Check for an existing migration or seed before creating any SQL file.
- **Git:** Work only on task branches; never edit, commit, push, merge, rebase, reset, or otherwise modify `main` or `develop`. The agent may create commits and push task branches when the user explicitly requests it. Never merge branches or create/manage pull requests; the user handles PRs and merges. Never change the Git username or email configuration.
- **Existing work:** preserve unrelated working-tree changes.
- **Secrets:** never expose `.env`, credentials or tokens.
- **Documentation:** the user maintains it. Do not create or update documentation unless explicitly requested.
- **Go files:** every `.go` file must have at most 150 physical lines, including tests and generated files, without exceptions.
- **Comments:** avoid redundant narration and decorative separators; preserve directives, licenses, exported API contracts, and non-obvious business or technical decisions.
- **Language:** write code, comments and repository instructions in English.

## Tests and verification

Run Go tests and verification inside `nix-shell`, for example `nix-shell --run 'make verify'`. Run `make test`, `make test-race`, and `make test-integration` as separate suites where relevant. `make verify` runs repository guardrails, formatting checks, and `go vet`; none of these commands may start the application. Integration tests use Testcontainers to start and remove their own disposable PostgreSQL container, so a separate database service or `DATABASE_URL` is not required; a working container runtime is required. CI runs the test suites separately and does not define a database service. Add regression tests for behavior changes and keep tests deterministic and independent.
