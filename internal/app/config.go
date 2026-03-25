package app

import "os"

// Config holds runtime settings for the web server.
type Config struct {
	Addr        string
	DatabaseURL string
}

// LoadConfig reads configuration from environment with safe defaults.
func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Addr:        ":" + port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}
