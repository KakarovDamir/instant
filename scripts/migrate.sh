#!/bin/bash

# Migration script for PostgreSQL database
# Usage: ./scripts/migrate.sh [up|down|status]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Load environment variables from .env file
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
else
    echo -e "${RED}Error: .env file not found${NC}"
    exit 1
fi

# Database connection parameters
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_DATABASE=${DB_DATABASE:-highload}
DB_USERNAME=${DB_USERNAME:-postgres}
DB_PASSWORD=${DB_PASSWORD:-damir}

# Migrations directory
MIGRATIONS_DIR="migrations"

# Function to check database connection
check_db_connection() {
    echo -e "${YELLOW}Checking database connection...${NC}"
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -c '\q' 2>/dev/null; then
        echo -e "${GREEN}✓ Database connection successful${NC}"
        return 0
    else
        echo -e "${RED}✗ Cannot connect to database${NC}"
        echo -e "${YELLOW}Waiting for database to be ready...${NC}"
        for i in {1..30}; do
            if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -c '\q' 2>/dev/null; then
                echo -e "${GREEN}✓ Database is ready${NC}"
                return 0
            fi
            sleep 1
        done
        echo -e "${RED}✗ Database is not ready after 30 seconds${NC}"
        return 1
    fi
}

# Function to create migrations table if not exists
create_migrations_table() {
    echo -e "${YELLOW}Creating migrations table if not exists...${NC}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" << EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    version VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);
EOF
    echo -e "${GREEN}✓ Migrations table ready${NC}"
}

# Function to check if migration was applied
is_migration_applied() {
    local version=$1
    local count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -t -c "SELECT COUNT(*) FROM schema_migrations WHERE version = '$version';" 2>/dev/null | tr -d ' ')
    [ "$count" -gt 0 ]
}

# Function to mark migration as applied
mark_migration_applied() {
    local version=$1
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -c "INSERT INTO schema_migrations (version) VALUES ('$version');" > /dev/null
}

# Function to unmark migration
unmark_migration() {
    local version=$1
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -c "DELETE FROM schema_migrations WHERE version = '$version';" > /dev/null
}

# Function to run migrations up
migrate_up() {
    echo -e "${YELLOW}Running migrations UP...${NC}"
    echo ""

    # Get all migration files (both with and without _up suffix)
    all_migrations=$(ls -1 $MIGRATIONS_DIR/*.sql 2>/dev/null | grep -v '_down.sql' | sort -V)

    if [ -z "$all_migrations" ]; then
        echo -e "${YELLOW}No migration files found${NC}"
        return 0
    fi

    for migration_file in $all_migrations; do
        # Extract version from filename (e.g., 001, 002, etc.)
        version=$(basename "$migration_file" | sed 's/_.*$//')
        filename=$(basename "$migration_file")

        if is_migration_applied "$version"; then
            echo -e "${GREEN}✓${NC} $version - $filename ${GREEN}(already applied)${NC}"
        else
            echo -e "${YELLOW}→${NC} Applying $version - $filename"
            if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -f "$migration_file" -q; then
                mark_migration_applied "$version"
                echo -e "${GREEN}✓${NC} $version - $filename ${GREEN}(applied successfully)${NC}"
            else
                echo -e "${RED}✗${NC} Migration $version failed"
                exit 1
            fi
        fi
    done

    echo ""
    echo -e "${GREEN}✓ All migrations completed successfully${NC}"
}

# Function to run migrations down
migrate_down() {
    echo -e "${YELLOW}Running last migration DOWN...${NC}"

    # Get last applied migration
    last_version=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -t -c "SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1;" 2>/dev/null | tr -d ' ')

    if [ -z "$last_version" ]; then
        echo -e "${YELLOW}No migrations to rollback${NC}"
        return 0
    fi

    # Find corresponding down migration
    down_file=$(ls -1 $MIGRATIONS_DIR/${last_version}_*_down.sql 2>/dev/null | head -n 1)

    if [ -z "$down_file" ]; then
        echo -e "${RED}✗ No down migration found for version $last_version${NC}"
        exit 1
    fi

    echo -e "${YELLOW}→${NC} Rolling back migration $last_version: $(basename $down_file)"
    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -f "$down_file" -q; then
        unmark_migration "$last_version"
        echo -e "${GREEN}✓${NC} Migration $last_version rolled back successfully"
    else
        echo -e "${RED}✗${NC} Rollback of migration $last_version failed"
        exit 1
    fi
}

# Function to show migration status
show_status() {
    echo -e "${YELLOW}Migration Status:${NC}"
    echo ""

    # Get all migration files
    for migration_file in $(ls -1 $MIGRATIONS_DIR/*.sql 2>/dev/null | grep -v '_down.sql' | sort -V); do
        version=$(basename "$migration_file" | sed 's/_.*$//')
        filename=$(basename "$migration_file")

        if is_migration_applied "$version"; then
            applied_at=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -t -c "SELECT applied_at FROM schema_migrations WHERE version = '$version';" 2>/dev/null | tr -d ' ')
            echo -e "${GREEN}✓${NC} $version - $filename ${GREEN}(applied at $applied_at)${NC}"
        else
            echo -e "${RED}✗${NC} $version - $filename ${RED}(not applied)${NC}"
        fi
    done

    echo ""
    applied_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USERNAME" -d "$DB_DATABASE" -t -c "SELECT COUNT(*) FROM schema_migrations;" 2>/dev/null | tr -d ' ')
    total_count=$(ls -1 $MIGRATIONS_DIR/*.sql 2>/dev/null | grep -v '_down.sql' | wc -l | tr -d ' ')
    echo -e "${YELLOW}Applied: $applied_count / $total_count migrations${NC}"
}

# Main script
case "${1:-up}" in
    up)
        check_db_connection
        create_migrations_table
        migrate_up
        ;;
    down)
        check_db_connection
        create_migrations_table
        migrate_down
        ;;
    status)
        check_db_connection
        create_migrations_table
        show_status
        ;;
    *)
        echo "Usage: $0 [up|down|status]"
        echo ""
        echo "Commands:"
        echo "  up     - Apply all pending migrations (default)"
        echo "  down   - Rollback last applied migration"
        echo "  status - Show migration status"
        exit 1
        ;;
esac
