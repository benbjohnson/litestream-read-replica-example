#!/bin/bash

# Determine which configuration file to use based on region.
if [ "$FLY_REGION" == "$FLY_PRIMARY_REGION" ]
then
	REPLICATION_MODE="primary"
else
	REPLICATION_MODE="replica"
fi

echo "Starting instance as ${REPLICATION_MODE}"

# Use correct configuration file based on replication mode.
mv /etc/litestream.${REPLICATION_MODE}.yml /etc/litestream.yml

# Create an empty database if we are a replica since our application will open as read-only.
if [ "$REPLICATION_MODE" == "replica" ] && [ ! -f "$DSN" ]
then
	echo "Initializing empty replica database"
	sqlite3 "$DSN" "PRAGMA journal_mode = wal"
fi

echo "Starting Litestream & application"

# Start litestream and the main application
litestream replicate -exec "/usr/local/bin/litestream-read-replica-example -dsn "$DSN" -addr :8080"
