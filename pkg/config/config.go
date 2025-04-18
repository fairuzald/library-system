package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppName            string        `mapstructure:"APP_NAME"`
	AppEnv             string        `mapstructure:"APP_ENV"`
	ServerPort         string        `mapstructure:"SERVER_PORT"`
	DBHost             string        `mapstructure:"DB_HOST"`
	DBPort             string        `mapstructure:"DB_PORT"`
	DBName             string        `mapstructure:"DB_NAME"`
	DBUser             string        `mapstructure:"DB_USER"`
	DBPassword         string        `mapstructure:"DB_PASSWORD"`
	DBSSLMode          string        `mapstructure:"DB_SSLMODE"`
	JWTSecret          string        `mapstructure:"JWT_SECRET"`
	JWTExpirationHours int           `mapstructure:"JWT_EXPIRATION_HOURS"`
	AccessTokenExpiry  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY"`
	RefreshTokenExpiry time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY"`
	RedisHost          string        `mapstructure:"REDIS_HOST"`
	RedisPort          string        `mapstructure:"REDIS_PORT"`
	RedisPassword      string        `mapstructure:"REDIS_PASSWORD"`
	GRPCPort           string        `mapstructure:"GRPC_PORT"`
	LogLevel           string        `mapstructure:"LOG_LEVEL"`
	BookServiceURL     string        `mapstructure:"BOOK_SERVICE_URL"`
	CategoryServiceURL string        `mapstructure:"CATEGORY_SERVICE_URL"`
	UserServiceURL     string        `mapstructure:"USER_SERVICE_URL"`
}

func LoadConfig(path string) (*Config, error) {
	_ = godotenv.Load(path)

	config := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		AccessTokenExpiry:  getEnvAsDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute),
		RefreshTokenExpiry: getEnvAsDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
		RedisHost:          getEnv("REDIS_HOST", "localhost"),
		RedisPort:          getEnv("REDIS_PORT", "6379"),
		GRPCPort:           getEnv("GRPC_PORT", "50051"),
	}

	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(&config); err != nil {
			return nil, fmt.Errorf("unable to decode config into struct: %w", err)
		}
	}

	requiredEnvs := []string{
		"APP_NAME",
		"DB_NAME",
		"DB_USER",
		"DB_PASSWORD",
		"JWT_SECRET",
	}

	for _, env := range requiredEnvs {
		if viper.GetString(env) == "" && os.Getenv(env) == "" {
			return nil, fmt.Errorf("required environment variable not set: %s", env)
		}
	}

	if config.AppName == "" {
		config.AppName = os.Getenv("APP_NAME")
	}
	if config.DBName == "" {
		config.DBName = os.Getenv("DB_NAME")
	}
	if config.DBUser == "" {
		config.DBUser = os.Getenv("DB_USER")
	}
	if config.DBPassword == "" {
		config.DBPassword = os.Getenv("DB_PASSWORD")
	}
	if config.JWTSecret == "" {
		config.JWTSecret = os.Getenv("JWT_SECRET")
	}
	if config.BookServiceURL == "" {
		config.BookServiceURL = os.Getenv("BOOK_SERVICE_URL")
	}
	if config.CategoryServiceURL == "" {
		config.CategoryServiceURL = os.Getenv("CATEGORY_SERVICE_URL")
	}
	if config.UserServiceURL == "" {
		config.UserServiceURL = os.Getenv("USER_SERVICE_URL")
	}

	return config, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) GetRedisAddress() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
