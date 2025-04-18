package config

type LoggingConfig struct {
	Level      string
	Production bool
	JsonFormat bool
}

func LoadLoggingConfig() *LoggingConfig {
	config := &LoggingConfig{
		Level:      getEnv("LOG_LEVEL", "info"),
		Production: getEnv("APP_ENV", "development") == "production",
		JsonFormat: getEnvAsBool("LOG_JSON", false),
	}

	if config.Production {
		config.JsonFormat = true
	}

	return config
}
