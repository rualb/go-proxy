package middleware

import (
	"fmt"
	"go-proxy/internal/service"
	"go-proxy/internal/tool/toolhttp"
	xlog "go-proxy/internal/tool/toollog"
	webfs "go-proxy/web"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

var (
	// ErrRateLimitExceeded denotes an error raised when rate limit is exceeded
	ErrRateLimitExceeded = echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
	// ErrExtractorError denotes an error raised when extractor function is unsuccessful
	ErrExtractorError = echo.NewHTTPError(http.StatusForbidden, "error while extracting identifier")
)

func Init(e *echo.Echo, appService service.AppService) {

	appConfig := appService.Config()

	e.HTTPErrorHandler = newHTTPErrorHandler(appService)

	e.Use(middleware.Recover()) // !!!

	if appConfig.HTTPServer.AccessLog {

		cnf := middleware.DefaultLoggerConfig

		if appConfig.GeoIP.Enabled {
			f := cnf.Format
			li := strings.LastIndex(f, "}") // insert before last "}"
			f = f[:li] + `,"country":"${header:x-country-code}"` + f[li:]
			cnf.Format = f
		}

		cnf.Skipper = func(c echo.Context) bool {
			return c.Request().URL.Path == "/favicon.ico"
		}

		e.Use(middleware.LoggerWithConfig(cnf))
	}

	initMaintenance(e, appService)
	initRedirect(e, appService)
	initContentSecurity(e, appService)
	initRateLimit(e, appService)
	initRequestID(e, appService)

	initProxy(e, appService)

	initGeoIP(e, appService) // .Pre

	{
		// prevent log
		e.GET("/favicon.ico", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })
	}
}

func newHTTPErrorHandler(appService service.AppService) echo.HTTPErrorHandler {

	appConfig := appService.Config()
	overrideStatus := appConfig.Proxy.OverrideStatus

	return func(err error, c echo.Context) {

		var status int

		if len(overrideStatus) > 0 {

			resp := c.Response()
			isRespEmpty := resp.Size == 0 && !resp.Committed

			if isRespEmpty {

				if status == 0 {
					if err != nil {

						// 502
						if errE, ok := err.(*echo.HTTPError); ok {
							status = errE.Code
						}

					}
				}
				// 502 404
				if status > 0 { // && !committed
					if redirect, ok := overrideStatus[status]; ok {
						switch {
						case strings.HasSuffix(redirect, ".html"):
							{
								data, err := webfs.Page(redirect)

								if err != nil {
									xlog.Error("Error on get page: %v", err)
								}

								// TODO not good content-length:0
								err = c.HTMLBlob(http.StatusServiceUnavailable, data)

								if err != nil {
									xlog.Error("Error on send data: %v", err)
								}

							}
						case strings.HasPrefix(redirect, "/"):
							{
								currentURL := c.Request().URL.String()
								URL, err := toolhttp.JoinURL(redirect,
									map[string]string{"return_url": currentURL},
								)

								if err != nil {
									xlog.Error("Error on url: %v", err)
								}

								// TODO redirect not work here
								err = c.Redirect(http.StatusSeeOther, URL)
								if err != nil {
									xlog.Error("Error on redirect: %v", err)
								}
							}
						}

					}
				}

			}

		}

		// status == http.StatusNotFound // 404

		c.Echo().DefaultHTTPErrorHandler(err, c)

	}

}

func initMaintenance(e *echo.Echo, appService service.AppService) {

	appConfig := appService.Config()
	// appConfig.IsMaint = true
	if appConfig.IsMaint {

		e.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {

			return func(c echo.Context) error {

				// err := next(c)

				// return err

				data, err := webfs.Page("maint.html")
				if err != nil {
					return err
				}
				c.Response().Header().Set("Retry-After", strconv.Itoa(10)) // seconds

				return c.HTMLBlob(http.StatusServiceUnavailable, data)
			}

		})

	}

}

func initRedirect(e *echo.Echo, appService service.AppService) {

	appConfig := appService.Config()

	//
	// cnf := middleware.DefaultRedirectConfig
	// cnf.Skipper = func(c echo.Context) bool {
	// 	req, scheme := c.Request(), c.Scheme()
	// 	host := req.Host
	// 	// if http:// and has www. prefix or qwe.example.com has more than t
	// 	return scheme == "https" && (strings.HasPrefix(host, "www.") || len(strings.SplitN(host, ".", 3)) > 2)
	// }

	hasSubDomain := func(c echo.Context) bool {
		// if .Host has sub domain sub.example.com
		return (len(strings.SplitN(c.Request().Host, ".", 3)) > 2)
	}

	// e.Pre(middleware.HTTPSWWWRedirectWithConfig(middleware.RedirectConfig{Skipper: hasSubDomain}))
	//
	if appConfig.HTTPServer.RedirectHTTPS {
		e.Pre(middleware.HTTPSRedirectWithConfig(middleware.RedirectConfig{})) // may be 307
	}

	if appConfig.HTTPServer.RedirectWWW {
		e.Pre(middleware.WWWRedirectWithConfig(middleware.RedirectConfig{Skipper: hasSubDomain}))
	}
	//
	// e.Pre(middleware.HTTPSNonWWWRedirect())

}

func initContentSecurity(e *echo.Echo, appService service.AppService) {

	// middleware.SecureWithConfig()
	// middleware.Timeout()

	appConfig := appService.Config()
	cs := appConfig.HTTPServer

	// if cs.RequestTimeout > 0 {
	// 	// TODO
	// 	// // from sources
	// 	// // WARNING: Timeout middleware causes more problems than it solves.
	// 	// // should be first middleware as it messes with request Writer
	// 	// e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
	// 	// 	Timeout: time.Duration(cs.RequestTimeout) * time.Second,
	// 	// }))
	// }

	if cs.BodyLimit != "" {
		e.Use(middleware.BodyLimit(cs.BodyLimit))

		xlog.Warn("Body limit is: %v", cs.BodyLimit)
	} else {
		xlog.Warn("Body limit is empty")
	}

	if len(cs.AllowOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: cs.AllowOrigins,
		}))

		xlog.Info("Allow origins: %v", cs.AllowOrigins)
	}

	{
		headersDel := cs.HeadersDel
		headersAdd := [][]string{}
		contentPolicy := cs.ContentPolicy
		for _, v := range cs.HeadersAdd {
			parts := strings.SplitN(v, ":", 2) // name=value
			if len(parts) < 2 {
				continue
			}
			parts[1] = strings.TrimSpace(parts[1])
			headersAdd = append(headersAdd, parts)
		}
		handlers := []func(r *echo.Response){}
		if len(headersDel) > 0 {
			handlers = append(handlers, func(r *echo.Response) {
				h := r.Header()
				for _, v := range headersDel {
					h.Del(v)
				}
			})
			xlog.Info("Headers del: %v", headersDel)
		}
		if len(headersAdd) > 0 {
			handlers = append(handlers, func(r *echo.Response) {
				h := r.Header()
				for _, v := range headersAdd {
					h.Add(v[0], v[1])
				}
			})
			xlog.Info("Headers add: %v", headersAdd)
		}
		if contentPolicy != "" {
			handlers = append(handlers, func(r *echo.Response) {
				h := r.Header()
				// if text/html; charset=utf-8
				if strings.HasPrefix(h.Get(echo.HeaderContentType), echo.MIMETextHTML) {
					h.Add(echo.HeaderContentSecurityPolicy, contentPolicy)
				}
			})
			xlog.Info("Content policy: %v", contentPolicy)
		}

		if len(handlers) > 0 {

			handler := func(r *echo.Response) {
				for _, v := range handlers {
					v(r)
				}
			}

			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {

					r := c.Response()

					h := func() {
						handler(r)
					}

					r.Before(h)

					return next(c)
				}
			})

		}

	}

	// TODO proxy add content origin headers CorsAccessControlAllowOrigin
	// e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {

	// 	return func(c echo.Context) error {

	// 		// x-frame-options:
	// 		c.Response().Header().Set(
	// 			"Content-Security-Policy",
	// 			"*",
	// 			// "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: *.openstreetmap.org *.googleapis.com *.gstatic.com *.youtube.com",
	// 		)

	// 		err := next(c)

	// 		return err
	// 	}

	// })

}
func initRateLimit(e *echo.Echo, appService service.AppService) {
	// TODO for public use (rate limit, cache, headers time out)

	appConfig := appService.Config()

	rateLimit := appConfig.HTTPServer.RateLimit
	rateBurst := appConfig.HTTPServer.RateBurst

	if rateLimit > 0.000001 { //
		rateStoreConfig := middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(rateLimit), //   rate.Limit(10),
			Burst:     rateBurst,
			ExpiresIn: 30 * time.Second,
		}

		xlog.Info("Starting rate control, store config: %v", rateStoreConfig)

		rateLimiterConfig := middleware.RateLimiterConfig{

			Skipper: middleware.DefaultSkipper,

			Store: middleware.NewRateLimiterMemoryStoreWithConfig(
				rateStoreConfig,
			),

			IdentifierExtractor: func(ctx echo.Context) (string, error) {
				id := ctx.RealIP()
				return id, nil
			},

			ErrorHandler: func(context echo.Context, err error) error {
				return context.JSON(http.StatusForbidden, nil)
			},

			DenyHandler: func(context echo.Context, identifier string, err error) error {
				return context.JSON(http.StatusTooManyRequests, nil)
			},
		}

		e.Use(middleware.RateLimiterWithConfig(rateLimiterConfig))
	}
}

func initGeoIP(e *echo.Echo, appService service.AppService) {
	appConfig := appService.Config()

	if appConfig.GeoIP.Enabled {
		e.Pre(NewGeoIP(appConfig.GeoIP))
	}
	// req ID

}
func initRequestID(e *echo.Echo, appService service.AppService) {

	appConfig := appService.Config()

	// req ID

	if appConfig.HTTPServer.RequestID {
		e.Use(middleware.RequestID())
	}

}
func initProxy(e *echo.Echo, appService service.AppService) {

	appConfig := appService.Config()
	{

		for _, target := range appConfig.Proxy.Targets {

			target = strings.TrimSpace(target)
			parts := strings.SplitN(target, " ", 2)
			target = strings.TrimSpace(parts[0])

			parsedURL, err := url.Parse(target)

			if err != nil {
				panic(fmt.Errorf("error on parse proxy target %v: %v", target, err))
			}

			origin := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host /*has port*/)
			prefix := parsedURL.Path

			{

				xlog.Info("Adding proxy target: %v => %v", prefix, origin)

				targetURL, err := url.Parse(origin) // downstream
				if err != nil {
					panic(fmt.Errorf("error on parse proxy target %v: %v", target, err))
				}

				balancer := middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{})
				balancer.AddTarget(&middleware.ProxyTarget{
					URL: targetURL,
				})

				proxyConfig := middleware.DefaultProxyConfig

				proxyConfig.Balancer = balancer

				funcMw := middleware.ProxyWithConfig(proxyConfig)

				e.RouteNotFound(prefix, nil, funcMw)
			}
		}

	}

}
