# main.py
import json
from datetime import datetime
from typing import Optional

from fastapi import FastAPI, WebSocket
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, confloat, Field

app = FastAPI(
    title="LatitudeX Backend",
    description="Minimal FastAPI backend that prints incoming GPS pings.",
    version="1.0",
)

# allow your mobile app (served e.g. via file:// or localhost) to POST here
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],        # or ["http://localhost:3000"] if you host it
    allow_methods=["*"],
    allow_headers=["*"],
)

class Ping(BaseModel):
    bus_id: str
    lat: confloat(gt=-90, lt= 90)
    lon: confloat(gt=-180, lt=180)
    ts: Optional[datetime] = Field(default_factory=datetime.utcnow)

@app.get("/", tags=["Root"])
async def root():
    return {"message": "Welcome to LatitudeX Backend"}

@app.get("/health", tags=["Health"])
async def health():
    return {"status": "ok"}

@app.post("/api/ping", status_code=200)
async def post_ping(p: Ping):
    # simply print what we got
    print(
        f"Received ping → "
        f"bus_id={p.bus_id!r}, "
        f"lat={p.lat:.6f}, lon={p.lon:.6f}, "
        f"ts={p.ts.isoformat()}"
    )
    return {"status": "received"}

@app.websocket("/ws")
async def ws_endpoint(ws: WebSocket):
    await ws.accept()
    await ws.send_text("WebSocket endpoint ready—but no live data yet.")
    await ws.close()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8000, log_level="info")
