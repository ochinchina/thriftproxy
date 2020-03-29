package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
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

type BackendInfo struct {
	Addr      string
	Readiness *ReadinessConf `yaml:"readiness,omitempty"`
}

type ProxiesConfigure struct {
	Admin struct {
		Addr string
	}
	Proxies []struct {
		Name     string
		Listen   string
		Backends []BackendInfo
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

func createReadiness(addr string, readinessConf *ReadinessConf) Readiness {
	if readinessConf == nil {
		return NewNullReadiness()
	}
	ip, _, err := splitAddr(addr)
	if err != nil {
		ip = addr
	}
	//if it is IPv6
	if strings.Index(ip, ":") != -1 && !strings.HasPrefix(ip, "[") {
		ip = fmt.Sprintf("[%s]", ip)
	}
	switch readinessConf.Protocol {
	case "tcp":
		return NewTcpReadiness(fmt.Sprintf("%s:%d", ip, readinessConf.Port))
	case "http":
		path := "/"
		if len(readinessConf.Path) > 0 {
			path = readinessConf.Path
		}
		url := fmt.Sprintf("http://%s:%d%s", ip, readinessConf.Port, path)
		return NewHttpReadiness(url)
	default:
		return NewNullReadiness()
	}
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
	fmt.Printf("config=%v\n", config)
	for _, proxy := range config.Proxies {
		roundRobin := NewRoundrobin()
		for _, backend := range proxy.Backends {
			roundRobin.AddBackend(backend.Addr, backend.Readiness)
		}
		proxyMgr.AddProxy(NewProxy(proxy.Name, proxy.Listen, roundRobin))
	}

	admin.Start()

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
