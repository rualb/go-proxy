package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go-proxy/internal/config/consts"

	"go-proxy/internal/service"

	xlog "go-proxy/internal/tool/toollog"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4/middleware"
)

func Init(e *echo.Echo, appService service.AppService) {

	initHealthController(e, appService)

	initSys(e, appService)
}

func initSys(e *echo.Echo, appService service.AppService) {

	// !!! DANGER for private(non-public) services only
	// or use non-public port via echo.New()

	appConfig := appService.Config()

	listen := appConfig.HTTPServer.Listen
	listenSys := appConfig.HTTPServer.ListenSys
	hasAnyService := false
	sysAPIKey := appConfig.HTTPServer.SysAPIKey
	hasAPIKey := sysAPIKey != ""
	hasListenSys := listenSys != ""
	startNewListener := listenSys != listen

	if !hasListenSys {
		return
	} else {
		xlog.Info("Sys api serve on: %v main: %v", listenSys, listen)
	}

	if !hasAPIKey {
		xlog.Panic("Sys api key is empty")
		return
	}

	if startNewListener {

		e = echo.New() // overwrite override

		e.Use(middleware.Recover())
		// e.Use(middleware.Logger())
	} else {
		xlog.Warn("Sys api serve in main listener: %v", listen)
	}

	sysAPIAccessAuthMW := middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "query:api-key,header:Authorization",
		Validator: func(key string, c echo.Context) (bool, error) {
			return key == sysAPIKey, nil
		},
	})

	if appConfig.HTTPServer.SysMetrics {
		hasAnyService = true
		// may be eSys := echo.New() // this Echo will run on separate port
		e.GET(
			consts.PathSysMetricsAPI,
			echoprometheus.NewHandler(),
			sysAPIAccessAuthMW,
		) // adds route to serve gathered metrics

	}

	if hasAnyService {

		if startNewListener {

			// start as async task
			go func() {
				xlog.Info("Starting sys api server: %v", listenSys)

				if err := e.Start(listenSys); err != nil {
					if err != http.ErrServerClosed {
						xlog.Error("%v", err)
					} else {
						xlog.Info("shutting down the server")
					}
				}
			}()

		} else {
			xlog.Info("Sys api server serve on main listener: %v", listen)
		}

	} else {
		xlog.Warn("No any active service for sys api")
	}

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
