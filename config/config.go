package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"tunnel-transporter/config/agent"
	"tunnel-transporter/config/log"
	"tunnel-transporter/config/server"
)

var (
	ClientConfig *Config
)

type Config struct {
	Log    log.Config `yaml:"log"`
	Server server.Config
	Agent  agent.Config
}

func ParseConfig(configPath string) error {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	ClientConfig = &Config{}
	if err = yaml.Unmarshal(bytes, ClientConfig); err != nil {
		return err
	}

	_ = applyConfig(ClientConfig)

	return nil
}

func applyConfig(clientConfig *Config) error {
	log.CreateLogger(&clientConfig.Log)
	_ = server.CreateServer(&clientConfig.Server)
	_ = agent.CreateAgent(&clientConfig.Agent)
	return nil
}
