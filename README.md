# LatitudeX Backend

A minimal FastAPI backend that prints incoming GPS pings from the mobile webapp.

## Setup

1. Ensure you are in the parent directory of LatitudeX.backend
2. Navigate to project directory:
   ```bash
   cd ${PROJECT_DIR}
   ```
3. Build Docker image:
   ```bash
   docker build -t latitudex-backend:latest .
   ```
4. Run container:
   ```bash
   docker run --rm -p 8000:8000 latitudex-backend:latest
   ```

## Endpoints

- **GET /** → Welcome message
- **GET /health** → {"status": "ok"}
- **POST /api/ping** → Accepts JSON `{bus_id, lat, lon, ts}`, prints ping
- **WebSocket /ws** → Returns a welcome message and closes

## Testing the ping endpoint

```bash
curl -X POST http://localhost:8000/api/ping \
  -H 'Content-Type: application/json' \
  -d '{"bus_id":1,"lat":12.34,"lon":56.78,"ts":"2025-07-13T12:00:00Z"}'
```
