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

Keep domain boundaries separate, verify against actual code/tests, and keep solutions simple for the MVP.
