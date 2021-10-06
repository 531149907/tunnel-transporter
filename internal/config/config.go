package config

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
)

type (
	AuthenticationType string
)

var (
	AppConfig *Config
)

const (
	None        AuthenticationType = "none"
	StaticToken AuthenticationType = "static-token"
)

type Config struct {
	Log struct {
		Level string
	}
	Server struct {
		AgentPort uint16 `yaml:"agent-port"`
		Http      struct {
			Port             uint16
			AgentIdHeaderKey string `yaml:"agent-id-header-key"`
		}
		Proxy struct {
			Authentication struct {
				Type AuthenticationType

				StaticToken struct {
					Token string
				} `yaml:"static-token"`
			}
		}
	}
	Agent struct {
		Id             string
		ServerEndpoint string `yaml:"server-endpoint"`
		LocalEndpoint  string `yaml:"local-endpoint"`
		Proxy          struct {
			Authentication struct {
				Type AuthenticationType

				StaticToken struct {
					Token string
				} `yaml:"static-token"`
			}
		}
	}
}

func ParseConfig(configPath string) error {
	bytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	AppConfig = &Config{}
	if err = yaml.Unmarshal(bytes, AppConfig); err != nil {
		return err
	}

	createLogger(AppConfig)

	return nil
}

func createLogger(config *Config) {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:               true,
		DisableColors:             false,
		ForceQuote:                false,
		DisableQuote:              true,
		EnvironmentOverrideColors: false,
		DisableTimestamp:          false,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05",
		DisableSorting:            false,
		SortingFunc:               nil,
		DisableLevelTruncation:    false,
		PadLevelText:              false,
		QuoteEmptyFields:          false,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			file := f.File + ":" + strconv.Itoa(f.Line)
			var filename string
			if fileLength := len(file); fileLength < 40 {
				filename = file + strings.Repeat(" ", 40-fileLength)
			} else {
				filename = file[fileLength-40:]
			}

			return " [" + filename + "] ", ""
		},
	})
	log.SetReportCaller(true)

	switch config.Log.Level {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	}
}
