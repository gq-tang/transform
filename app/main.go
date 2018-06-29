package main

import (
	"flag"
	"syscall"

	"os"
	"os/signal"

	"github.com/gq-tang/transform/setting"
	"github.com/gq-tang/transform/transmit"
	log "github.com/sirupsen/logrus"
)

func main() {
	p := transmit.NewUDPPool()
	p.Server()

	// 安全退出，退出前关闭相关资源
	sigChan := make(chan os.Signal)
	exitChan := make(chan struct{})
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("Signal", <-sigChan).Info("signal received.")
	go func() {
		log.Warning("stopping server")
		p.Close()
		exitChan <- struct{}{}
	}()
	select {
	case <-exitChan:
	case s := <-sigChan:
		log.WithField("signal", s).Info("signal received. stopping immediately.")
	}
}

// 加载配置文件，初始化相关信息
func init() {
	var cfg_file string
	flag.StringVar(&cfg_file, "file", "", "Use -file <configuration file>")
	if cfg_file != "" {
		setting.LoadConfig(cfg_file)
	} else {
		setting.LoadConfig("./config/app.toml", "../config/app.toml", "/etc/transform/app.toml")
	}

	log.SetLevel(log.Level(setting.Cfg.General.LogLevel))
}
