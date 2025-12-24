package middleware

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

// Recovery middleware recovers from panics and logs the stack trace
func Recovery(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic and stack trace
				log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())

				// Return 500 error
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				c.Error(echo.NewHTTPError(500, fmt.Sprintf("internal server error: %v", err)))
			}
		}()

		return next(c)
	}
}
