# Transit POC

This repository contains the full stack code for the Transit POC, including a Go backend and a React (Vite) frontend.

## Prerequisites

To run this project locally, ensure you have the following installed:

*   **Docker & Docker Compose**: For running the database, cache, and backend services.
*   **Go (1.21+)**: If you intend to build or test the backend locally outside of Docker.
*   **Node.js & npm**: For building and running the development server for the frontend.
*   **Make**: To use the provided Makefile commands for standardizing workflows.

---

## 🚀 One-Step Full Stack Deployment

For convenience, a `deploy.sh` script is provided to spin up both the backend and build the frontend in one go.

1.  **Make the script executable** (first time only):
    ```bash
    chmod +x deploy.sh
    ```
2.  **Run the script**:
    ```bash
    ./deploy.sh
    ```

This script will verify prerequisites, use the `Makefile` to bring up the backend Docker stack, change into the `frontend` directory, install dependencies, and build the React app.

---

## Backend Management (via Makefile)

The backend relies heavily on `make` commands to simplify Docker orchestration and development workflows.

### Quick Start (Backend Only)

```bash
# 1. Verify dependencies
make check-deps

# 2. Spin up the entire Docker stack (Postgres, Redis, Go Backend)
make docker-up

# 3. Verify all services are healthy and running
make docker-health
```

### Common `make` Commands

Here are the most useful commands exported by the `Makefile`:

*   **`make help`**: Shows a comprehensive list of all available commands.
*   **`make docker-down`**: Stops all containers associated with the stacked, but **preserves volumes (data)**.
*   **`make docker-clean`**: Stops containers and **destroys all volumes** (full reset).
*   **`make docker-logs`**: Tails logs for all running services (Ctrl+C to stop).
*   **`make db-migrate`**: Manually run database migrations within the Docker postgres container.
*   **`make db-seed`**: Seed the database with sample data.
*   **`make rebuild-backend`**: Rebuild and restart *only* the backend Go service without restarting the databases. Useful for fast iteration.

### Service Endpoints

When the stack is running, the services map to the following endpoints on your host machine:

*   **Backend API**: `http://localhost:8080`
*   **Health Check**: `http://localhost:8080/health`
*   **PostgreSQL**: `localhost:5432` (`transit_user` / `transit_password`)
*   **Redis**: `localhost:6379`

---

## Frontend Management

The frontend is a standard React application built using Vite.

### Quick Start (Frontend Only)

1.  Change into the frontend directory:
    ```bash
    cd frontend
    ```
2.  Install dependencies:
    ```bash
    npm install
    ```
3.  Start the development server:
    ```bash
    npm run dev
    ```

This will run the frontend on a local port (usually `http://localhost:5173`) with Hot Module Replacement (HMR) enabled.

### Building for Production

To create an optimized production build:

```bash
npm run build
```

This will output the compiled static assets into `frontend/dist`. You can locally preview this production build with `npm run preview`.
