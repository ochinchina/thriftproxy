package main

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"runtime"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)

	if runtime.GOOS == "windows" {
		log.SetFormatter(&log.TextFormatter{DisableColors: true, FullTimestamp: true})
	} else {
		log.SetFormatter(&log.TextFormatter{DisableColors: false, FullTimestamp: true})
	}

	log.SetLevel(log.DebugLevel)
}

func initLog(logFile string, strLevel string, logSize int, backups int) {
	level, err := log.ParseLevel(strLevel)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
	if len(logFile) <= 0 {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(&lumberjack.Logger{Filename: logFile,
			MaxSize:    logSize,
			MaxBackups: backups})
	}
}

type ReadinessConf struct {
	Protocol string
	Port     int
	Path     string `yaml:"path,omitempty"`
}
type CircuitbreakConf struct {
	SuccessiveFailures int    `yaml:"successiveFailures"`
	PauseTime          string `yaml:"pauseTime"`
}
type BackendInfo struct {
	Addr           string
	Readiness      *ReadinessConf    `yaml:"readiness,omitempty"`
	CircuitBreaker *CircuitbreakConf `yaml:"circuitBreaker,omitempty"`
}

type ProxiesConfigure struct {
	Admin struct {
		Addr string
	}
	Metrics struct {
		Addr string
	}
	Proxies []struct {
		Name           string
		Listen         string
		RequestTimeout string `yaml:"requestTimeout,omitempty"`
		Backends       []BackendInfo
	}
}

func loadConfig(fileName string) (*ProxiesConfigure, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	r := &ProxiesConfigure{}

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(r)

	if err != nil {
		return nil, err
	}
	return r, nil

}

func startMetrics(addr string) {
	var server http.Server
	server.Addr = addr
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	server.Handler = router
	go server.ListenAndServe()

}
func startProxies(c *cli.Context) error {

	config, err := loadConfig(c.String("config"))

	if err != nil {
		return err
	}
	strLevel := c.String("log-level")
	fileName := c.String("log-file")
	logSize := c.Int("log-size")
	backups := c.Int("log-backups")
	initLog(fileName, strLevel, logSize, backups)
	proxyMgr := NewProxyMgr()
	admin := NewAdmin(config.Admin.Addr, proxyMgr)
	defTimeout := time.Duration(60) * time.Second
	for _, proxy := range config.Proxies {
		roundRobin := NewRoundrobin()
		for _, backend := range proxy.Backends {
			roundRobin.AddBackend(&backend)
		}
		proxyMgr.AddProxy(NewProxy(proxy.Name, proxy.Listen, convertDuration(proxy.RequestTimeout, defTimeout), roundRobin))
	}

	admin.Start()
	startMetrics(config.Metrics.Addr)

	proxyMgr.Run()

	return nil
}

func main() {
	app := &cli.App{
		Name:  "thriftproxy",
		Usage: "a proxy between thrift client and thrift backend servers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Aliases:  []string{"c"},
				Required: true,
				Usage:    "Load configuration from `FILE`",
			},
			&cli.StringFlag{
				Name:  "log-file",
				Usage: "log file name",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "one of following level: Trace, Debug, Info, Warn, Error, Fatal, Panic",
			},
			&cli.IntFlag{
				Name:  "log-size",
				Usage: "size of log file in Megabytes",
				Value: 50,
			},
			&cli.IntFlag{
				Name:  "log-backups",
				Usage: "number of log rotate files",
				Value: 10,
			},
		},
		Action: startProxies,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Fail to start application")
	}
}
