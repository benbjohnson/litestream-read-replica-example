#!/bin/bash

# Determine which configuration file to use based on region.
if [ "$FLY_REGION" == "$FLY_PRIMARY_REGION" ]
then
	REPLICATION_MODE = "primary"
else
	REPLICATION_MODE = "replica"
fi

# Use correct configuration file based on replication mode.
mv /etc/litestream.${REPLICATION_MODE}.yml /etc/litestream.yml

# Restore database if we are the primary and the database doesn't exist.
if [ "$REPLICATION_MODE" == "primary" && ! -f "$DSN" ]
then
	litestream restore -v -if-replica-exists -o "$DSN" "$REPLICA_URL"
fi

# Create an empty database if we are a replica since our application will open as read-only.
if [ "$REPLICATION_MODE" == "replica" && ! -f "$DSN" ]
then
	sqlite3 "$DSN" "PRAGMA journal_mode = wal"
fi

# Start litestream and the main application
litestream replicate -exec "/usr/local/bin/litestream-read-replica-example -dsn "$DSN" -addr :8080"
