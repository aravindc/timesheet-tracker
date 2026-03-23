#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# db.sh — backup or restore the timesheet-tracker database
#
# Only backs up the four application tables in the public schema:
#   users, projects, work_days, hourly_rates
#
# Usage:
#   ./db.sh backup                      # backup using SUPABASE_DB_URL from .env.app
#   ./db.sh backup  [DB_URL]            # backup from a specific DB URL
#   ./db.sh restore [DB_URL]            # restore latest backup to a specific DB URL
#   ./db.sh restore [DB_URL] [file]     # restore a specific SQL file to a DB URL
# ---------------------------------------------------------------------------

# Application tables to include in backup/restore
APP_TABLES=(users projects work_days hourly_rates)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$(dirname "$SCRIPT_DIR")/.env.app"
BACKUP_DIR="$SCRIPT_DIR/backup"
PG_IMAGE="postgres:17"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

usage() {
  echo "Usage:"
  echo "  $0 backup                       # backup current DB (reads URL from .env.app)"
  echo "  $0 backup  <DB_URL>             # backup from a specific DB URL"
  echo "  $0 restore <DB_URL>             # restore latest backup to DB URL"
  echo "  $0 restore <DB_URL> <file.sql>  # restore a specific SQL file to DB URL"
  exit 1
}

load_env_url() {
  if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: .env.app not found at $ENV_FILE"
    exit 1
  fi
  local url
  url=$(grep -E '^SUPABASE_DB_URL=' "$ENV_FILE" | head -1 | cut -d'=' -f2-)
  if [[ -z "$url" ]]; then
    echo "ERROR: SUPABASE_DB_URL not set in $ENV_FILE"
    exit 1
  fi
  echo "$url"
}

latest_backup() {
  local latest
  latest=$(ls -t "$BACKUP_DIR"/*.sql 2>/dev/null | head -1)
  if [[ -z "$latest" ]]; then
    echo "ERROR: No SQL backup files found in $BACKUP_DIR"
    exit 1
  fi
  echo "$latest"
}

check_docker() {
  if ! docker info &>/dev/null; then
    echo "ERROR: Docker is not running"
    exit 1
  fi
}

# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

do_backup() {
  local db_url="$1"
  local timestamp
  timestamp=$(date +"%Y%m%d_%H%M%S")
  local outfile="$BACKUP_DIR/backup_${timestamp}.sql"

  mkdir -p "$BACKUP_DIR"
  check_docker

  echo "==> Backing up database..."
  echo "    Target file: $outfile"

  # Build --table flags for each app table in the public schema
  local table_flags=()
  for t in "${APP_TABLES[@]}"; do
    table_flags+=(--table "public.${t}")
  done

  docker run --rm "$PG_IMAGE" \
    pg_dump "$db_url" \
      --no-owner --no-acl --format=plain \
      --schema=public \
      "${table_flags[@]}" \
    | sed "s/SELECT pg_catalog.set_config('search_path', '', false);/SELECT pg_catalog.set_config('search_path', 'public', false);/" \
    > "$outfile"

  local lines
  lines=$(wc -l < "$outfile")
  echo "    Done. ${lines} lines written."
  echo ""
  echo "    To restore this backup:"
  echo "    $0 restore <DB_URL> $outfile"
}

do_restore() {
  local db_url="$1"
  local infile="$2"

  if [[ ! -f "$infile" ]]; then
    echo "ERROR: File not found: $infile"
    exit 1
  fi

  check_docker

  echo "==> Restoring database..."
  echo "    Source file: $infile"
  echo ""
  echo "WARNING: This will apply all SQL in the backup file to the target database."
  read -r -p "    Continue? [y/N] " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 0
  fi

  docker run --rm -i "$PG_IMAGE" \
    psql "$db_url" \
      --single-transaction \
      -c "SET search_path TO public;" \
      -f /dev/stdin \
    < "$infile"

  echo ""
  echo "==> Restore complete."
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

if [[ $# -lt 1 ]]; then
  usage
fi

COMMAND="$1"

case "$COMMAND" in
  backup)
    if [[ $# -ge 2 ]]; then
      DB_URL="$2"
    else
      DB_URL=$(load_env_url)
    fi
    do_backup "$DB_URL"
    ;;

  restore)
    if [[ $# -lt 2 ]]; then
      echo "ERROR: restore requires a DB URL"
      usage
    fi
    DB_URL="$2"
    if [[ $# -ge 3 ]]; then
      SQL_FILE="$3"
    else
      SQL_FILE=$(latest_backup)
      echo "    No file specified, using latest: $SQL_FILE"
    fi
    do_restore "$DB_URL" "$SQL_FILE"
    ;;

  *)
    echo "ERROR: Unknown command '$COMMAND'"
    usage
    ;;
esac
