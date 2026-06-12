package config

import (
	"fmt"
	"os"
	"strconv"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type ServerConfig struct {
	Port      int
	TableName string
}

type Config struct {
	DB     DBConfig
	Server ServerConfig
}

// FIX: sensitive — не логируем значение, только факт наличия
func getEnv(key string, sensitive bool) string {
	val, exists := os.LookupEnv(key)
	if !exists {
		fmt.Printf("WARNING: env variable %s is not set\n", key)
		return ""
	}
	if sensitive {
		fmt.Printf("OK: %s = [set]\n", key)
	} else {
		fmt.Printf("OK: %s = %s\n", key, val)
	}
	return val
}

func InitConfig() (*Config, error) {
	var cfg Config

	cfg.DB.Host = getEnv("DB_HOST", false)
	cfg.DB.Name = getEnv("DB_NAME", false)
	cfg.DB.User = getEnv("DB_USER", false)
	cfg.DB.Password = getEnv("DB_PASSWORD", true) // FIX: sensitive=true

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", false))
	if err != nil {
		return nil, fmt.Errorf("DB_PORT is not a valid number: %w", err)
	}
	cfg.DB.Port = dbPort

	appPort, err := strconv.Atoi(getEnv("APP_PORT", false))
	if err != nil {
		return nil, fmt.Errorf("APP_PORT is not a valid number: %w", err)
	}
	cfg.Server.Port = appPort

	cfg.Server.TableName = getEnv("DB_TABLENAME", false)

	return &cfg, nil
}
