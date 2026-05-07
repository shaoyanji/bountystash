package app

import "os"

// Config holds runtime settings for the web server.
type Config struct {
	Addr         string
	DatabaseURL  string
	WebhookURL   string
	SupabaseURL  string
	SupabaseKey  string
}

// LoadConfig reads configuration from environment with safe defaults.
func LoadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Addr:         ":" + port,
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		WebhookURL:   os.Getenv("WEBHOOK_URL"),
		SupabaseURL:  os.Getenv("SUPABASE_URL"),
		SupabaseKey:  os.Getenv("SUPABASE_KEY"),
	}
}
