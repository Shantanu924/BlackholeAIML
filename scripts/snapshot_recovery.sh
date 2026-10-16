#!/bin/bash
# VAJRA Mainnet - Disaster Recovery & Snapshot Script

COMMAND=$1
BACKUP_DIR="/var/backups/vajra"
DATE=$(date +%Y%m%d_%H%M%S)

case "$COMMAND" in
  "snapshot")
    echo "📸 Taking LevelDB snapshot of VAJRA Mainnet nodes..."
    mkdir -p $BACKUP_DIR
    
    # Pause node processes gracefully (if running under systemd, you'd systemctl stop vajra-relay)
    echo "⚠️ Warning: Make sure nodes are paused or traffic is routed away before snapshotting live DBs!"
    
    # Archive the core relay chain database
    if [ -d "../core/relay-chain/blockchaindb_8080" ]; then
        tar -czf "$BACKUP_DIR/relay_db_$DATE.tar.gz" -C ../core/relay-chain blockchaindb_8080
        echo "✅ Relay Chain database snapshot saved to $BACKUP_DIR/relay_db_$DATE.tar.gz"
    else
        echo "❌ Blockchain DB not found at expected path."
    fi

    # Archive the wallet database
    if [ -d "../services/wallet/data" ]; then
        tar -czf "$BACKUP_DIR/wallet_db_$DATE.tar.gz" -C ../services/wallet data
        echo "✅ Wallet database snapshot saved to $BACKUP_DIR/wallet_db_$DATE.tar.gz"
    fi
    ;;
    
  "restore")
    SNAPSHOT_FILE=$2
    if [ -z "$SNAPSHOT_FILE" ]; then
        echo "❌ Usage: ./snapshot_recovery.sh restore <snapshot_tar_gz>"
        exit 1
    fi
    
    if [ ! -f "$SNAPSHOT_FILE" ]; then
        echo "❌ Snapshot file not found: $SNAPSHOT_FILE"
        exit 1
    fi

    echo "⚠️ WARNING: This will overwrite current node data. Press Ctrl+C to cancel."
    sleep 3

    echo "🔄 Restoring snapshot..."
    # Assuming snapshot is relay_db
    tar -xzf "$SNAPSHOT_FILE" -C ../core/relay-chain
    echo "✅ Restore complete! Please verify integrity before starting node."
    ;;
    
  *)
    echo "Usage: ./snapshot_recovery.sh [snapshot|restore <file>]"
    echo "  snapshot - Archives current LevelDB state"
    echo "  restore  - Restores a previous LevelDB tar.gz archive"
    exit 1
    ;;
esac
