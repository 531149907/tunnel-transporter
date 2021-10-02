package log

import (
	log "github.com/sirupsen/logrus"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	Level string `yaml:"level"`
}

func CreateLogger(logConfig *Config) {
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

	switch logConfig.Level {
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
