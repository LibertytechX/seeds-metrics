#!/bin/sh
set -e

echo "⏳ Waiting for PostgreSQL to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" 2>/dev/null; do
  sleep 1
done
echo "✅ PostgreSQL is ready"

echo "🗄️  Running migrations..."
for migration in /migrations/*.sql; do
  echo "  → Applying $migration"
  PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -q \
    -v ON_ERROR_STOP=1 \
    -f "$migration" 2>&1 || true
done
echo "✅ Migrations complete"

echo "🚀 Starting API server..."
exec ./main
