# gotaro

GoTaro is a simple web-based task tracker written in Go (server-rendered HTML, no SPA).

## Run locally

1. Start PostgreSQL and create a database.
2. Set `DATABASE_URL` (e.g. `postgres://user:pass@localhost:5432/gotaro?sslmode=disable`).
3. From the repo root: `go run ./cmd/app`
4. Open [http://localhost:8080](http://localhost:8080) (override with `LISTEN_ADDR`).

Migrations run on startup. Optional: `GOTARO_COOKIE_SECURE=true` when serving over HTTPS.

## Seed test data

Create a test user and populate tasks (with projects/tags) using:

`go run ./cmd/seed`

Defaults:
- Email: `test@example.com`
- Password: `test12345`
- Tasks: `80`
- Projects: `6`
- Tags: `10`
- Existing tasks/projects/tags for that user are wiped before seeding.

Examples:
- `go run ./cmd/seed -email qa@example.com -password supersecret -tasks 150`
- `go run ./cmd/seed -wipe=false -tasks 40` (append more tasks instead of wiping)

## Features (summary)

- Tasks: title, optional description, status, priority, due date, tags, optional **project** (category).
- Filter/search/sort lists; open, completed, and archived views; **CSV export** of the current filters.
- **Stats** summary on list pages; **one-click Done** on open tasks; flash toasts with auto-dismiss.
- Register/login; data stored in PostgreSQL.

## Original core requirements

- Users can create, edit, delete, and mark tasks as done.
- Each task has at least: title, optional description, status, priority, due date, and tags.
- Users can filter tasks by status, priority, tag, and due date.
- Users can search for tasks by title and description.
- Users can sort tasks by created date, due date, priority, and status.
- Users can see all open tasks on the main page by default.
- Users can view completed tasks separately.
