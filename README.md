# forum

A web forum built in Go with SQLite, supporting registration/login, posts with
categories, comments, likes/dislikes, and post filtering.

## Features

- **Auth** — registration (email/username/password), login with cookie-based
  sessions (24h expiry, UUID session IDs), bcrypt-hashed passwords
- **Posts** — registered users can create posts tagged with one or more
  categories; visible to everyone, registered or not
- **Comments** — registered users can comment on any post; visible to everyone
- **Likes/dislikes** — registered users can react to posts and comments;
  clicking the same reaction again removes it, the opposite reaction switches
  it; counts are visible to everyone
- **Filtering** — by category, by the logged-in user's own posts ("My
  Posts"), and by posts the logged-in user has liked ("Liked Posts")

## Tech stack

- Go 1.22, standard library `net/http` (no router/framework)
- SQLite via [`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) (CGO)
- [`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) for password hashing
- [`gofrs/uuid`](https://github.com/gofrs/uuid) for session IDs
- `html/template` for server-rendered pages, no frontend framework

## Project structure

```
cmd/server/          entry point
internal/
  auth/               password hashing, session ID generation
  database/           SQLite connection + embedded migrations
  handlers/           HTTP handlers (auth, posts, comments, reactions) + middleware
  models/             plain structs (User, Post, Comment, Category, Session)
  repository/         all SQL, one file per table/entity
  testutil/           shared test helpers (in-memory test DB)
  utils/              input validation
web/
  templates/          HTML templates
  static/css/         stylesheet
data/                 SQLite database file (created at runtime, gitignored)
```

## Running locally

Requires Go 1.22+ and a C toolchain (gcc) — `go-sqlite3` uses CGO.

```bash
go mod download
go build -o forum-server ./cmd/server
./forum-server
```

The server listens on `:8080` and creates `data/forum.db` (with all tables)
on first run.

## Running with Docker

```bash
docker compose up --build
```

This builds the app in a multi-stage Docker image and runs it with a named
volume (`forum-data`) mounted at `/app/data`, so the database persists across
container restarts and rebuilds.

## Running tests

```bash
go test ./...
```

Repository-layer tests run against a real SQLite database in memory (see
`internal/testutil`), covering:
- user creation, lookup, and duplicate-email/username handling (including
  at the database constraint level, not just application-level checks)
- session creation, expiry, and the "one active session per user" rule
- reaction toggling (like → like again → removed; like → dislike → switched,
  never stacked) and per-user isolation
- post filtering by category, by author, and by liked status
- input validation boundary cases (username/password length limits, email
  format)

## Default categories

Four categories are seeded automatically on first run: General, Technology,
Gaming, Random. There's no UI for creating additional categories — this is a
deliberate scope decision, matching the subject's "implementation and choice
of categories is up to you."

## Notes on design decisions

- **One session per user.** Logging in from a new place invalidates any
  previous session for that user, rather than allowing multiple concurrent
  sessions.
- **Sessions and users use bcrypt + UUIDs** (the bonus tasks from the
  subject) rather than being optional add-ons bolted on separately.
- **At least one category is required per post**, matching the subject's
  "associate one or more categories to it" wording.
- **No CSRF protection.** Not required by the subject; worth adding if this
  were exposed beyond local/trusted use.
