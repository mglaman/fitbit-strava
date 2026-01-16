package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	FitbitClientID     string
	FitbitClientSecret string
	FitbitRedirectURL  string

	StravaClientID     string
	StravaClientSecret string
	StravaRedirectURL  string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		// .env is optional if env vars are set otherwise, but for this CLI it's expected
		fmt.Println("Warning: Error loading .env file")
	}

	cfg := &Config{
		FitbitClientID:     os.Getenv("FITBIT_CLIENT_ID"),
		FitbitClientSecret: os.Getenv("FITBIT_CLIENT_SECRET"),
		FitbitRedirectURL:  os.Getenv("FITBIT_REDIRECT_URL"),

		StravaClientID:     os.Getenv("STRAVA_CLIENT_ID"),
		StravaClientSecret: os.Getenv("STRAVA_CLIENT_SECRET"),
		StravaRedirectURL:  os.Getenv("STRAVA_REDIRECT_URL"),
	}

	if cfg.FitbitClientID == "" || cfg.FitbitClientSecret == "" {
		return nil, fmt.Errorf("missing Fitbit credentials in .env")
	}
	if cfg.StravaClientID == "" || cfg.StravaClientSecret == "" {
		return nil, fmt.Errorf("missing Strava credentials in .env")
	}

	return cfg, nil
}
