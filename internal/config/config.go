package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DatabaseURL       string        `env:"DATABASE_URL,notEmpty" envDefault:"postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable"`
	RedisURL          string        `env:"REDIS_URL,notEmpty"    envDefault:"redis://localhost:6379/0"`
	Port              string        `env:"GYM_API_PORT"          envDefault:"8081"`
	AuthBaseURL       string        `env:"AUTH_BASE_URL"         envDefault:"http://localhost:8080"`
	S3Endpoint        string        `env:"S3_ENDPOINT"           envDefault:"http://localhost:9000"`
	S3AccessKey       string        `env:"S3_ACCESS_KEY"         envDefault:"minioadmin"`
	S3SecretKey       string        `env:"S3_SECRET_KEY"         envDefault:"minioadmin"`
	S3Bucket          string        `env:"S3_BUCKET"             envDefault:"gym-media"`
	S3UseSSL          bool          `env:"S3_USE_SSL"            envDefault:"false"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT"   envDefault:"5s"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT"      envDefault:"10s"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
