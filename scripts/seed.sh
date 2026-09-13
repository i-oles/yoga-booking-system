#!/usr/bin/env bash
# Dev-only helper: POSTs 4 sample classes to a running server via the admin
# API, so a fresh local checkout has something to look at right away.
# Usage: make run (in one terminal), then make seed (in another).
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

if [ -z "${AUTH_SECRET:-}" ] && [ -f .env ]; then
  # read just this one key, since other .env values (e.g. the Gmail app
  # password) may contain spaces and aren't safe to `source` as shell
  AUTH_SECRET="$(grep -m1 '^AUTH_SECRET=' .env | cut -d '=' -f2-)"
fi

if [ -z "${AUTH_SECRET:-}" ]; then
  echo "AUTH_SECRET is not set (check your .env)" >&2
  exit 1
fi

future_time() {
  # $1 = days ahead, $2 = "HH:MM:SS"
  if date -v+1d >/dev/null 2>&1; then
    date -u -v+"$1"d "+%Y-%m-%dT$2Z"
  else
    date -u -d "+$1 days" "+%Y-%m-%dT$2Z"
  fi
}

payload=$(cat <<JSON
[
  {"start_time": "$(future_time 3 18:00:00)", "class_level": "Beginner", "class_name": "Hatha Yoga", "max_capacity": 10, "location": "Ożarowska 75/36"},
  {"start_time": "$(future_time 4 19:00:00)", "class_level": "Intermediate", "class_name": "Vinyasa Flow", "max_capacity": 8, "location": "Ogród Saski"},
  {"start_time": "$(future_time 5 07:30:00)", "class_level": "All levels", "class_name": "Morning Flow", "max_capacity": 12, "location": "Ogród Krasińskich"},
  {"start_time": "$(future_time 6 10:00:00)", "class_level": "Advanced", "class_name": "Ashtanga", "max_capacity": 6, "location": "Park Moczydło"}
]
JSON
)

response=$(curl -sS -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/classes" \
  -H "Authorization: Bearer $AUTH_SECRET" \
  -H "Content-Type: application/json" \
  -d "$payload")

status="${response##*$'\n'}"
body="${response%$'\n'*}"

if [ "$status" != "201" ]; then
  echo "seed failed: $status $body" >&2
  exit 1
fi

echo "seeded 4 classes:"
echo "$body"
