package service

import (
	"go-proxy/internal/config"
	"os"

	"time"

	xlog "go-proxy/internal/util/utillog"
	"net/http"
)

// AppService all services
type AppService interface {
	Config() *config.AppConfig
	// Logger() logger.AppLogger
}
type defaultAppService struct {
	configSource *config.AppConfigSource
}

func mustConfigRuntime(appConfig *config.AppConfig) {
	t, ok := http.DefaultTransport.(*http.Transport)

	if ok {
		x := appConfig.HTTPTransport

		if x.MaxIdleConns > 0 {
			xlog.Info("set Http.Transport.MaxIdleConns=%v", x.MaxIdleConns)
			t.MaxIdleConns = x.MaxIdleConns
		}
		if x.IdleConnTimeout > 0 {
			xlog.Info("set Http.Transport.IdleConnTimeout=%v", x.IdleConnTimeout)
			t.IdleConnTimeout = time.Duration(x.IdleConnTimeout) * time.Second
		}
		if x.MaxConnsPerHost > 0 {
			xlog.Info("set Http.Transport.MaxConnsPerHost=%v", x.MaxConnsPerHost)
			t.MaxConnsPerHost = x.MaxConnsPerHost
		}

		if x.MaxIdleConnsPerHost > 0 {
			xlog.Info("set Http.Transport.MaxIdleConnsPerHost=%v", x.MaxIdleConnsPerHost)
			t.MaxIdleConnsPerHost = x.MaxIdleConnsPerHost
		}

	} else {
		xlog.Error("cannot init http.Transport")
	}
}

func (x *defaultAppService) mustConfig() {

	d, _ := os.Getwd()

	xlog.Info("current work dir: %v", d)

	x.configSource = config.MustNewAppConfigSource()

	appConfig := x.Config() // first call, init

	mustConfigRuntime(appConfig)

}

func (x *defaultAppService) mustBuild() {

}

// MustNewAppServiceProd
func MustNewAppServiceProd() AppService {

	appService := &defaultAppService{}

	appService.mustConfig()
	appService.mustBuild()

	return appService

}
func MustNewAppServiceProdNewAppServiceTesting() AppService {
	return MustNewAppServiceProd()
}

func (x *defaultAppService) Config() *config.AppConfig { return x.configSource.Config() }
