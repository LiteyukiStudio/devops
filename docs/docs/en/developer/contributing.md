# Development Rules

## Basics

- Read existing code and docs before editing.
- Do not commit, push, or switch branches unless explicitly requested.
- Move one small goal through one traceable cycle.
- Update the `docs/` documentation site when adding features or changing flows.
- Update `TODO.md` when the plan, acceptance criteria, or status changes.

## Frontend

- Use `pnpm`.
- Prefer shadcn/ui for primitives.
- Use React Hook Form + Zod for forms.
- User-visible text must go through i18n.
- Lists should reuse the shared list component.
- Status must use semantic badges.

## Backend

- Use PostgreSQL, not SQLite.
- Do not store secrets or tokens as plaintext in business tables.
- External platform capabilities are adapted through backend providers, services, and APIs. The frontend must not orchestrate third-party APIs.
- Long-running work goes to workers, not synchronous HTTP requests.

## Verification

Use targeted checks for small changes. Run full verification, preferably with browser acceptance, when a change spans multiple domains, authentication, permissions, secrets, database migrations, or deployment runtime behavior.

## Documentation experience

Docs should reduce user effort. When writing docs, answer:

- What is the user trying to finish now?
- What is the shortest path?
- What should success look like?
- Where should they look first when it fails?

Architecture and internal boundaries belong in developer docs. They should not block a user from getting started.
