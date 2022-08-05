package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	if strings.ToLower(os.Getenv("LOGLEVEL")) == "production" {
		log.Formatter = new(logrus.JSONFormatter)
	}

	log.SetReportCaller(true)
}

func GetLogger() *logrus.Logger {
	switch strings.ToLower(os.Getenv("LOGLEVEL")) {
	case "trace":
		log.Level = logrus.TraceLevel
		break
	case "error":
		log.Level = logrus.ErrorLevel
		break
	case "warn":
		log.Level = logrus.WarnLevel
		break
	case "info":
		log.Level = logrus.InfoLevel
		break
	default:
		log.Level = logrus.DebugLevel
		break
	}

	return log
}
