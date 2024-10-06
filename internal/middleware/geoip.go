package middleware

import (
	"go-proxy/internal/config"
	xlog "go-proxy/internal/tool/toollog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/oschwald/geoip2-golang"
)

func NewGeoIP(cnf config.AppConfigGeoIP) echo.MiddlewareFunc {

	if cnf.File == "" {
		xlog.Panic("GeoIp db file is empty")
	}
	filename, err := filepath.Abs(cnf.File)

	if err != nil {
		xlog.Panic("GeoIp db file: %v error: %v", filename, err)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		xlog.Panic("GeoIp db file: %v error: %v", filename, err)
	}

	xlog.Info("GeoIp db file: %v", filename)

	db, err := geoip2.Open(filename)
	if err != nil {
		xlog.Panic("GeoIp db file: %v error: %v", filename, err)
	}

	// defer db.Close()

	return func(next echo.HandlerFunc) echo.HandlerFunc {

		return func(c echo.Context) error {
			countryCode := ""
			ipStr := c.RealIP()
			ip := net.ParseIP(ipStr)
			country, err := db.Country(ip)
			if err != nil {
				xlog.Debug("Ip to country ip: %v error: %v", ipStr, err)
			}

			if country != nil {
				countryCode = country.Country.IsoCode
			}

			countryCode = strings.ToLower(countryCode)
			// c.Request().Header.Del("X-Country-Code")
			c.Request().Header.Set("X-Country-Code", countryCode) // map[string][]string

			return next(c)
		}

	}

}
