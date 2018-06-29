package setting

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	General struct {
		Port     int `toml:"port"`
		LogLevel int `toml:"log_level"`
	} `toml:"general"`
	TcpServer struct {
		Server            string `toml:"server"`
		KeepAlive         bool   `toml:"keep_alive"`
		KeepAlivePeriod   int    `toml:"keep_alive_period"`
		CloseClientPeriod int    `toml:"close_client_period"`
	} `toml:"tcp_server"`
}

var Cfg *Configuration
var once sync.Once

func LoadConfig(files ...string) {
	fn := func() {
		if len(files) == 0 {
			log.Fatal("please provide config file path.")
		}
		Cfg = &Configuration{}
		for _, fileName := range files {
			file, err := os.Open(fileName)
			if err != nil {
				continue
			}
			data, err := ioutil.ReadAll(file)
			if err != nil {
				if file != nil {
					file.Close()
				}
				continue
			}
			if file != nil {
				file.Close()
			}

			err = toml.Unmarshal(data, Cfg)
			if err != nil {
				continue
			}
			if Cfg.TcpServer.CloseClientPeriod < 300 {
				Cfg.TcpServer.CloseClientPeriod = 300
			}
			return
		}
		log.Fatal("cannot load config file.")
	}
	once.Do(fn)
}
