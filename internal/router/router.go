package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"go-proxy/internal/config/consts"

	"go-proxy/internal/service"

	xlog "go-proxy/internal/util/utillog"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4/middleware"
)

func Init(e *echo.Echo, appService service.AppService) {

	initDebugController(e, appService)

	initSys(e, appService)
}
func initSys(e *echo.Echo, appService service.AppService) {

	// !!! DANGER for private(non-public) services only
	// or use non-public port via echo.New()

	appConfig := appService.Config()

	listen := appConfig.HTTPServer.Listen
	listenSys := appConfig.HTTPServer.ListenSys
	sysMetrics := appConfig.HTTPServer.SysMetrics
	hasAnyService := sysMetrics
	sysAPIKey := appConfig.HTTPServer.SysAPIKey
	hasAPIKey := sysAPIKey != ""
	hasListenSys := listenSys != ""
	startNewListener := listenSys != listen

	if !hasListenSys {
		return
	}

	if !hasAnyService {
		return
	}

	if !hasAPIKey {
		xlog.Panic("sys api key is empty")
		return
	}

	if startNewListener {

		e = echo.New() // overwrite override

		e.Use(middleware.Recover())
		// e.Use(middleware.Logger())
	} else {
		xlog.Warn("sys api serve in main listener: %v", listen)
	}

	sysAPIAccessAuthMW := middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "query:api-key,header:Authorization",
		Validator: func(key string, c echo.Context) (bool, error) {
			return key == sysAPIKey, nil
		},
	})

	if sysMetrics {
		// may be eSys := echo.New() // this Echo will run on separate port
		e.GET(
			consts.PathSysMetricsAPI,
			echoprometheus.NewHandler(),
			sysAPIAccessAuthMW,
		) // adds route to serve gathered metrics

	}

	if startNewListener {

		// start as async task
		go func() {
			xlog.Info("sys api serve on: %v main: %v", listenSys, listen)

			if err := e.Start(listenSys); err != nil {
				if err != http.ErrServerClosed {
					xlog.Error("%v", err)
				} else {
					xlog.Info("shutting down the server")
				}
			}
		}()

	} else {
		xlog.Info("sys api server serve on main listener: %v", listen)
	}

}

func initDebugController(e *echo.Echo, _ service.AppService) {
	e.GET(consts.PathProxyPingDebugAPI, func(c echo.Context) error { return c.String(http.StatusOK, "pong") })
	//
	// status
	e.GET("/-/health", func(c echo.Context) error { return c.JSON(http.StatusOK, struct{}{}) })
	//
	e.GET("/-/probe/startup", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	e.GET("/-/probe/ready", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	e.GET("/-/probe/live", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	// curl -X POST -H "Content-Type: application/json" -d '{}' http://127.0.0.1/proxy/api/status
	// {"message":"missing csrf token in the form parameter"}
	// csrf check

	e.GET(consts.PathProxyStatusDebugAPI, func(c echo.Context) error { return c.JSON(http.StatusOK, struct{}{}) })
	e.POST(consts.PathProxyStatusDebugAPI, func(c echo.Context) error { return c.JSON(http.StatusOK, struct{}{}) })
}

/////////////////////////////////////////////////////
