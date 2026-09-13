# ABOUT

A web app for booking yoga classes, built for a real yoga instructor. Anyone
can book a spot without creating an account — that's intentional, not a
missing feature. The whole flow runs over email instead of a login system:

1. A visitor picks a class on the home page and fills in the sign-up form
   (first name, last name, email). That doesn't book the spot yet — it
   creates a `pending_booking` and emails a confirmation link.
2. Clicking that link is what actually books the spot. It turns the
   `pending_booking` into a real `booking`. If nobody clicks it within an
   hour, the pending booking is dropped automatically.
3. The confirmation email that follows also carries a cancellation link —
   no login, no password, just a tokenized URL tied to that one booking.
   Cancelling is a single click on that link.
4. Everything else is communicated by email too: a reminder ~24h before the
   class, and a notice if the instructor edits or cancels a class the
   person is booked into.
5. Regulars don't need an account either: the instructor (via the admin API)
   can activate a "pass" (`karnet`) for a person's email — a pool of
   prepaid slots that can also be attached retroactively to that person's
   existing bookings.

An admin API (`/api/v1/*`, Bearer-token protected) sits alongside the public
site for managing classes, bookings, passes, and contacts.

Tech stack:

- Go 1.24 + [Gin](https://github.com/gin-gonic/gin) for HTTP routing.
- Server-side rendered HTML (`html/template`) for the public site.
- GORM + SQLite, with the schema auto-migrated on startup — no manual
  migrations.
- Email via Gmail SMTP, swappable for an in-memory mock sender in dev (see
  `mockEmailSender` below).

# DEVELOPMENT

Requirements: Go 1.24+, gcc (for `mattn/go-sqlite3`, needs CGO — usually
already available on macOS/Linux).

clone the repository:

```
git clone <repo-url>
cd yoga-website
```

set up local config:

```
cp .env.example .env
# defaults to CONFIG=dev and mockEmailSender=true — no real Gmail account needed
```

run the server:

```
go run cmd/yoga/main.go
# or: make run
```

On startup the SQLite database (`DATABASE_PATH`, default `yoga.db`) is
created and migrated automatically. Server listens on `:8080` —
http://localhost:8080/

the database starts empty, so the home page has nothing to show yet. In a
second terminal, seed 4 sample classes via the admin API:

```
make seed
```

reload http://localhost:8080/ and the 4 classes should be there. See
[EXAMPLES](#examples) below to also walk through the booking flow by hand.

# FEATURES

- public site — list of upcoming classes, sign-up via a form (first name,
  last name, email), booking confirmed via an emailed link, cancellation
  via a tokenized link.
- pending bookings — submitting the form creates a `pending_booking` with a
  confirmation token emailed to the user; the actual `booking` is only
  created once that link is clicked. Unconfirmed pending bookings older
  than one hour are cleaned up automatically in the background on startup.
- reminders — on process startup, a one-off background round of reminder
  emails is sent to everyone booked into a class starting within 24h.
- passes (karnety) — an admin can activate a pass (a pool of prepaid slots)
  for an email, optionally attaching it to that person's existing
  pass-less bookings.
- contacts — a simple mailing list, managed via the API.
- admin API (`/api/v1/*`) — CRUD for classes, viewing/deleting
  bookings/pending bookings, activating passes, contacts. Protected by a
  static Bearer token.
- mocked email (dev mode) — instead of sending mail via Gmail SMTP,
  messages are kept in memory and viewable at `/emails`, so the whole
  booking flow (including the confirmation link) works locally without a
  real mailbox.

# API

Public site (HTML):

```
GET    /                                        --> home page, list of upcoming classes
GET    /error                                   --> generic error page
GET    /classes/:class_id/pending_bookings/form --> sign-up form for a class
POST   /pending_bookings                        --> submit sign-up form, emails a confirmation link (rate-limited: burst 2, ~1 req/s)
GET    /bookings?token=                         --> confirm a pending booking, turns it into a booking
GET    /bookings/:id/cancel_form?token=         --> cancellation form
DELETE /bookings/:id?token=                     --> cancel a booking
GET    /emails                                  --> only when mockEmailSender: true — preview of every "sent" email
```

Admin API (`/api/v1/*`) — every endpoint requires header
`Authorization: Bearer <AUTH_SECRET>`. Domain errors come back as
`{"error": "..."}` with status `400` (validation), `404` (not found), `409`
(conflict), or `500`.

```
POST   /api/v1/classes                    --> create one or more classes
GET    /api/v1/classes                    --> list classes with current/max capacity
PATCH  /api/v1/classes/:class_id          --> partially update a class, notifies existing bookings
DELETE /api/v1/classes/:class_id          --> delete a class, notifies existing bookings
GET    /api/v1/classes/:class_id/bookings --> list bookings for a class
GET    /api/v1/bookings                   --> list all bookings
DELETE /api/v1/bookings/:booking_id       --> admin-deletes a booking (no token needed)
GET    /api/v1/pending_bookings           --> list unconfirmed sign-ups
PUT    /api/v1/passes                     --> activate a pass for an email
GET    /api/v1/contacts                   --> list contacts
POST   /api/v1/contacts                   --> add contacts (duplicate emails silently skipped)
```

Note on locations: `location` must match (case-insensitively) one of the
locations hardcoded in `internal/application/location/link_provider.go`,
e.g. `"Ożarowska 75/36"`, `"Ogród Krasińskich"`, `"Ogród Saski"`,
`"Park Moczydło"` — that's a name → Google Maps link mapping specific to
this instance. `POST /api/v1/classes` enforces this and rejects an unknown
`location` with `400`. **`PATCH` does not re-check it** — setting an
unrecognized `location` via update is a known gap: the change still gets
persisted, but it then breaks the home page and `GET /api/v1/classes` with
a `500` for everyone, until it's corrected. Stick to the known locations
above for both create and update.

# CONFIGURATION

A JSON file selected by `CONFIG` (`dev` --> `config/dev.json`, `prod` -->
`config/prod.json`) holds timeouts, listen address, SMTP settings,
`domainAddr` (base URL used in email links), `mockEmailSender`,
`isVacation` (a flag shown on the home page).

A `.env` file (or plain env vars) overrides secrets:

```
CONFIG               --> dev or prod, which file under config/ to load
DATABASE_PATH        --> path to the SQLite file, e.g. yoga.db
AUTH_SECRET          --> secret for the admin API (Authorization: Bearer <secret>)
NOTIFIER_LOGIN       --> Gmail SMTP login (required even when mockEmailSender: true, any dummy value works in dev)
NOTIFIER_PASSWORD    --> Gmail SMTP password (same as above)
```

`.env.example` has safe local-dev values — copy it to `.env` and swap in
real secrets if/when needed.

# EXAMPLES

`make seed` (see [DEVELOPMENT](#development)) already creates 4 classes for
you. The steps below walk through the same "create a class" call by hand,
then the rest of the booking flow end to end. Assuming `.env.example` is
unchanged (`AUTH_SECRET=local-dev-secret`):

---

- create a class (use a future date so it shows up on the home page):
```
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
the response includes the `id` of the created class — open
http://localhost:8080/ and it should be visible there.

---

- sign up for the class (or use the form in the browser at
  `/classes/<class_id>/pending_bookings/form`):
```
curl -X POST http://localhost:8080/pending_bookings \
  --data-urlencode "email=jan@example.com" \
  --data-urlencode "first_name=Jan" \
  --data-urlencode "last_name=Kowalski" \
  --data-urlencode "class_id=<class_id>"
```

---

- grab the confirmation link from the "sent" email (mocked in dev, no real
  email goes out):

  http://localhost:8080/emails

  find the message with a link like
  `http://localhost:8080/bookings?token=...` and open it — this confirms
  the booking and creates a `booking`. The follow-up confirmation email
  (with the cancellation link) shows up on that same `/emails` page.

---

- inspect the data via the admin API:
```
curl http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer local-dev-secret"

curl http://localhost:8080/api/v1/pending_bookings \
  -H "Authorization: Bearer local-dev-secret"
```

---

- (optional) activate a pass and attach it to the booking above:
```
curl -X PUT http://localhost:8080/api/v1/passes \
  -H "Authorization: Bearer local-dev-secret" \
  -H "Content-Type: application/json" \
  -d '{"email": "jan@example.com", "initial_assigned_slots": 1, "total_slots": 10}'
```

# TESTING

```
make test    # go test -v ./...
make lint    # golangci-lint run ./...
make build   # go build -o bin/yoga cmd/yoga/main.go
```

Mocks (`mock/`, generated with `mockgen` from interfaces in
`internal/domain` and `internal/application/*`) are refreshed via
`make mocks`.

# DOCKER

```
docker compose up --build
```

`docker-compose.yml` reads `NOTIFIER_LOGIN`, `NOTIFIER_PASSWORD`,
`AUTH_SECRET`, `CONFIG` from environment variables/`.env` in the directory
you run it from, and keeps the SQLite database in the `sqlite_data`
volume. Real deployment (Fly.io) uses `fly.toml`
(`config/prod.json`, `mockEmailSender: false` — real Gmail SMTP credentials
required there).
