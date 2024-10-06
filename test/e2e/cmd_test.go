package e2e

import (
	"context"
	xcmd "go-proxy/internal/cmd"
	"go-proxy/internal/config"
	"go-proxy/internal/tool/toolhttp"
	xlog "go-proxy/internal/tool/toollog"
	"go-proxy/internal/tool/tooltest"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// TestHealthController_Check_Stats tests the ?cmd=stats case in the Check method
func TestCmd(t *testing.T) {
	// Setup Echo context
	// github: listen tcp 127.0.0.1:80: bind: permission denied "sudo go test"

	config.CmdLine.Listen = "localhost:10080"
	config.CmdLine.ListenTLS = "localhost:10443"

	cwd, _ := os.Getwd() // ..go-proxy/test/e2e"
	projectRoot, _ := tooltest.GetProjectRoot()

	t.Logf("Cwd: %v", cwd)
	t.Logf("Project root: %v", projectRoot)

	config.CmdLine.CertDir = filepath.Join(projectRoot, "configs/cert")

	config.CmdLine.Targets = append(config.CmdLine.Targets,
		"http://localhost:10081/test/*",
	)
	config.CmdLine.CertHosts = append(config.CmdLine.CertHosts,
		"localhost",
	)

	eSrv := echo.New()
	eSrv.GET("/test/test12345", func(c echo.Context) error {
		return c.String(http.StatusOK, c.QueryParam("msg"))
	})
	go func() {
		eSrv.Start("localhost:10081")
	}()

	cmd := xcmd.Command{}

	go cmd.Exec()

	time.Sleep(3 * time.Second)

	urls := []struct {
		title  string
		url    string
		search string
	}{
		{title: "test proxy", search: "1234567890", url: "http://localhost:10080/test/test12345?msg=1234567890"},
	}

	for _, itm := range urls {

		t.Run(itm.title, func(t *testing.T) {

			t.Logf("url %v", itm.url)
			respData, err := toolhttp.GetBytes(itm.url, nil, nil)

			if err != nil {
				t.Errorf("Error : %v", err)
			}

			if !strings.Contains(string(respData), itm.search) {
				t.Errorf("Error on %v", itm.url)
			} else {

				xlog.Info("Test ok: %v", itm.title)
				t.Logf("Test ok: %v", itm.title)
			}

		})

	}

	cmd.Stop()

	eSrv.Shutdown(context.TODO())

	time.Sleep(1 * time.Second)

}
