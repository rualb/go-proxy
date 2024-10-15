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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestCmd1(t *testing.T) {
	// Setup Echo context
	// github: listen tcp 127.0.0.1:80: bind: permission denied "sudo go test"

	config.CmdLine.Listen = "127.0.0.1:10080"
	config.CmdLine.ListenTLS = "127.0.0.1:10443"

	cwd, _ := os.Getwd() // ..go-proxy/test/e2e"
	projectRoot, _ := tooltest.GetProjectRoot()

	t.Logf("Cwd: %v", cwd)
	t.Logf("Project root: %v", projectRoot)

	config.CmdLine.CertDir = filepath.Join(projectRoot, "configs/cert")

	config.CmdLine.Upstreams = append(config.CmdLine.Upstreams,
		"http://127.0.0.1:10081/test1",
	)

	config.CmdLine.Upstreams = append(config.CmdLine.Upstreams,
		"http://127.0.0.1:10082/test2?server=127.0.0.1:10083",
	)
	config.CmdLine.CertHosts = append(config.CmdLine.CertHosts,
		"localhost",
	)

	for i := 1; i <= 3; i++ {
		e1 := echo.New()
		pathSuffix := strconv.Itoa(i)
		listen := "127.0.0.1:" + strconv.Itoa(10080+i)
		if i == 3 {
			pathSuffix = "2"
		}
		path := "/test" + pathSuffix
		e1.GET(path, func(c echo.Context) error {
			return c.String(http.StatusOK, "test "+strconv.Itoa(i))
		})
		go func() {
			t.Logf("Temp server on: %v%v", listen, path)

			err := e1.Start(listen)
			if err != nil {
				t.Logf("Error : %v", err)
			}
		}()

		defer e1.Shutdown(context.TODO())
	}

	cmd := xcmd.Command{}

	go cmd.Exec()

	time.Sleep(2 * time.Second)

	urls := []struct {
		title  string
		url    string
		search string
	}{
		{title: "test1", search: "test 1", url: "http://127.0.0.1:10080/test1"},
		{title: "test2", search: "test 2", url: "http://127.0.0.1:10080/test2"},
		{title: "test3 (RoundRobinBalancer)", search: "test 3", url: "http://127.0.0.1:10080/test2"},
	}

	for _, itm := range urls {

		t.Run(itm.title, func(t *testing.T) {

			t.Logf("url %v", itm.url)
			respData, err := toolhttp.GetBytes(itm.url, nil, nil)
			respDataStr := string(respData)
			if err != nil {
				t.Errorf("Error : %v", err)
			}

			if !strings.Contains(respDataStr, itm.search) {
				t.Errorf("Error on %v", itm.url)
			} else {

				xlog.Info("Test ok: %v", itm.title)
				t.Logf("Test ok: %v", itm.title)
			}

		})

	}

	cmd.Stop()

	time.Sleep(1 * time.Second)

}
