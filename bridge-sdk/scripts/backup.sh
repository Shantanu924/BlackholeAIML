#!/usr/bin/env bash
# BlackHole Bridge SDK Production Backup Script
# Automatically dumps database states and archives them with a 7-day retention policy.

set -euo pipefail

# Configurations
BACKUP_DIR="./backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
OUTPUT_ARCHIVE="${BACKUP_DIR}/bridge_backup_${TIMESTAMP}.tar.gz"
TEMP_DIR="${BACKUP_DIR}/temp_${TIMESTAMP}"

# Database paths
BOLT_DB_PATH="./main_bridge/data"
POSTGRES_CONTAINER="bridge-postgres"
POSTGRES_USER="bridge"
POSTGRES_DB="bridge_db"

echo "📂 Starting BlackHole Bridge backup task at $(date)..."

# Ensure backup directory exists
mkdir -p "${BACKUP_DIR}"
mkdir -p "${TEMP_DIR}"

# 1. Back up BoltDB databases (persistent cache / event deduplication states)
if [ -d "${BOLT_DB_PATH}" ]; then
    echo "💾 Backing up BoltDB files from ${BOLT_DB_PATH}..."
    cp -r "${BOLT_DB_PATH}" "${TEMP_DIR}/boltdb"
else
    echo "⚠️ BoltDB directory not found at ${BOLT_DB_PATH}, skipping BoltDB backup."
fi

# 2. Back up Postgres DB (using pg_dump inside running container if accessible)
if docker ps --filter "name=${POSTGRES_CONTAINER}" --format '{{.Names}}' | grep -q "${POSTGRES_CONTAINER}"; then
    echo "🐘 Container ${POSTGRES_CONTAINER} is running, dumping Postgres database..."
    docker exec "${POSTGRES_CONTAINER}" pg_dump -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" > "${TEMP_DIR}/postgres_dump.sql"
else
    echo "⚠️ Postgres container ${POSTGRES_CONTAINER} is not running or not accessible. Skipping PG dump."
fi

# 3. Save logs if available
if [ -d "./main_bridge/logs" ]; then
    echo "📝 Backing up system logs..."
    cp -r "./main_bridge/logs" "${TEMP_DIR}/logs"
fi

# 4. Compress to tarball
echo "📦 Packaging and compressing backup into ${OUTPUT_ARCHIVE}..."
tar -czf "${OUTPUT_ARCHIVE}" -C "${TEMP_DIR}" .

# 5. Clean up temporary files
rm -rf "${TEMP_DIR}"

# 6. Apply retention policy: Remove backups older than 7 days
echo "🧹 Applying retention policy (deleting backups older than 7 days)..."
find "${BACKUP_DIR}" -name "bridge_backup_*.tar.gz" -type f -mtime +7 -delete

echo "✅ Backup completed successfully!"
echo "Backup archive location: ${OUTPUT_ARCHIVE}"
