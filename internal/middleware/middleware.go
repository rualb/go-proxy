package middleware

import (
	"fmt"
	"go-proxy/internal/config/consts"
	"go-proxy/internal/service"
	"go-proxy/internal/util/utilhttp"
	xlog "go-proxy/internal/util/utillog"
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

	initGeoIP(e, appService) // .Pre

	if appConfig.HTTPServer.AccessLog {

		cfg := middleware.DefaultLoggerConfig

		if appConfig.GeoIP.Enabled {
			f := cfg.Format
			li := strings.LastIndex(f, "}") // insert before last "}"
			f = f[:li] + `,"country":"${header:x-country-code}"` + f[li:]
			cfg.Format = f
		}

		cfg.Skipper = func(c echo.Context) bool {
			return c.Request().URL.Path == "/favicon.ico"
		}

		e.Use(middleware.LoggerWithConfig(cfg))
	}

	initMaintenance(e, appService)
	initRedirect(e, appService)
	initContentSecurity(e, appService)
	initRateLimit(e, appService)
	initRequestID(e, appService)

	initProxy(e, appService)

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
									xlog.Error("error on get page: %v", err)
								}

								// TODO not good content-length:0
								err = c.HTMLBlob(http.StatusServiceUnavailable, data)

								if err != nil {
									xlog.Error("error on send data: %v", err)
								}

							}
						case strings.HasPrefix(redirect, "/"):
							{
								currentURL := c.Request().URL.String()
								URL, err := utilhttp.JoinURL(redirect,
									map[string]string{"next": currentURL},
								)

								if err != nil {
									xlog.Error("error on url: %v", err)
								}

								// TODO redirect not work here
								err = c.Redirect(http.StatusSeeOther, URL)
								if err != nil {
									xlog.Error("error on redirect: %v", err)
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
	// cfg := middleware.DefaultRedirectConfig
	// cfg.Skipper = func(c echo.Context) bool {
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

	// X-Frame-Options: SAMEORIGIN; => Content-Security-Policy: frame-ancestors 'self';
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

	if cs.CSRF {
		// 1
		e.Use(csrfAddCookieAndValidateMiddleware(appService))
		// 2
		e.Use(csrfAddHeaderMiddleware(appService))

		xlog.Info("csrf middleware enabled")
	} else {
		xlog.Warn("csrf middleware disabled")
	}

	if cs.BodyLimit != "" {
		e.Use(middleware.BodyLimit(cs.BodyLimit))

		xlog.Warn("body limit is: %v", cs.BodyLimit)
	} else {
		xlog.Warn("body limit is empty")
	}

	if len(cs.AllowOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: cs.AllowOrigins,
		}))

		xlog.Info("allow origins: %v", cs.AllowOrigins)
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
			xlog.Info("headers del: %v", headersDel)
		}
		if len(headersAdd) > 0 {
			handlers = append(handlers, func(r *echo.Response) {
				h := r.Header()
				for _, v := range headersAdd {
					h.Add(v[0], v[1])
				}
			})
			xlog.Info("headers add: %v", headersAdd)
		}
		if contentPolicy != "" {
			handlers = append(handlers, func(r *echo.Response) {
				h := r.Header()
				// if text/html; charset=utf-8
				if strings.HasPrefix(h.Get(echo.HeaderContentType), echo.MIMETextHTML) {
					h.Add(echo.HeaderContentSecurityPolicy, contentPolicy)
				}
			})
			xlog.Info("content policy: %v", contentPolicy)
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

// MiddlewareFunc to add a header if the request path ends with /status
func csrfAddHeaderMiddleware(_ service.AppService) echo.MiddlewareFunc {

	anyAuthStatusReq := func(c echo.Context) bool {
		// Check if the request path contains `/assets/` and follows the pattern `/*/assets/*`
		path := c.Request().URL.Path // c.Path() return pattern like /blog/*
		return c.Request().Method == http.MethodGet &&
			consts.PathAuthStatusAPI == path
		// strings.HasSuffix(path, "/status")
	}

	csrfAddHeader := func(c echo.Context) {
		csrf, _ := c.Get("_csrf").(string)
		if csrf != "" {
			c.Response().Header().Set("X-CSRF-Token", csrf)
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if the request path ends with "/status"
			if anyAuthStatusReq(c) {
				// Add header
				csrfAddHeader(c)
			}
			// Continue to the next handler
			return next(c)
		}
	}
}

// csrfAddCookieAndValidateMiddleware adds CSRF cookies for `/status` requests
// and validates CSRF tokens for unsafe methods.
func csrfAddCookieAndValidateMiddleware(_ service.AppService) echo.MiddlewareFunc {

	//

	anyAuthStatusReq := func(c echo.Context) bool {
		// Check if the request path contains `/assets/` and follows the pattern `/*/assets/*`
		path := c.Request().URL.Path // c.Path() return pattern like /blog/*
		return c.Request().Method == http.MethodGet &&
			consts.PathAuthStatusAPI == path
		// strings.HasSuffix(path, "/status")
	}

	// Check if the HTTP method is "safe".
	isSafeMethod := func(c echo.Context) bool {
		switch c.Request().Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			return true
		}
		return false
	}

	anyAssetsReq := func(c echo.Context) bool {
		// Check if the request path contains `/assets/` and follows the pattern `/*/assets/*`
		path := c.Request().URL.Path // c.Path() return pattern like /blog/*
		return strings.Contains(path, "/assets/")

	}

	// Skipper logic for excluding middleware on certain requests.
	skipper := func(c echo.Context) bool {
		if anyAssetsReq(c) {
			return true // Skip middleware for `/assets/` requests.
		}
		if isSafeMethod(c) && !anyAuthStatusReq(c) {
			return true // Skip middleware for safe methods except `/status`.
		}
		return false // Do not skip middleware for other cases.
	}

	csrfConfig := middleware.CSRFConfig{
		Skipper: skipper,

		TokenLookup: "header:X-CSRF-Token,form:_csrf",
		CookiePath:  "/",
		// CookieDomain:   "example.com",
		// CookieSecure:   true, // https only
		CookieHTTPOnly: true,
		CookieName:     "_csrf",
		ContextKey:     "_csrf",
		CookieSameSite: http.SameSiteDefaultMode,
	}

	return middleware.CSRFWithConfig(csrfConfig)

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
			ExpiresIn: 60 * time.Second,
		}

		xlog.Info("starting rate control, store config: %v", rateStoreConfig)

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
	} else {
		xlog.Warn("rate limit not active")
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

		for _, upstream := range appConfig.Proxy.Upstreams {

			// httputil.NewSingleHostReverseProxy(serverURL)

			trg, err := newProxyUpstream(upstream)
			if err != nil {
				xlog.Panic("error on try add proxy upstream: %v", err)
			}

			{

				balancer := middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{})

				for _, v := range trg.server {
					xlog.Info("adding proxy upstream: %v => %v", trg.prefix, v)
					serverURL, err := url.Parse(v) // downstream
					if err != nil {
						panic(fmt.Errorf("error on parse proxy upstream %v: %v", upstream, err))
					}
					// .NewRandomBalancer()
					balancer.AddTarget(&middleware.ProxyTarget{
						URL:  serverURL,
						Name: v, // !!! server ID
					})
				}

				proxyConfig := middleware.DefaultProxyConfig
				proxyConfig.Balancer = balancer
				// proxyConfig.RetryCount = 0 // 0, meaning requests are never retried
				proxyConfig.RetryCount = len(trg.server) - 1
				proxyConfig.Rewrite = trg.rewrite
				proxyConfig.ErrorHandler = func(c echo.Context, err error) error {
					return err
				}
				proxyConfig.ModifyResponse = func(r *http.Response) error {
					return nil
				}
				funcMw := middleware.ProxyWithConfig(proxyConfig)
				e.RouteNotFound(trg.prefix, nil, funcMw)
			}
		}

	}

}

type proxyUpstream struct {
	server  []string
	prefix  string
	rewrite map[string]string
}

func newProxyUpstream(upstream string) (*proxyUpstream, error) {

	upstream = strings.TrimSpace(upstream)
	// parts := strings.SplitN(upstream, " ", 2)
	// upstream = strings.TrimSpace(parts[0])
	// http://127.0.0.1:10082/test2?server=127.0.0.1:10083

	parsedURL, err := url.Parse(upstream)

	if err != nil {
		return nil, fmt.Errorf("error on parse proxy upstream %v: %v", upstream, err)
		// panic()
	}

	r := &proxyUpstream{}
	r.server = append(r.server,
		fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host /*has port*/),
	)
	args := parsedURL.Query()

	{
		// extra servers
		serverExt := args["server"]
		for _, v := range serverExt {
			r.server = append(r.server,
				fmt.Sprintf("%s://%s", parsedURL.Scheme, v /*has port*/),
			)
		}
	}

	{
		rewrite := args["rewrite"]

		if len(rewrite) > 0 {
			xlog.Info("rewrite path conds: %+v", rewrite)
		}

		for _, v := range rewrite {
			parts := strings.SplitN(v, ":", 2)
			if len(parts) != 2 {
				xlog.Error("[ERROR] cannot parse token for rewrite: %s", v)
				continue // may be panic
			}

			if len(r.rewrite) == 0 {
				r.rewrite = map[string]string{}
			}
			r.rewrite[parts[0]] = parts[1]
		}

		if len(rewrite) > 0 {
			xlog.Info("rewrite path map:%+v", r.rewrite)
		}
	}

	r.prefix = parsedURL.Path

	return r, nil
}
