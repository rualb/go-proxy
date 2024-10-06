package controller

import (
	"go-proxy/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthController struct {
	appService service.AppService
	webCtxt    echo.Context
}

func NewHealthController(appService service.AppService, c echo.Context) *HealthController {
	return &HealthController{
		appService: appService,
		webCtxt:    c,
	}
}

// Check health
// benchmark db ?cmd=db
// benchmark db ?cmd=db_raw
// get string of length ?cmd=length&length=10
// get stats db ?cmd=stats
func (x *HealthController) Check() error {

	c := x.webCtxt
	res := ""

	switch c.QueryParam("cmd") {

	case "ping":
		res = "pong"

	default:
		return c.String(http.StatusBadRequest, "cmd not defined")
	}

	return c.String(http.StatusOK, res)
}
