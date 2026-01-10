package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogConfig struct {
	Format LogFormat
	Level  logrus.Level
}

type LogFormat string

const (
	LogText LogFormat = "text"
	LogJSON LogFormat = "json"
)

func parseLogFormat(v string) (LogFormat, error) {
	switch strings.ToLower(v) {
	case "text":
		return LogText, nil
	case "json":
		return LogJSON, nil
	default:
		return "", fmt.Errorf("invalid log format %q", v)
	}
}
