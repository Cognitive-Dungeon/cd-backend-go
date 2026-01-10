package config

func Load() (*Config, error) {
	raw := defaultRawConfig()

	applyFlags(&raw)

	if err := applyEnv(&raw); err != nil {
		return nil, err
	}

	return normalize(raw)
}
