package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"os"
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

type ProxiesConfigure struct {
	Admin struct {
		Addr string
	}
	Proxies []struct {
		Name     string
		Listen   string
		Backends []string
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

func startProxies(c *cli.Context) error {

	config, err := loadConfig(c.String("config"))

	if err != nil {
		return err
	}
	proxyMgr := NewProxyMgr()
	admin := NewAdmin(config.Admin.Addr, proxyMgr)
	for _, proxy := range config.Proxies {
		roundRobin := NewRoundrobin()
		for _, backend := range proxy.Backends {
			roundRobin.AddBackend(backend)
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
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Load configuration from `FILE`",
			},
		},
		Action: startProxies,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Fail to start application")
	}
}
