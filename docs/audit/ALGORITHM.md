# Transit POC: Algorithms

## 1. Spatial Indexing Strategy (H3 + PostGIS)
To achieve fast geospatial proximity queries at scale, Transit uses a hybrid approach:
- **H3 (Uber)**: Uses resolution 9 (approx 175m edge length). This allows O(1) integer/string matching for basic geofencing (e.g., checking if a bus is at a stop by seeing if their hex strings are identical).
- **PostGIS**: Used as a fallback for precise distance calculations (`ST_DWithin`), avoiding the bounding inaccuracies of hex grids on larger ranges.

## 2. Distance and ETA Calculation
The core prediction loop relies on pure geometric mathematics rather than live traffic routing engines (like OSRM or Google Maps).

### The Math: Haversine Formula
```go
// backend/pkg/geo/distance.go
func Haversine(lat1, lng1, lat2, lng2 float64) float64
```
Calculates the great-circle distance between two points on a sphere given their longitudes and latitudes.

### The ETA Algorithm
1. Find distance `d` via Haversine.
2. Read bus speed `v`.
3. If `v < 1.0 km/h`, assume `v = 20.0 km/h` (DefaultSpeed fallback to prevent infinite ETAs for stopped buses).
4. `ETA_minutes = Round((d / v) * 60)`.

## 3. "Is Approaching" Detection Loop
Defined in `backend/internal/services/arrivals.go`, this algorithm flags a bus as approaching a stop:
1. Determine `TargetStop` coordinates.
2. Calculate the H3 `k-ring` of radius 3 around the stop (this produces an array of 37 surrounding hex cells, covering approx 500m).
3. Calculate the current `Bus_Hex` of the approaching bus.
4. If `Bus_Hex` exists within the `k-ring` array, flag `isApproaching = true`.

*(Note: Due to a structural bug, this algorithm is currently orphaned and bypassed by the HTTP handler in favor of a simpler calculation.)*
