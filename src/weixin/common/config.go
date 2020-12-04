package common

import (
	"errors"
	"gopkg.in/ini.v1"
)

type config struct {
	WebAddress   string `ini:"web"`
	RedisAddress string `ini:"redis"`
	DataFile     string `ini:"data_file"`
	IniCfg       *ini.File
}

var Config = new(config)

func ParseConfig(configPath string) (*config, error) {
	if len(configPath) == 0 {
		return nil, errors.New("error config path")
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, err
	}

	//只进行读操作 用于提升性能
	cfg.BlockMode = false

	err = cfg.MapTo(Config)
	if err != nil {
		return nil, err
	}

	if len(Config.WebAddress) == 0 || len(Config.RedisAddress) == 0 {
		return nil, errors.New("error config address")
	}

	Config.IniCfg = cfg

	Logger.Printf("load server config %+v", Config)

	return Config, nil
}
