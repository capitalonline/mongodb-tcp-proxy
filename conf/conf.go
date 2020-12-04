package conf

import (
	"flag"
	"github.com/BurntSushi/toml"
)

type Config struct {
	App     *App
	Log     *Log
	Storage *Storage
}

var (
	// config
	confPath string
	// Conf .
	Conf = &Config{}
)

type Storage struct {
	Type string
	DSN  string
}
type Log struct {
	Path string
}
type App struct {
	Host string
	Port string
}

func init() {
	flag.StringVar(&confPath, "conf", "./application.toml", ".toml config file")
}

func Init() error {
	_, err := toml.DecodeFile(confPath, &Conf)
	return err
}
