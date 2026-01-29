package config

type rawConfig struct {
	Server struct {
		Port    int
		ShardID uint
	}

	Log struct {
		Level  string
		Format string
	}

	Sim struct {
		MasterSeed int64
	}

	Replay struct {
		Path string
	}
}

func defaultRawConfig() rawConfig {
	var raw rawConfig

	raw.Server.Port = 8080
	raw.Server.ShardID = 0
	raw.Log.Level = "info"
	raw.Log.Format = "text"
	raw.Sim.MasterSeed = 0

	return raw
}
