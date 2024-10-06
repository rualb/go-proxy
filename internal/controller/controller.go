package controller

import (
	"github.com/labstack/echo/v4"
)

func IsGET(c echo.Context) bool {
	return c.Request().Method == "GET"
}

func IsPOST(c echo.Context) bool {
	return c.Request().Method == "POST"
}
