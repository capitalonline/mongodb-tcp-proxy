package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	mongodb_proxy "mongodb-proxy"
	"mongodb-proxy/conf"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	flag.Parse()
}

func main() {
	if conf.Init() != nil {
		panic("load config error")
		return
	}
	// log
	{
		logPath := conf.Conf.Log.Path
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			logrus.SetOutput(file)
		} else {
			panic(err)
		}
		//logrus.AddHook()
	}
	// monitor
	{
		go http.ListenAndServe("0.0.0.0:7475", nil)
	}
	// Handle SIGTERM to exit cleanly
	{
		signals := make(chan os.Signal)
		signal.Notify(signals, syscall.SIGTERM)
		go func() {
			<-signals
			os.Exit(0)
		}()
	}

	{
		pm := mongodb_proxy.NewProxyManager(conf.Conf)
		pm.Start()
		server := mongodb_proxy.NewRestEndpoint(pm, conf.Conf.App)
		server.Run()
	}

}
