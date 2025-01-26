package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"go-proxy/internal/config/consts"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	xlog "go-proxy/internal/util/utillog"

	"go-proxy/internal/util/utilconfig"
)

var (
	AppVersion  = ""
	AppCommit   = ""
	AppDate     = ""
	ShortCommit = ""
)

func dumpVersionAndExitIf() {

	if CmdLine.Version {
		fmt.Printf("version: %s\n", AppVersion)
		fmt.Printf("commit: %s\n", AppCommit)
		fmt.Printf("date: %s\n", AppDate)
		//
		os.Exit(0)
	}

}

type CmdLineConfig struct {
	Config    string
	CertDir   string
	Env       string
	Name      string
	IsMaint   bool
	Version   bool
	Upstreams []string // ["",""]
	CertHosts []string // ["",""]
	GeoIPFile string

	SysAPIKey string
	Listen    string
	ListenTLS string
	ListenSys string

	DumpConfig bool
}

const (
	envDevelopment = "development"
	envTesting     = "testing"
	envStaging     = "staging"
	envProduction  = "production"
)

var envNames = []string{
	envDevelopment, envTesting, envStaging, envProduction,
}

var CmdLine = CmdLineConfig{}

// ReadFlags read app flags
func ReadFlags() {

	_ = os.Args

	flag.StringVar(&CmdLine.Config, "config", "", "path to dir with config files")
	flag.StringVar(&CmdLine.CertDir, "cert-dir", "", "path to dir with cert files")
	flag.StringVar(&CmdLine.SysAPIKey, "sys-api-key", "", "sys api key")
	flag.StringVar(&CmdLine.Listen, "listen", "", "listen")
	flag.StringVar(&CmdLine.ListenTLS, "listen-tls", "", "listen TLS")
	flag.StringVar(&CmdLine.ListenSys, "listen-sys", "", "listen sys")
	flag.StringVar(&CmdLine.GeoIPFile, "geo-ip-file", "", "Path to file GeoLite2-Country.mmdb")

	flag.BoolVar(&CmdLine.IsMaint, "is-maint", false, "Maintenance mode")
	flag.StringVar(&CmdLine.Env, "env", "", "environment: development, testing, staging, production")
	flag.StringVar(&CmdLine.Name, "name", "", "app name")

	flag.Func("upstream", "Proxy upstream URL", func(value string) error {
		CmdLine.Upstreams = append(CmdLine.Upstreams, value)
		return nil
	})

	flag.Func("upstreams", "Proxy upstreams URL as json array", func(value string) (err error) {
		if value != "" {
			tmp := []string{}
			if err = json.Unmarshal([]byte(value), &tmp); err != nil {
				CmdLine.Upstreams = append(CmdLine.Upstreams, tmp...)
			}
		}
		return err
	})

	flag.Func("cert_host", "Define host for TLS", func(value string) error {
		CmdLine.CertHosts = append(CmdLine.CertHosts, value)
		return nil
	})

	flag.Func("cert_hosts", "Define hosts for TLS as json array", func(value string) (err error) {
		if value != "" {
			tmp := []string{}
			if err = json.Unmarshal([]byte(value), &tmp); err != nil {
				CmdLine.CertHosts = append(CmdLine.CertHosts, tmp...)
			}
		}
		return err
	})

	flag.BoolVar(&CmdLine.Version, "version", false, "app version")

	flag.BoolVar(&CmdLine.DumpConfig, "dump-config", false, "dump config")

	flag.Parse() // dont use from init()

	dumpVersionAndExitIf()
}

type envReader struct {
	envError error
	prefix   string
}

func NewEnvReader() envReader {
	return envReader{prefix: "app_"}
}
func (x *envReader) readEnv(name string) string {
	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	{
		// APP_TITLE
		if envName != "" {
			envValue := os.Getenv(envName)
			if envValue != "" {
				xlog.Info("reading %q value from env: %v = %v", name, envName, envValue)
				return envValue
			}
		}
	}

	{
		// APP_TITLE_FILE
		envNameFile := strings.ToUpper(envName + "_file") //
		filePath := os.Getenv(envNameFile)
		if filePath != "" { // file path
			filePath = filepath.Clean(filePath)
			xlog.Info("reading %q value from file: %v = %v", name, envNameFile, filePath)
			if data, err := os.ReadFile(filePath); err == nil {
				return string(data)
			} else {
				x.envError = err
			}
		}
	}

	return ""
}

func (x *envReader) String(p *string, name string, cmdValue *string) {

	// from cmd
	if cmdValue != nil && *cmdValue != "" {
		xlog.Info("reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}

	// from env
	{
		envValue := x.readEnv(name)
		if envValue != "" {
			*p = envValue
		}
	}

}

func (x *envReader) StringArray(p *[]string, name string, cmdValue *[]string) {
	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive
	if cmdValue != nil && len(*cmdValue) > 0 {
		xlog.Info("reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue // error: p = cmdValue

		return
	}

	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("reading %q value from env: %v = %v", name, envName, envValue)
			tmp := []string{}
			if err := json.Unmarshal([]byte(envValue), &tmp); err != nil {
				x.envError = err
			}
			if len(tmp) > 0 {
				*p = tmp // error: p = &tmp
			}
			return
		}
	}

}

func (x *envReader) Bool(p *bool, name string, cmdValue *bool) {

	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	if cmdValue != nil && *cmdValue {
		xlog.Info("reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}
	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("reading %q value from env: %v = %v", name, envName, envValue)
			*p = envValue == "1" || envValue == "true"
			return
		}
	}
}

func (x *envReader) Float64(p *float64, name string, cmdValue *float64) {

	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	if cmdValue != nil && math.Abs(*cmdValue) > 0.000001 {
		xlog.Info("reading float64 %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}
	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("reading float64 %q value from env: %v = %v", name, envName, envValue)

			if v, err := strconv.ParseFloat(envValue, 64); err == nil {
				*p = v
			} else {
				x.envError = err
			}

		}
	}

}
func (x *envReader) Int(p *int, name string, cmdValue *int) {

	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	if cmdValue != nil && *cmdValue != 0 {
		xlog.Info("reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}
	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("reading %q value from env: %v = %v", name, envName, envValue)

			if v, err := strconv.Atoi(envValue); err == nil {
				*p = v
			} else {
				x.envError = err
			}

		}
	}

}

type Database struct {
	Dialect   string `json:"dialect"`
	Host      string `json:"host"`
	Port      string `json:"port"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
	User      string `json:"user"`
	Password  string `json:"password"`
	MaxOpen   int    `json:"max_open"`
	MaxIdle   int    `json:"max_idle"`
	IdleTime  int    `json:"idle_time"`
	Migration bool   `json:"migration"`
}

// type AppConfigLog struct {
// 	Level int `json:"level"` // 0=Error 1=Warn 2=Info 3=Debug
// }

type AppConfigProxy struct {
	Upstreams      []string       `json:"upstreams"` // records "http://127.0.0.1:8080/api/test/ping"
	OverrideStatus map[int]string `json:"override_status"`
}

type AppConfigHTTPTransport struct {
	MaxIdleConns        int `json:"max_idle_conns,omitempty"`
	MaxIdleConnsPerHost int `json:"max_idle_conns_per_host,omitempty"`
	IdleConnTimeout     int `json:"idle_conn_timeout,omitempty"`
	MaxConnsPerHost     int `json:"max_conns_per_host,omitempty"`
}
type AppConfigHTTPServer struct {
	AccessLog     bool     `json:"access_log"`
	RateLimit     float64  `json:"rate_limit"`
	RateBurst     int      `json:"rate_burst"`
	Listen        string   `json:"listen"`
	ListenTLS     string   `json:"listen_tls"`
	AutoTLS       bool     `json:"auto_tls"`
	CertHosts     []string `json:"cert_hosts"`
	RedirectHTTPS bool     `json:"redirect_https"`
	RedirectWWW   bool     `json:"redirect_www"`
	RequestID     bool     `json:"request_id"`

	CertDir           string `json:"cert_dir"`
	RequestTimeout    int    `json:"request_timeout,omitempty"`     // 5 to 30 seconds
	ReadTimeout       int    `json:"read_timeout,omitempty"`        // 5 to 30 seconds
	WriteTimeout      int    `json:"write_timeout,omitempty"`       // 10 to 30 seconds, WriteTimeout > ReadTimeout
	IdleTimeout       int    `json:"idle_timeout,omitempty"`        // 60 to 120 seconds
	ReadHeaderTimeout int    `json:"read_header_timeout,omitempty"` // default get from ReadTimeout

	SysMetrics bool   `json:"sys_metrics"` //
	SysAPIKey  string `json:"sys_api_key"`
	ListenSys  string `json:"listen_sys"`

	AllowOrigins []string `json:"allow_origins"`
	HeadersDel   []string `json:"headers_del"`
	HeadersAdd   []string `json:"headers_add"`
	// default-src 'self' 'unsafe-inline' 'unsafe-eval' data: *.openstreetmap.org *.googleapis.com *.gstatic.com *.youtube.com
	ContentPolicy string `json:"content_policy"`

	BodyLimit string `json:"body_limit"` // 2M 2000K 1G

	TLSSessionCacheSize int  `json:"tls_session_cache_size"` //
	TLSSessionCache     bool `json:"tls_session_cache"`      //
	TLSSessionTickets   bool `json:"tls_session_tickets"`    //

	CSRF bool `json:"csrf"` //
}

type AppConfigMod struct {
	Name    string `json:"-"`
	Env     string `json:"env"` // prod||'' dev stage
	Debug   bool   `json:"-"`
	IsMaint bool   `json:"is_maint"`
	Title   string `json:"title"`

	ConfigPath []string `json:"-"` // []string{".", os.Getenv("APP_CONFIG"), flagAppConfig}
}

type AppConfigGeoIP struct {
	File         string   `json:"file"`
	Enabled      bool     `json:"enabled"`
	AllowCountry []string `json:"allow_country"`
	BlockCountry []string `json:"block_country"`
}

type AppConfig struct {
	AppConfigMod

	// Log AppConfigLog `json:"logger"`

	DB    Database `json:"database"`
	Redis Database `json:"redis"`

	Proxy AppConfigProxy `json:"proxy"`

	HTTPTransport AppConfigHTTPTransport `json:"http_transport"`

	HTTPServer AppConfigHTTPServer `json:"http_server"`

	GeoIP AppConfigGeoIP `json:"geo_ip"`
}

func NewAppConfig() *AppConfig {

	res := &AppConfig{

		// Log: AppConfigLog{
		// 	Level: consts.LogLevelWarn,
		// },

		DB: Database{
			Dialect:  "postgres",
			Host:     "localhost",
			Port:     "5432",
			Name:     "postgres",
			User:     "postgres",
			Password: "postgres",
			MaxOpen:  0,
			MaxIdle:  0,
			IdleTime: 0,
		},

		Redis: Database{
			Host:     "localhost",
			Port:     "6379",
			Name:     "redis",
			User:     "redis",
			Password: "redis",
		},

		AppConfigMod: AppConfigMod{
			Name:       consts.AppName,
			ConfigPath: []string{"."},
			Title:      "",
			Env:        envProduction,
			Debug:      false,
			IsMaint:    false,
		},

		Proxy: AppConfigProxy{},

		HTTPTransport: AppConfigHTTPTransport{},
		HTTPServer: AppConfigHTTPServer{
			RequestTimeout: 20,
			ReadTimeout:    5,
			WriteTimeout:   10,
			IdleTimeout:    30,

			RateLimit: 5,
			RateBurst: 10,

			Listen:    "127.0.0.1:80",
			ListenTLS: "127.0.0.1:443",
			CertDir:   "",

			SysAPIKey: "",

			CertHosts: []string{},

			BodyLimit: "2M", // 2M 2000K 1G

			TLSSessionCache:   false,
			TLSSessionTickets: false,

			CSRF: true,
		},
	}

	return res
}

func (x *AppConfig) readEnvName() error {
	reader := NewEnvReader()
	// APP_ENV -env
	reader.String(&x.Env, "env", &CmdLine.Env)
	reader.String(&x.Name, "name", &CmdLine.Name)

	if err := x.validateEnv(); err != nil {
		return err
	}

	configPath := slices.Concat(strings.Split(os.Getenv("APP_CONFIG"), ";"), strings.Split(CmdLine.Config, ";"))
	configPath = slices.Compact(configPath)
	configPath = slices.DeleteFunc(
		configPath,
		func(x string) bool {
			return x == ""
		},
	)

	for i := 0; i < len(configPath); i++ {
		configPath[i] += "/" + x.Name
	}

	// if len(configPath) == 0 {
	// 	configPath = []string{"."} // default
	// }

	if len(configPath) == 0 {
		xlog.Warn("config path is empty")
	} else {
		xlog.Info("config path: %v", configPath)
	}

	x.ConfigPath = configPath

	return nil
}
func (x *AppConfig) readEnvVar() error {
	reader := NewEnvReader()

	reader.Int(&x.DB.MaxOpen, "db_max_open", nil)
	reader.Int(&x.DB.MaxIdle, "db_max_idle", nil)
	reader.Int(&x.DB.IdleTime, "db_idle_time", nil)

	reader.String(&x.Env, "env", nil)
	reader.String(&x.Title, "title", nil)

	// Http server
	reader.Bool(&x.HTTPServer.AccessLog, "http_access_log", nil)
	reader.Float64(&x.HTTPServer.RateLimit, "http_rate_limit", nil)
	reader.Int(&x.HTTPServer.RateBurst, "http_rate_burst", nil)
	reader.String(&x.HTTPServer.Listen, "http_listen", nil)        // =>listen
	reader.String(&x.HTTPServer.ListenTLS, "http_listen_tls", nil) // =>listen_tls
	reader.Bool(&x.HTTPServer.AutoTLS, "http_auto_tls", nil)
	reader.Bool(&x.HTTPServer.RedirectHTTPS, "http_redirect_https", nil)
	reader.Bool(&x.HTTPServer.RedirectWWW, "http_redirect_www", nil)
	reader.String(&x.HTTPServer.CertDir, "http_cert_dir", nil) // =>cert_dir
	reader.Int(&x.HTTPServer.ReadTimeout, "http_read_timeout", nil)
	reader.Int(&x.HTTPServer.WriteTimeout, "http_write_timeout", nil)
	reader.Int(&x.HTTPServer.IdleTimeout, "http_idle_timeout", nil)
	reader.Int(&x.HTTPServer.ReadHeaderTimeout, "http_read_header_timeout", nil)
	reader.String(&x.HTTPServer.ListenSys, "http_listen_sys", nil)  // =>listen_sys
	reader.String(&x.HTTPServer.SysAPIKey, "http_sys_api_key", nil) // =>sys_api_key
	reader.StringArray(&x.HTTPServer.AllowOrigins, "http_allow_origins", nil)
	reader.StringArray(&x.HTTPServer.HeadersDel, "http_headers_del", nil)
	reader.StringArray(&x.HTTPServer.HeadersAdd, "http_headers_add", nil)
	reader.String(&x.HTTPServer.ContentPolicy, "http_content_policy", nil)
	reader.String(&x.HTTPServer.BodyLimit, "http_body_limit", nil) // =>body_limit
	reader.Int(&x.HTTPServer.RequestTimeout, "http_request_timeout", nil)

	reader.String(&x.HTTPServer.CertDir, "cert_dir", &CmdLine.CertDir)

	reader.String(&x.GeoIP.File, "geo_ip_file", &CmdLine.GeoIPFile)

	reader.Bool(&x.IsMaint, "is_maint", &CmdLine.IsMaint)

	reader.String(&x.HTTPServer.Listen, "listen", &CmdLine.Listen)
	reader.String(&x.HTTPServer.ListenTLS, "listen_tls", &CmdLine.ListenTLS)
	reader.String(&x.HTTPServer.ListenSys, "listen_sys", &CmdLine.ListenSys)

	reader.String(&x.HTTPServer.SysAPIKey, "sys_api_key", &CmdLine.SysAPIKey)

	reader.StringArray(&x.Proxy.Upstreams, "tragets", &CmdLine.Upstreams)
	reader.StringArray(&x.HTTPServer.CertHosts, "cert_hosts", &CmdLine.CertHosts)

	reader.StringArray(&x.GeoIP.AllowCountry, "allow_country", nil)
	reader.StringArray(&x.GeoIP.BlockCountry, "block_country", nil)

	reader.StringArray(&x.HTTPServer.AllowOrigins, "allow_origins", nil)
	reader.StringArray(&x.HTTPServer.HeadersDel, "headers_del", nil)
	reader.StringArray(&x.HTTPServer.HeadersAdd, "headers_add", nil)
	reader.String(&x.HTTPServer.ContentPolicy, "content_policy", nil)

	reader.String(&x.HTTPServer.BodyLimit, "body_limit", nil)
	reader.Int(&x.HTTPServer.RequestTimeout, "request_timeout", nil)

	reader.Int(&x.HTTPServer.TLSSessionCacheSize, "http_tls_session_cache_size", nil)
	reader.Bool(&x.HTTPServer.TLSSessionCache, "http_tls_session_cache", nil)
	reader.Bool(&x.HTTPServer.TLSSessionTickets, "http_tls_session_tickets", nil)

	if reader.envError != nil {
		return reader.envError
	}

	return nil
}

func (x *AppConfig) validateEnv() error {

	if x.Env == "" {
		x.Env = envProduction
	}
	x.Debug = x.Env == envDevelopment
	if !slices.Contains(envNames, x.Env) {
		xlog.Warn("non-standart env name: %v", x.Env)
	}

	return nil

}
func (x AppConfig) validate() error {

	if x.HTTPServer.Listen == "" && x.HTTPServer.ListenTLS == "" {
		return fmt.Errorf("socket Listen and ListenTLS are empty")
	}

	return nil
}

type AppConfigSource struct {
	config *AppConfig
}

func MustNewAppConfigSource() *AppConfigSource {

	res := &AppConfigSource{}

	err := res.Load() // init

	if err != nil {
		panic(err)
	}

	return res

}

func (x *AppConfigSource) Load() error {

	res := NewAppConfig()

	{
		cwd, _ := os.Getwd()
		xlog.Info("current work dir: %v", cwd)
	}
	{
		err := res.readEnvName()
		if err != nil {
			return err
		}
	}

	{
		for i := 0; i < len(res.ConfigPath); i++ {

			dir := res.ConfigPath[i]
			fileName := fmt.Sprintf("config.%s.json", res.Env)

			xlog.Info("loading config from: %v", dir)

			err := utilconfig.LoadConfig(res /*pointer*/, dir, fileName)

			if err != nil {
				return err
			}

		}

	}

	{
		err := res.readEnvVar()
		if err != nil {
			return err
		}

	}

	{
		err := res.validate()
		if err != nil {
			return err
		}
	}

	xlog.Info("config loaded: Name=%v Env=%v Debug=%v ", res.Name, res.Env, res.Debug)

	x.config = res

	if CmdLine.DumpConfig {
		data, _ := json.MarshalIndent(res, "", " ")
		fmt.Println(string(data))
	}

	return nil
}

func (x *AppConfigSource) Config() *AppConfig {

	return x.config

}

// FromJSON from json
func (x *AppConfig) FromJSON(data string) error {

	if data == "" {
		return nil
	}

	err := json.Unmarshal([]byte(data), x)

	if err != nil {
		return err
	}

	return nil
}
