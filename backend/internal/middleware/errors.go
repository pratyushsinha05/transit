package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func HTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Internal Server Error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = result(he.Message)
	}

	c.Logger().Error(err)
	_ = c.JSON(code, map[string]string{"error": message})
}

func result(message interface{}) string {
	if s, ok := message.(string); ok {
		return s
	}
	return "unknown error"
}
