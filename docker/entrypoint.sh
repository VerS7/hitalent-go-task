#!/bin/sh
set -e

echo "[STAGE 1] waiting for PostgreSQL..."
until pg_isready -d "$DATABASE_URL" >/dev/null 2>&1; do
  sleep 1
done

echo "[STAGE 2] applying migrations..."
goose -dir /app/migrations postgres "$DATABASE_URL" up

echo "[STAGE 1] starting API server..."
exec /app/bin/server
