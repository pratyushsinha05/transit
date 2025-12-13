# Use the official Postgres image as base
FROM postgres:16-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
      wget gnupg ca-certificates \
 && echo "deb http://apt.postgresql.org/pub/repos/apt bookworm-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
 && wget -qO- https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
 && apt-get update && apt-get install -y --no-install-recommends \
      postgresql-$PG_MAJOR-postgis-3 \
      postgresql-$PG_MAJOR-postgis-3-scripts \
      postgresql-$PG_MAJOR-pgrouting \
 && rm -rf /var/lib/apt/lists/*


# Copy the init script into the automatic initialization folder
COPY init.sql /docker-entrypoint-initdb.d/

# Set environment variables (can also be handled in docker-compose)
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=busdb