package cmd

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"go-proxy/internal/config"
	"go-proxy/internal/middleware"
	"go-proxy/internal/service"

	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	xlog "go-proxy/internal/util/utillog"

	"go-proxy/internal/router"

	"github.com/labstack/echo/v4"
	elog "github.com/labstack/gommon/log"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type Command struct {
	AppService service.AppService
	WebDriver  *echo.Echo

	stop context.CancelFunc
}

func (x *Command) Stop() {

	x.stop()
}

func (x *Command) Exec() {

	defer xlog.Sync()

	x.AppService = service.MustNewAppServiceProd()

	x.WebDriver = echo.New()
	x.WebDriver.Logger.SetLevel(elog.INFO) // has "file":"cmd.go","line":"85"
	//

	//
	middleware.Init(x.WebDriver, x.AppService) // 1
	router.Init(x.WebDriver, x.AppService)     // 2

	defer func() {

		xlog.Info("bye")
	}()

	x.startWithGracefulShutdown()

	time.Sleep(400 * time.Microsecond)
}

func applyServerTLS(s *http.Server, c *config.AppConfig) {

	sessionCache := c.HTTPServer.TLSSessionCache
	sessionTickets := c.HTTPServer.TLSSessionTickets

	cfg := s.TLSConfig
	//
	if sessionCache {
		sessionCacheSize := c.HTTPServer.TLSSessionCacheSize
		if sessionCacheSize <= 0 {
			sessionCacheSize = 128
		}
		cfg.ClientSessionCache = tls.NewLRUClientSessionCache(sessionCacheSize)

		xlog.Info("enabled TLS session cache: size: %v", sessionCacheSize)
	}

	if sessionTickets {
		//
		cfg.SessionTicketsDisabled = false
		go func() {
			ticker := time.NewTicker(24 * time.Hour) // Rotate every 24 hours
			defer ticker.Stop()
			for range ticker.C {
				var newKey [32]byte
				_, err := rand.Read(newKey[:])
				if err != nil {
					xlog.Error("failed to generate session ticket key: %v", err)
				} else {
					cfg.SetSessionTicketKeys([][32]byte{newKey}) // Replace the keys
				}
			}
		}()

		xlog.Info("enabled TLS session tickets")
	}

	s.TLSConfig = cfg

}
func applyServer(s *http.Server, c *config.AppConfig) {

	s.ReadTimeout = time.Duration(c.HTTPServer.ReadTimeout) * time.Second
	s.WriteTimeout = time.Duration(c.HTTPServer.WriteTimeout) * time.Second
	s.IdleTimeout = time.Duration(c.HTTPServer.IdleTimeout) * time.Second
	s.ReadHeaderTimeout = time.Duration(c.HTTPServer.ReadHeaderTimeout) * time.Second

}

func (x *Command) startWithGracefulShutdown() {

	appConfig := x.AppService.Config()

	// Graceful shutdown

	webDriver := x.WebDriver

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()
	x.stop = stop

	// Start server

	{

		applyServer(webDriver.Server, appConfig)
		applyServer(webDriver.TLSServer, appConfig)

		serve := func(listen string) {
			xlog.Info("server starting: %v", listen)

			defer func() {
				xlog.Info("server exiting")
				if r := recover(); r != nil {
					// Log or handle the panic
					panic(fmt.Errorf("error panic: %v", r))
				}
			}()

			if err := webDriver.Start(listen); err != nil {
				if err != http.ErrServerClosed {
					xlog.Error("%v", err)
				} else {
					xlog.Info("shutting down the server")
				}
			}

		}

		serveTLS := func(listen string, certDir string,
			certHosts []string,
		) {

			xlog.Info("server starting: %v, cert from: %v", listen, certDir)

			if certDir == "" {
				xlog.Panic("certificate dir not defined")
				return
			}

			if len(certHosts) == 0 {
				xlog.Panic("certificate host not defined")
				return
			}

			certDir, _ = filepath.Abs(certDir)

			crt, _ := filepath.Abs(filepath.Join(certDir, certHosts[0]))
			key, _ := filepath.Abs(filepath.Join(certDir, certHosts[0]))

			for _, itm := range []string{certDir, crt, key} {
				if _, err := os.Stat(itm); os.IsNotExist(err) {
					xlog.Panic("path not exists : %v error: %v", itm, err)
				}

				xlog.Info("cert path: %v", itm) // 1 2 3
			}

			if err := webDriver.StartTLS(listen, crt, key); err != nil {
				if err != http.ErrServerClosed {
					xlog.Error("%v", err)
				} else {
					xlog.Info("shutting down the server")
				}
			}

		}
		serveAutoTLS := func(listen string,
			certDir string,
			certHosts []string,
			debug bool,
		) {

			xlog.Info("server hosts: %v", certHosts)

			xlog.Info("server starting with auto TLS: %v, cert from: %v", listen, certDir)

			if certDir == "" {
				xlog.Panic("certificate dir not defined")
				return
			}

			if len(certHosts) == 0 {
				xlog.Panic("certificate host not defined")
				return
			}

			certDir, _ = filepath.Abs(certDir)

			for _, itm := range []string{certDir} {
				if _, err := os.Stat(itm); os.IsNotExist(err) {
					xlog.Panic("path not exists : %v error: %v", itm, err)
				}

				xlog.Info("cert path: %v", itm)
			}

			// .Email
			webDriver.AutoTLSManager.Prompt = autocert.AcceptTOS
			webDriver.AutoTLSManager.HostPolicy = autocert.HostWhitelist(certHosts...) // Add your domain(s) here
			webDriver.AutoTLSManager.Cache = autocert.DirCache(certDir)                // Directory for storing certificates

			if debug {
				webDriver.AutoTLSManager.Client = &acme.Client{
					DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
				}
			}

			if err := webDriver.StartAutoTLS(listen); err != nil {
				if err != http.ErrServerClosed {
					xlog.Error("%v", err)
				} else {
					xlog.Info("shutting down the server")
				}
			}

		}

		if appConfig.HTTPServer.Listen != "" {
			go serve(appConfig.HTTPServer.Listen)
		}

		if appConfig.HTTPServer.ListenTLS != "" {

			applyServerTLS(webDriver.TLSServer, appConfig)

			if appConfig.HTTPServer.AutoTLS {
				go serveAutoTLS(appConfig.HTTPServer.ListenTLS,
					appConfig.HTTPServer.CertDir,
					appConfig.HTTPServer.CertHosts,
					appConfig.Debug,
				)
			} else {
				go serveTLS(appConfig.HTTPServer.ListenTLS,
					appConfig.HTTPServer.CertDir,
					appConfig.HTTPServer.CertHosts,
				)
			}

		}

	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctx.Done()
	xlog.Info("interrupt signal")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	xlog.Info("shutdown web driver")
	if err := webDriver.Shutdown(ctx); err != nil {
		xlog.Error("error on shutdown server: %v", err)
	}
}
