# brewupdater

## Session startup

At the start of every session, read the following files before doing any other work:

1. `CONTEXT.md` — domain model and authoritative terminology
2. `CONTEXT.local.md` (if present) — local working context; gitignored
3. `doc/plan/README.md` — implementation plan, session workflow and conventions

When working on a plan session, also read its `doc/plan/S<n>-*.md` file and stay within its scope.

## Conventions

Code, test, lint, comment and commit conventions are in the *Conventions* section of `doc/plan/README.md`.
They follow the OD team guidelines (`go.md`, `code-comments.md`, `commit-and-mr.md`), adapted for a personal GitHub project.
