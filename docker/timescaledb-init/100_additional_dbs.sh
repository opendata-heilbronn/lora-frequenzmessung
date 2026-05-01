if [ -z "${POSTGRESQL_PASSWORD:-}" ]; then
	POSTGRESQL_PASSWORD=${POSTGRES_PASSWORD:-}
fi
export PGPASSWORD="$POSTGRESQL_PASSWORD"

MGMT_USER="${MANAGEMENT_DB_USER:-management}"
MGMT_PASS="${MANAGEMENT_DB_PASSWORD:-management}"
echo "select 'create user ${MGMT_USER} with password ''${MGMT_PASS}''' where not exists (SELECT FROM pg_user WHERE usename = '${MGMT_USER}')\gexec" | psql -U "${POSTGRES_USER}" postgres
echo "select 'create database management with owner ${MGMT_USER}' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'management')\gexec" | psql -U "${POSTGRES_USER}" postgres
