package config

type Config struct {
	Server ServerConfig
	Log    LogConfig
	Sim    SimulationConfig
	Replay ReplayConfig
}

type ServerConfig struct {
	ShardID uint8
	Port    uint16
}

type SimulationConfig struct {
	MasterSeed int64
}

type ReplayConfig struct {
	Path string
}
