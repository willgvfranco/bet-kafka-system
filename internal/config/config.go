package config

import "os"

type Config struct {
	KafkaBrokers string
	RedisAddr    string
	MongoURI     string
	HTTPPort     string
	MetricsPort  string
}

func Load() *Config {
	return &Config{
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		MongoURI:     getEnv("MONGO_URI", "mongodb://localhost:27017/betting"),
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		MetricsPort:  getEnv("METRICS_PORT", "9100"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
