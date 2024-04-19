#!/bin/sh

# Default to secure settings
export PGADMIN_CONFIG_SERVER_MODE=True
export PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=True

# Check if environment is set to 'dev' and adjust settings
if [ "$ENVIRONMENT" = "dev" ]; then
    export PGADMIN_CONFIG_SERVER_MODE=False
    export PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED=False
fi


SERVERS_JSON_PATH="/pgadmin4/servers.json"
PGPASSFILE="$HOME/.pgpass"

# Create the .pgpass file for password
echo "Creating pgpass file at $PGPASSFILE"
echo "${POSTGRES_HOST}:*:*:${POSTGRES_USER}:${POSTGRES_PASSWORD}" > "$PGPASSFILE"
chmod 600 $PGPASSFILE
if [ "$ENVIRONMENT" = "dev" ]; then
    cat $PGPASSFILE
fi
echo "pgpass file created successfully."

echo "Creating servers.json file in $SERVERS_JSON_PATH"
cat << EOF > $SERVERS_JSON_PATH
{
    "Servers": {
        "1": {
            "Name": "Primary Database",
            "Group": "Servers",
            "Host": "${POSTGRES_HOST}",
            "Port": 5432,
            "MaintenanceDB": "${POSTGRES_DB}",
            "Username": "${POSTGRES_USER}",
            "PassFile": "$PGPASSFILE",
            "SSLMode": "prefer"
        }
    }
}
EOF

echo "$SERVERS_JSON_PATH file created successfully."
cat $SERVERS_JSON_PATH

echo "Starting pgAdmin4..."
exec /entrypoint.sh
echo "pgAdmin4 started."

