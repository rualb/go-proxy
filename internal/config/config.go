package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"go-proxy/internal/config/consts"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	xlog "go-proxy/internal/tool/toollog"

	"go-proxy/internal/tool/toolconfig"
	"go-proxy/internal/tool/toolhttp"
)

var (
	AppVersion  = ""
	AppCommit   = ""
	AppDate     = ""
	ShortCommit = ""
)

func dumpVersionAndExitIf() {

	if CmdLine.Version {
		fmt.Printf("Version: %s\n", AppVersion)
		fmt.Printf("Commit: %s\n", AppCommit)
		fmt.Printf("Date: %s\n", AppDate)
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
	Targets   []string
	CertHosts []string
	GeoIPFile string
	Listen    string
	ListenTLS string
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
	flag.StringVar(&CmdLine.Config, "config", "", "Path to dir with config files")
	flag.StringVar(&CmdLine.CertDir, "cert-dir", "", "Path to dir with cert files")
	flag.StringVar(&CmdLine.GeoIPFile, "geo-ip-file", "", "Path to file GeoLite2-Country.mmdb")

	flag.BoolVar(&CmdLine.IsMaint, "is-maint", false, "Maintenance mode")
	flag.StringVar(&CmdLine.Env, "env", "", "Environment: development, testing, staging, production")
	flag.StringVar(&CmdLine.Name, "name", "", "App name")

	flag.Func("target", "Proxy target as URL", func(value string) error {
		CmdLine.Targets = append(CmdLine.Targets, value)
		return nil
	})
	flag.Func("cert_host", "Define host for TLS", func(value string) error {
		CmdLine.CertHosts = append(CmdLine.CertHosts, value)
		return nil
	})

	flag.StringVar(&CmdLine.Listen, "listen", "", "Listen")
	flag.StringVar(&CmdLine.ListenTLS, "listen-tls", "", "Listen TLS")

	flag.BoolVar(&CmdLine.Version, "version", false, "App version")

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

func (x *envReader) String(p *string, name string, cmdValue *string) {
	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive
	if cmdValue != nil && *cmdValue != "" {
		xlog.Info("Reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}

	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("Reading %q value from env: %v = %v", name, envName, envValue)
			*p = envValue
			return
		}
	}

}

func (x *envReader) Bool(p *bool, name string, cmdValue *bool) {

	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	if cmdValue != nil && *cmdValue {
		xlog.Info("Reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}
	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("Reading %q value from env: %v = %v", name, envName, envValue)
			*p = envValue == "1" || envValue == "true"
			return
		}
	}
}

func (x *envReader) Int(p *int, name string, cmdValue *int) {

	envName := strings.ToUpper(x.prefix + name) // *nix case-sensitive

	if cmdValue != nil && *cmdValue != 0 {
		xlog.Info("Reading %q value from cmd: %v", name, *cmdValue)
		*p = *cmdValue
		return
	}
	if envName != "" {
		envValue := os.Getenv(envName)
		if envValue != "" {
			xlog.Info("Reading %q value from env: %v = %v", name, envName, envValue)

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
	Targets        []string       `json:"targets"` // records "http://localhost:8080/api/test/ping"
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

	CertDir string `json:"cert_dir"`

	ReadTimeout       int `json:"read_timeout,omitempty"`        // 5 to 30 seconds
	WriteTimeout      int `json:"write_timeout,omitempty"`       // 10 to 30 seconds, WriteTimeout > ReadTimeout
	IdleTimeout       int `json:"idle_timeout,omitempty"`        // 60 to 120 seconds
	ReadHeaderTimeout int `json:"read_header_timeout,omitempty"` // default get from ReadTimeout
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
	File    string `json:"file"`
	Enabled bool   `json:"enabled"`
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
			ReadTimeout:  5,
			WriteTimeout: 10,
			IdleTimeout:  30,

			RateLimit: 5,
			RateBurst: 10,

			Listen:    "localhost:80",
			ListenTLS: "localhost:443",
			CertDir:   "",
			CertHosts: []string{},
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
		xlog.Warn("Config path is empty")
	} else {
		xlog.Info("Config path: %v", configPath)
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

	reader.String(&x.HTTPServer.Listen, "listen", &CmdLine.Listen)
	reader.String(&x.HTTPServer.ListenTLS, "listen_tls", &CmdLine.ListenTLS)
	reader.String(&x.HTTPServer.CertDir, "cert_dir", &CmdLine.CertDir)

	reader.String(&x.GeoIP.File, "geo_ip_file", &CmdLine.GeoIPFile)

	reader.Bool(&x.IsMaint, "is_maint", &CmdLine.IsMaint)

	x.Proxy.Targets = append(x.Proxy.Targets, CmdLine.Targets...)
	x.HTTPServer.CertHosts = append(x.HTTPServer.CertHosts, CmdLine.CertHosts...)

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
		xlog.Warn("Non-standart env name: %v", x.Env)
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
		xlog.Info("Current work dir: %v", cwd)
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

			xlog.Info("Loading config from: %v", dir)

			err := toolconfig.LoadConfig(res /*pointer*/, dir, fileName)

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

	xlog.Info("Config loaded: Name=%v Env=%v Debug=%v ", res.Name, res.Env, res.Debug)

	x.config = res

	// data, _ := json.MarshalIndent(res, "", "\t")

	// toolfile.WriteBytes("./dump.json", data)

	return nil
}

func (x *AppConfigSource) Config() *AppConfig {

	return x.config

}

// func (x *AppConfig) ApplyConfigFromFilesList(files string, errIfNotExists bool) error {

// 	if files == "" {
// 		return nil
// 	}

// 	for _, x := range strings.Split(files, ";") {
// 		err := x.ApplyConfigFromFile(x, errIfNotExists)

// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// FromFile errIfNotExists argument soft binding, no error if file not exists
func (x *AppConfig) FromFile(dir string, file string) error {

	if file == "" {
		return nil
	}

	if !strings.HasSuffix(file, ".json") && !strings.HasPrefix(file, "config.") {
		return fmt.Errorf("error file not match config.*.json: %v", file)
	}

	fullPath, err := filepath.Abs(filepath.Join(dir, file))

	if err != nil {
		return err
	}

	// fmt.Println("Reading config from file: ", file)
	fullPath = filepath.Clean(fullPath)
	data, err := os.ReadFile(fullPath)

	// , errIfNotExists bool
	// if os.IsNotExist(err) && !errIfNotExists {

	// 	xlog.Info("File not exists: %v", fullPath)

	// 	return nil
	// }

	if err != nil {
		return fmt.Errorf("error with file %v: %v", fullPath, err)
	}

	xlog.Info("Loading config from file: %v", fullPath)

	err = x.FromJSON(string(data))
	if err != nil {
		return err
	}

	return nil
}

// FromURL errIfNotExists argument soft binding, no error if file not exists
func (x *AppConfig) FromURL(dir string, file string) error {

	if file == "" {
		return nil
	}

	if !strings.HasSuffix(file, ".json") && !strings.HasPrefix(file, "config.") {
		return fmt.Errorf("error file not match config.*.json: %v", file)
	}

	fullPath := dir + "/" + file

	_, err := url.Parse(fullPath)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// fmt.Println("Reading config from file: ", file)

	data, err := toolhttp.GetBytes(fullPath, nil, nil)

	if err != nil {
		return fmt.Errorf("error with file %v: %v", fullPath, err)
	}

	xlog.Info("Loading config from file: %v", fullPath)

	err = x.FromJSON(string(data))
	if err != nil {
		return err
	}

	return nil
}

// func (x *AppConfig) ApplyConfigFromEnv(env string) {
// 	x.ApplyConfigFromFile(fmt.Sprintf("config.%s.json", env))
// }

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
