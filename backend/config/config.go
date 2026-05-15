package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBHost       string
	DBPort       int
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecret    string
	DashboardURL string
	ServerPort   int
	PasswordMaxAge int // 密码有效期（天）
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "3306"))
	serverPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	passwordMaxAge, _ := strconv.Atoi(getEnv("PASSWORD_MAX_AGE", "90"))

	return &Config{
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         port,
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "k8sgate"),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-in-production"),
		DashboardURL:   getEnv("DASHBOARD_URL", "https://kubernetes-dashboard.kubernetes-dashboard.svc"),
		ServerPort:     serverPort,
		PasswordMaxAge: passwordMaxAge,
	}
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + strconv.Itoa(c.DBPort) + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
