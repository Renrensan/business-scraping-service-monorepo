package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        int
	NodeEnv     string
	LogLevel    string
	LogFormat   string
	DatabaseURL string
	RedisURL    string
	FrontendURL string

	SuperAdminEmail    string
	SuperAdminUsername string
	SuperAdminPassword string

	MaxConcurrentScrapers  int
	MaxConcurrentEnrichers int
	DefaultUserAgent       string
	Proxies                []string
	GooglePlacesAPIKey     string
	// GooglePlacesBaseURL overrides the Places API host; only for local end-to-end tests with a fake server.
	GooglePlacesBaseURL string
	// ShutdownGrace is how long running jobs may keep going after SIGTERM before they are requeued.
	// Keep it below the container's stop timeout (docker compose stop_grace_period).
	ShutdownGrace time.Duration
	// EnableWebScraper allows the experimental Google Maps HTML scraper (used for engine "web", or
	// when no API key is set). It parses Maps HTML with a regex and breaks silently when Google
	// changes the page, so it is off by default.
	EnableWebScraper bool

	// WhatsApp alerts through Evolution API; all four must be set, otherwise alerts are only logged.
	EvolutionAPIURL     string
	EvolutionAPIKey     string
	EvolutionInstance   string
	AlertWhatsAppNumber string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Overload("../.env")
	_ = godotenv.Overload(".env")

	port := getEnvInt("PORT", 8080)
	nodeEnv := cleanStr(getEnv("NODE_ENV", "development"))
	logLevel := cleanStr(getEnv("LOG_LEVEL", "debug"))
	logFormat := cleanStr(getEnv("LOG_FORMAT", "text"))

	dbURL := cleanStr(getEnv("DATABASE_URL", "postgres://postgres:postgres@127.0.0.1:5434/bizscrape_db?sslmode=disable"))
	redisURL := cleanStr(getEnv("REDIS_URL", "redis://127.0.0.1:6380"))
	frontendURL := cleanStr(getEnv("FRONTEND_URL", "http://localhost:5173"))

	superAdminEmail := cleanStr(getEnv("SUPERADMIN_EMAIL", "admin@bizscrape.local"))
	superAdminUsername := cleanStr(getEnv("SUPERADMIN_USERNAME", "admin"))
	superAdminPassword := cleanStr(getEnv("SUPERADMIN_PASSWORD", "Admin12345!"))

	maxScrapers := getEnvInt("MAX_CONCURRENT_SCRAPERS", 5)
	maxEnrichers := getEnvInt("MAX_CONCURRENT_ENRICHERS", 10)
	defaultUA := cleanStr(getEnv("DEFAULT_USER_AGENT", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"))
	googlePlacesAPIKey := cleanStr(getEnv("GOOGLE_PLACES_API_KEY", ""))

	rawProxies := cleanStr(getEnv("PROXIES", ""))
	var proxies []string
	if rawProxies != "" {
		for _, p := range strings.Split(rawProxies, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				proxies = append(proxies, trimmed)
			}
		}
	}

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
		Port:                   port,
		NodeEnv:                nodeEnv,
		LogLevel:               logLevel,
		LogFormat:              logFormat,
		DatabaseURL:            dbURL,
		RedisURL:               redisURL,
		FrontendURL:            frontendURL,
		SuperAdminEmail:        superAdminEmail,
		SuperAdminUsername:     superAdminUsername,
		SuperAdminPassword:     superAdminPassword,
		GooglePlacesBaseURL:    cleanStr(getEnv("GOOGLE_PLACES_BASE_URL", "")),
		ShutdownGrace:          time.Duration(getEnvInt("SHUTDOWN_GRACE_SECONDS", 25)) * time.Second,
		EnableWebScraper:       strings.EqualFold(cleanStr(getEnv("ENABLE_WEB_SCRAPER", "false")), "true"),
		EvolutionAPIURL:        cleanStr(getEnv("EVOLUTION_API_URL", "")),
		EvolutionAPIKey:        cleanStr(getEnv("EVOLUTION_API_KEY", "")),
		EvolutionInstance:      cleanStr(getEnv("EVOLUTION_INSTANCE", "")),
		AlertWhatsAppNumber:    cleanStr(getEnv("ALERT_WHATSAPP_NUMBER", "")),
		MaxConcurrentScrapers:  maxScrapers,
		MaxConcurrentEnrichers: maxEnrichers,
		DefaultUserAgent:       defaultUA,
		Proxies:                proxies,
		GooglePlacesAPIKey:     googlePlacesAPIKey,
	}, nil
}

func cleanStr(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	return strings.TrimSpace(s)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		clean := cleanStr(val)
		if intVal, err := strconv.Atoi(clean); err == nil {
			return intVal
		}
	}
	return defaultVal
}
