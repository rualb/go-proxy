package middleware

import (
	"go-proxy/internal/config"
	xlog "go-proxy/internal/util/utillog"
	webfs "go-proxy/web"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/oschwald/geoip2-golang"
)

func NewGeoIP(cfg config.AppConfigGeoIP) echo.MiddlewareFunc {

	if cfg.File == "" {
		xlog.Panic("gis data file is empty")
	}
	filename, err := filepath.Abs(cfg.File)

	if err != nil {
		xlog.Panic("gis data file: %v error: %v", filename, err)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		xlog.Panic("gis data file: %v error: %v", filename, err)
	}

	xlog.Info("gis data file: %v", filename)

	handler := &gisHandler{}
	handler.mustOpenData(filename)
	handler.loadLists(cfg.AllowCountry, cfg.BlockCountry)

	if len(cfg.AllowCountry) > 0 {
		xlog.Info("allow country: %v", cfg.AllowCountry)
	}

	if len(cfg.BlockCountry) > 0 {
		xlog.Info("block country: %v", cfg.BlockCountry)
	}

	// defer handler.closeDb()

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(c echo.Context) error {

			ipStr := c.RealIP()
			countryCode := handler.ipToCountry(ipStr)

			// c.Request().Header.Del("X-Country-Code")
			c.Request().Header.Set("X-Country-Code", countryCode) // map[string][]string

			{

				// c.Set("country", countryCode)

				if handler.isBlocked(countryCode) {
					// block
					data, _ := webfs.Status(http.StatusUnavailableForLegalReasons)

					// c.Response().Header().Set("X-Country-Code", countryCode) //

					return c.HTMLBlob(http.StatusUnavailableForLegalReasons, data)

				}
			}

			return next(c)
		}

	}

}

type gisHandler struct {
	allowList map[string]bool // country qw,er
	blockList map[string]bool // country qw,er
	db        *geoip2.Reader
}

func (x *gisHandler) mustOpenData(filename string) {
	var err error
	x.db, err = geoip2.Open(filename)
	if err != nil {
		xlog.Panic("gis data file: %v error: %v", filename, err)
	}
}

//	func (x *gisHandler) closeDb() {
//		if x.db == nil {
//			x.db.Close()
//		}
//	}

func (x *gisHandler) ipToCountry(ipStr string) string {

	res := ""
	if x.db == nil {
		return res
	}
	ip := net.ParseIP(ipStr)
	country, err := x.db.Country(ip)

	if err != nil {
		xlog.Debug("ip to country IP: %v error: %v", ipStr, err)
	}

	if country != nil {
		res = country.Country.IsoCode // Upper case QQ iso code-2
		// res = strings.ToLower(res)
	}

	return res
}

func (x *gisHandler) isBlocked(countryCode string) bool {

	if len(x.allowList) > 0 {
		return !x.allowList[countryCode]
	}

	if len(x.blockList) > 0 {
		return x.blockList[countryCode]
	}

	return false

}

func (x *gisHandler) loadLists(allowList []string, blockList []string) {

	x.allowList = map[string]bool{} // country qw,er
	x.blockList = map[string]bool{} // country qw,er

	for _, v := range allowList {
		x.allowList[v] = true
	}

	for _, v := range blockList {
		x.blockList[v] = true
	}
	{
		// 127.0.0.1 and etc
		// countryMap[""] = true
	}

}

// func isLocalIP(ipStr string) bool {

// 	return strings.HasPrefix(ipStr, "127.0.0.") ||
// 		strings.HasPrefix(ipStr, "10.") ||
// 		strings.HasPrefix(ipStr, "192.168.0.")
// }
