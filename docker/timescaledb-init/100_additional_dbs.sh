if [ -z "${POSTGRESQL_PASSWORD:-}" ]; then
	POSTGRESQL_PASSWORD=${POSTGRES_PASSWORD:-}
fi
export PGPASSWORD="$POSTGRESQL_PASSWORD"

echo "select 'create user management with password ''management''' where not exists (SELECT FROM pg_user WHERE usename = 'management')\gexec" | psql -U "${POSTGRES_USER}" postgres
echo "select 'create database management with owner management' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'management')\gexec" | psql -U "${POSTGRES_USER}" postgres
