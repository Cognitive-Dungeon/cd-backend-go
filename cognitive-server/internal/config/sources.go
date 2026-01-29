package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func applyFlags(raw *rawConfig) {
	flag.IntVar(&raw.Server.Port, "port", raw.Server.Port, "Server port")
	flag.StringVar(&raw.Log.Level, "log", raw.Log.Level, "Log level")
	flag.StringVar(&raw.Log.Format, "log-format", raw.Log.Format, "Log format")
	flag.StringVar(&raw.Replay.Path, "replay", raw.Replay.Path, "Replay path")
	flag.Int64Var(&raw.Sim.MasterSeed, "seed", raw.Sim.MasterSeed, "Master seed for generation")
	flag.Parse()
}

func applyEnv(raw *rawConfig) error {
	if v := os.Getenv("CD_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("CD_PORT: %w", err)
		}
		raw.Server.Port = p
	}

	if v := os.Getenv("CD_LOG_LEVEL"); v != "" {
		raw.Log.Level = v
	}

	if v := os.Getenv("CD_LOG_FORMAT"); v != "" {
		raw.Log.Format = v
	}

	if v := os.Getenv("CD_REPLAY"); v != "" {
		raw.Replay.Path = v
	}

	if v := os.Getenv("CD_MASTER_SEED"); v != "" {
		raw.Sim.MasterSeed, _ = strconv.ParseInt(v, 10, 64)
	}

	return nil
}
