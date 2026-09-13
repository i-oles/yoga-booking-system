# Otojoga — yoga-website

A web app for booking yoga classes. Go backend (Gin + GORM/SQLite), a simple
public site (booking via a form + email confirmation link), and a protected
admin API for managing classes, bookings, passes, and contacts.

## Table of contents

- [Features](#features)
- [Architecture & tech stack](#architecture--tech-stack)
- [Data model](#data-model)
- [Endpoints](#endpoints)
  - [Public site (HTML)](#public-site-html)
  - [Admin API](#admin-api-apiv1)
- [Configuration](#configuration)
- [Running locally](#running-locally)
- [End-to-end smoke test (test data)](#end-to-end-smoke-test-test-data)
- [Tests, lint, build](#tests-lint-build)
- [Docker](#docker)

## Features

- **Public site** — list of upcoming classes, sign-up via a form (first name,
  last name, email), booking confirmed via an emailed link, cancellation via
  a tokenized link.
- **Pending bookings** — submitting the form doesn't create a booking right
  away; it creates a `pending_booking` with a confirmation token emailed to
  the user. The actual `booking` is only created once that link is clicked.
  Unconfirmed pending bookings older than one hour are cleaned up
  automatically in the background on startup.
- **Reminders** — on process startup, a one-off background round of reminder
  emails is sent to everyone booked into a class starting within the next 24h.
- **Passes (karnety)** — an admin can activate a pass (a pool of prepaid
  slots) for a given email, optionally attaching it immediately to that
  person's existing bookings that aren't tied to any pass yet.
- **Contacts** — a simple mailing list (e.g. for a newsletter), managed via
  the API.
- **Admin API** (`/api/v1/*`) — CRUD for classes, viewing/deleting
  bookings/pending bookings, activating passes, contacts. Protected by a
  static Bearer token.
- **Mocked email (dev mode)** — instead of actually sending mail via Gmail
  SMTP, messages are kept in memory and can be viewed at `/emails`. This lets
  you exercise the whole booking flow (including the confirmation link)
  locally without configuring a real mailbox.

## Architecture & tech stack

- Go 1.24, [Gin](https://github.com/gin-gonic/gin) as the router/HTTP layer,
  server-side rendering (`html/template` via Gin) for the public site.
- [GORM](https://gorm.io/) + SQLite (`mattn/go-sqlite3`) as the database;
  schema is created automatically (`AutoMigrate`) on startup — no manual
  migrations needed, and `assets/sqlite/*.sql` is just a historical/reference
  copy of the schema, unused by the code.
- Email sent via Gmail SMTP (`gopkg.in/gomail.v2`), swappable for an in-memory
  sender (`mockEmailSender`).
- Configuration: a JSON file (`config/dev.json` / `config/prod.json`) plus
  environment variables from `.env` (`godotenv`) overriding select secrets.
- Layers: `internal/interfaces/http` (HTTP/HTML/API) →
  `internal/application/*` (services/use cases) →
  `internal/domain` (models, repository contracts) →
  `internal/infrastructure` (SQLite, notifier, sender).

## Data model

| Entity | Key fields | Meaning |
|---|---|---|
| **Class** | `id`, `start_time`, `class_level`, `class_name`, `max_capacity`, `location` | A single class occurrence (time, level, name, capacity, location). `location` is free text; it doesn't need to match `location.LinkProvider` (see below) — if it doesn't, the class still displays everywhere, it just has no clickable Google Maps link. |
| **PendingBooking** | `id`, `class_id`, `email`, `first_name`, `last_name`, `confirmation_token`, `created_at` | A sign-up waiting for email confirmation (TTL: 1h). |
| **Booking** | `id`, `class_id`, `pass_id?`, `first_name`, `last_name`, `email`, `confirmation_token`, `reminded_at?`, `created_at` | A confirmed booking; optionally linked to a pass. |
| **Pass** | `id`, `email`, `total_slots`, `created_at`, `updated_at` | A pass — a pool of usable slots tied to an email. |
| **Contact** | `id`, `email`, `first_name`, `last_name` | A mailing-list contact (separate from bookings). |

## Endpoints

### Public site (HTML)

| Method | Path | Description |
|---|---|---|
| GET | `/` | Home page — list of upcoming classes (`index.html`). |
| GET | `/error` | Generic error page. |
| GET | `/classes/:class_id/pending_bookings/form` | Sign-up form for a given class. |
| POST | `/pending_bookings` | Accepts the sign-up form (`email`, `class_id`, `first_name`, `last_name`), creates a `pending_booking`, and emails a confirmation link. Rate-limited (burst 2, ~1 req/s). |
| GET | `/bookings?token=` | The link from the confirmation email — turns a `pending_booking` into a `booking` and emails a confirmation. |
| GET | `/bookings/:id/cancel_form?token=` | Cancellation form/page (requires the token from the email). |
| DELETE | `/bookings/:id?token=` | Cancels a booking given its token. |
| GET | `/emails` | **Only when `mockEmailSender: true`.** Preview of every "sent" email (HTML body) — useful for grabbing the confirmation link without a real mailbox. |

### Admin API (`/api/v1/*`)

Every endpoint requires the header:

```
Authorization: Bearer <AUTH_SECRET>
```

where `<AUTH_SECRET>` is the `AUTH_SECRET` environment variable. Domain
errors are returned as `{"error": "..."}` with status `400` (validation),
`404` (not found), `409` (conflict, e.g. deleting a class that still has
bookings), or `500`.

**Classes**

| Method | Path | Body | Description |
|---|---|---|---|
| POST | `/api/v1/classes` | array of `CreateClassRequest` | Creates one or more classes. |
| GET | `/api/v1/classes` | `{"only_upcoming_classes": bool, "classes_limit": int\|null}` (JSON body despite being a GET) | Lists classes with current/max capacity. |
| PATCH | `/api/v1/classes/:class_id` | `UpdateClassRequest` (all fields optional, plus `message`) | Partially updates a class; notifies existing bookings by email about the change. |
| DELETE | `/api/v1/classes/:class_id` | `{"message": "optional message"}` | Deletes a class; notifies existing bookings about the cancellation. |
| GET | `/api/v1/classes/:class_id/bookings` | — | Lists bookings for a given class. |

> **Note:** for a class to get a clickable Google Maps link, `location` must
> match (case-insensitively) one of the locations hardcoded in
> `internal/application/location/link_provider.go`, e.g.
> `"Ożarowska 75/36"`, `"Ogród Krasińskich"`, `"Ogród Saski"`,
> `"Park Moczydło"`. That's a name → Google Maps link mapping specific to
> this instance. Any other value is accepted and displayed just fine — the
> class simply won't have a map link.

Example — creating a class:

```json
POST /api/v1/classes
[
  {
    "start_time": "2026-09-20T18:00:00+02:00",
    "class_level": "Beginner",
    "class_name": "Hatha Yoga",
    "max_capacity": 12,
    "location": "Ożarowska 75/36"
  }
]
```

**Bookings**

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/bookings` | Lists all bookings (with class and pass data). |
| DELETE | `/api/v1/bookings/:booking_id` | Admin deletes a booking (no token required). |

**Pending bookings**

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/pending_bookings` | Lists unconfirmed sign-ups. |

**Passes**

| Method | Path | Body | Description |
|---|---|---|---|
| PUT | `/api/v1/passes` | `{"email": "...", "initial_assigned_slots": 0, "total_slots": 10}` | Activates a pass for an email; if `initial_assigned_slots > 0`, attaches the pass to that many of that person's existing pass-less bookings — there must be exactly that many, otherwise `409`. |

Example:

```json
PUT /api/v1/passes
{ "email": "jan@example.com", "initial_assigned_slots": 2, "total_slots": 10 }
```

**Contacts**

| Method | Path | Body | Description |
|---|---|---|---|
| GET | `/api/v1/contacts` | — | Lists contacts. |
| POST | `/api/v1/contacts` | array of `{"email", "first_name", "last_name"}` | Adds contacts (duplicate emails are silently skipped). |

## Configuration

The app combines two configuration sources:

1. **A JSON file** selected by the `CONFIG` variable (`dev` →
   `config/dev.json`, `prod` → `config/prod.json`) — timeouts, listen
   address, SMTP settings, `domainAddr` (base URL used in email links),
   `mockEmailSender`, `isVacation` (a flag shown on the home page).
2. **A `.env` file** (loaded via `godotenv`, optional — plain environment
   variables work too) — overrides secrets that aren't kept in the JSON:

   | Variable | Description |
   |---|---|
   | `CONFIG` | `dev` or `prod` — which file under `config/` to load. |
   | `DATABASE_PATH` | Path to the SQLite file (e.g. `yoga.db`). |
   | `AUTH_SECRET` | Secret for the admin API (`Authorization: Bearer <secret>`). |
   | `NOTIFIER_LOGIN` / `NOTIFIER_PASSWORD` | Gmail SMTP login credentials. **Required even when `mockEmailSender: true`** (config validation requires non-empty values) — in dev mode these can be any dummy values, since no real email is sent. |

The repo includes `.env.example` with safe local-dev values — copy it to
`.env` and swap in real secrets if/when needed.

## Running locally

Requirements: Go 1.24+, gcc (for `mattn/go-sqlite3`, which needs CGO — usually
already available on macOS/Linux).

```bash
git clone <repo-url>
cd yoga-website

cp .env.example .env
# the default .env.example has CONFIG=dev and mockEmailSender=true (see config/dev.json)
# — no real Gmail account needed to see the app working

go run cmd/yoga/main.go
# or: make run
```

On startup:

- The SQLite database (`DATABASE_PATH`, default `yoga.db`) is created and
  migrated automatically — nothing to initialize by hand.
- The server listens on `:8080` (`config/dev.json` → `listenAddress`).
- Home page: http://localhost:8080/

## End-to-end smoke test (test data)

The database starts empty, so to see the site actually working you first
need to add some classes via the admin API. Assuming `.env.example` is
unchanged (`AUTH_SECRET=local-dev-secret`):

1. **Add a class** (use a future date so it shows up on the home page):

   ```bash
   curl -X POST http://localhost:8080/api/v1/classes \
     -H "Authorization: Bearer local-dev-secret" \
     -H "Content-Type: application/json" \
     -d '[{
           "start_time": "2026-12-01T18:00:00+01:00",
           "class_level": "Beginner",
           "class_name": "Hatha Yoga",
           "max_capacity": 10,
           "location": "Ożarowska 75/36"
         }]'
   ```

   The response includes the `id` of the created class. Use one of the
   locations described above in [Admin API](#admin-api-apiv1) if you want a
   clickable map link on the home page; any other value works too, just
   without the link.

2. **Open the home page** — http://localhost:8080/ — the class you added
   should be visible. Copy its `id` from the response above (or from the
   "book" link on the page).

3. **Sign up** via the form in the browser
   (`http://localhost:8080/classes/<class_id>/pending_bookings/form`), or
   directly:

   ```bash
   curl -X POST http://localhost:8080/pending_bookings \
     --data-urlencode "email=jan@example.com" \
     --data-urlencode "first_name=Jan" \
     --data-urlencode "last_name=Kowalski" \
     --data-urlencode "class_id=<class_id>"
   ```

4. **Grab the confirmation link from the "sent" email** (since
   `mockEmailSender: true` in dev, no real email goes out):

   http://localhost:8080/emails

   Find the message with a link like
   `http://localhost:8080/bookings?token=...` and open it in a browser — this
   confirms the booking and creates a `booking`. The follow-up confirmation
   email (with the cancellation link) will show up on that same `/emails`
   page.

5. **Inspect the data via the admin API**:

   ```bash
   curl http://localhost:8080/api/v1/bookings \
     -H "Authorization: Bearer local-dev-secret"

   curl http://localhost:8080/api/v1/pending_bookings \
     -H "Authorization: Bearer local-dev-secret"
   ```

6. (Optional) **Activate a pass** and attach it to the booking above:

   ```bash
   curl -X PUT http://localhost:8080/api/v1/passes \
     -H "Authorization: Bearer local-dev-secret" \
     -H "Content-Type: application/json" \
     -d '{"email": "jan@example.com", "initial_assigned_slots": 1, "total_slots": 10}'
   ```

## Tests, lint, build

```bash
make test    # go test -v ./...
make lint    # golangci-lint run ./...
make build   # go build -o bin/yoga cmd/yoga/main.go
```

Mocks (`mock/`, generated with `mockgen` from interfaces in `internal/domain`
and `internal/application/*`) are refreshed via `make mocks`.

## Docker

```bash
docker compose up --build
```

`docker-compose.yml` reads `NOTIFIER_LOGIN`, `NOTIFIER_PASSWORD`,
`AUTH_SECRET`, `CONFIG` from environment variables/`.env` in the directory
you run Docker Compose from, and keeps the SQLite database in the
`sqlite_data` volume. Real deployment (Fly.io) uses `fly.toml`
(`config/prod.json`, `mockEmailSender: false` — real Gmail SMTP credentials
are required there).
