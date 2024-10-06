package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go-proxy/internal/config/consts"

	"go-proxy/internal/service"
)

func Init(e *echo.Echo, appService service.AppService) {

	initHealthController(e, appService)

}

func initHealthController(e *echo.Echo, _ service.AppService) {

	// handler := func(c echo.Context) error {
	// 	ctrl := controller.NewHealthController(appService, c)
	// 	return ctrl.Check()
	// }
	// //
	// e.GET(consts.PathTestHealthAPI, handler)
	//
	e.GET(consts.PathTestPingAPI, func(c echo.Context) error { return c.String(http.StatusOK, "pong") })
	//
}

/////////////////////////////////////////////////////
