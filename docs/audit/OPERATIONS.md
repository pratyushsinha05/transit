# Transit POC: Operations

## 1. Quick Start & Execution
The repository abstracts all infrastructure complexity into a single shell script and a `Makefile`.

To run the entire system:
```bash
./deploy.sh
```
This single command:
1. Verifies prerequisites (Docker, Make, Node.js).
2. Uses `docker-compose` to spin up PostgreSQL, Redis, and the Go backend.
3. Builds the React/Vite frontend.
4. Starts a local frontend preview server on port 4173.

To tear down the environment (destroying databases and node_modules):
```bash
./cleanup.sh
```

## 2. CI/CD & Build Targets (Makefile)
The `Makefile` at the root acts as the central command registry:
- `make build`: Compiles the Go binary (`cmd/server/main.go`).
- `make docker-up`: Builds and boots the backend stack.
- `make rebuild-backend`: Hot-swaps the Go API container in Docker without bringing down Postgres or Redis, allowing for fast backend iteration.
- `make test-unit`: Executes the Go unit testing suite.

## 3. Configuration Management
Configuration is loaded dynamically on startup via the `github.com/joho/godotenv` package. 
Default environment variables are declared in `infra/docker-compose.yml`. Key toggles include:
- `DB_MAX_CONNS` / `DB_MIN_CONNS`: Connection pool limits.
- `H3_RESOLUTION`: Sets the global geospatial grid precision (defaults to 9).
- `LOG_LEVEL`: Adjusts stdout verbosity.

## 4. Test Harness and Coverage
- **Framework**: Standard Go `testing` package.
- **Coverage**: Severely lacking. The unit tests (`distance_test.go`, `geofencing_test.go`) achieve 100% coverage on pure math functions in `pkg/geo`, but all HTTP handlers, database queries, and WebSocket logic sit at **0% coverage**. There are no integration tests.
