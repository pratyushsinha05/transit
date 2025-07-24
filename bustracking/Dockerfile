# Use the official Postgres image as base
FROM postgres:15

# Copy the init script into the automatic initialization folder
COPY init.sql /docker-entrypoint-initdb.d/

# Set environment variables (can also be handled in docker-compose)
ENV POSTGRES_USER=postgres
ENV POSTGRES_PASSWORD=postgres
ENV POSTGRES_DB=busdb