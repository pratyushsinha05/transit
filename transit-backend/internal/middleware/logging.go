package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

func Logging(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		err := next(c)
		if err != nil {
			c.Error(err)
		}

		stop := time.Now()
		latency := stop.Sub(start)

		req := c.Request()
		res := c.Response()

		log.Printf(`{"method":"%s","path":"%s","status":%d,"latency_ms":%d}`,
			req.Method, req.URL.Path, res.Status, latency.Milliseconds())

		return err
	}
}
