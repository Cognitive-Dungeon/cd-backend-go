package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

func normalize(raw rawConfig) (*Config, error) {
	port, err := parsePort(raw.Server.Port)
	if err != nil {
		return nil, err
	}

	level, err := logrus.ParseLevel(strings.ToLower(raw.Log.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q", raw.Log.Level)
	}

	format, err := parseLogFormat(raw.Log.Format)
	if err != nil {
		return nil, err
	}

	if raw.Server.ShardID > 255 {
		return nil, fmt.Errorf("shard id out of range: %d", raw.Server.ShardID)
	}

	return &Config{
		Server: ServerConfig{
			Port:    port,
			ShardID: uint8(raw.Server.ShardID),
		},
		Log: LogConfig{
			Level:  level,
			Format: format,
		},
		Sim: SimulationConfig{
			MasterSeed: raw.Sim.MasterSeed,
		},
		Replay: ReplayConfig{
			Path: raw.Replay.Path,
		},
	}, nil
}

func parsePort(p int) (uint16, error) {
	if p <= 0 || p > 65535 {
		return 0, fmt.Errorf("invalid port %d", p)
	}
	return uint16(p), nil
}
