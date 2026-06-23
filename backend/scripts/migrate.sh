#!/bin/sh

# Database Migration Script

# Runs all *.up.sql files in migrations/ directory in order.
# Uses psql directly — no external migration tool dependency.
#
# Usage:
#   ./migrate.sh           # Run all pending migrations
#   ./migrate.sh down      # Rollback last migration set

set -e

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/app/migrations}"
DATABASE_URL="${DATABASE_URL:-postgres://myjob:myjob_dev@localhost:5432/myjob?sslmode=disable}"

# Extract connection details from DATABASE_URL
DB_HOST=$(echo "$DATABASE_URL" | sed -n 's|.*@\([^:]*\):\([0-9]*\)/.*|\1|p')
DB_PORT=$(echo "$DATABASE_URL" | sed -n 's|.*@\([^:]*\):\([0-9]*\)/.*|\2|p')
DB_NAME=$(echo "$DATABASE_URL" | sed -n 's|.*/\([^?]*\).*|\1|p')
DB_USER=$(echo "$DATABASE_URL" | sed -n 's|://\([^:]*\):.*|\1|p')
DB_PASS=$(echo "$DATABASE_URL" | sed -n 's|://[^:]*:\([^@]*\)@.*|\1|p')

export PGPASSWORD="$DB_PASS"

echo "Running migrations against $DB_NAME on $DB_HOST:$DB_PORT..."

if [ "$1" = "down" ]; then
    echo "Rolling back migrations..."
    for f in $(ls -r "$MIGRATIONS_DIR"/*.down.sql 2>/dev/null); do
        echo "  Applying: $(basename "$f")"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$f" -v ON_ERROR_STOP=1
    done
    echo "Rollback complete."
else
    echo "Applying migrations..."
    for f in $(ls "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sort); do
        echo "  Applying: $(basename "$f")"
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$f" -v ON_ERROR_STOP=1
    done
    echo "Migrations complete."
fi
