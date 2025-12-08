package geo

import "math"

const (
	EarthRadiusKm = 6371.0
	MinSpeedKmh   = 1.0
	DefaultSpeed  = 20.0
)

// Haversine calculates the distance between two points in kilometers.
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLng := (lng2 - lng1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLng/2)*math.Sin(dLng/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EarthRadiusKm * c
}

// CalculateETA calculates the estimated time of arrival in minutes.
func CalculateETA(fromLat, fromLng, toLat, toLng, speedKmh float64) int {
	distanceKm := Haversine(fromLat, fromLng, toLat, toLng)

	effectiveSpeed := speedKmh
	if effectiveSpeed < MinSpeedKmh {
		effectiveSpeed = DefaultSpeed
	}

	timeHours := distanceKm / effectiveSpeed
	return int(math.Round(timeHours * 60))
}
