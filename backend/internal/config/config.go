// Package config holds the application settings, read once from the environment at startup
// (main loads the .env file first).
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Database struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// URL returns the connection string of the given database on this server
func (d Database) URL(name string) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, d.Port),
		Path:   "/" + name,
	}
	return u.String()
}

type Renault struct {
	Username       string
	Password       string
	Country        string
	GigyaAPIKey    string // empty: the public key of the MyRenault app
	KamereonAPIKey string // empty: the public key of the MyRenault app
}

type WhatsApp struct {
	Phone  string
	APIKey string
}

type Config struct {
	Port          string
	CORSOrigins   []string // empty: every origin is allowed
	JWTSecret     string
	AdminPassword string // used once, to create the admin user when it does not exist
	MCPAuthToken  string // empty: /mcp needs no token

	SaveTagValueAPIKey string
	LocationPostAPIKey string

	Database Database
	Renault  Renault
	WhatsApp WhatsApp

	EcoForestAuth   string // value of the "Authorization: Basic" header of the EcoForest web interface
	RohanAirTuyaID  string
	RohanAirTuyaKey string

	Latitude  float64 // house location for sunrise/sunset triggers; both zero: those triggers are skipped
	Longitude float64
}

// Load reads the settings from the environment. It fails when a required variable is missing or
// a numeric one is not a number.
func Load() (*Config, error) {
	var errs []error

	cfg := &Config{
		Port:               env("PORT", "4000"),
		CORSOrigins:        splitList(os.Getenv("CORS_ORIGINS")),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		MCPAuthToken:       os.Getenv("MCP_AUTH_TOKEN"),
		SaveTagValueAPIKey: os.Getenv("SAVE_TAG_VALUE_API_KEY"),
		LocationPostAPIKey: os.Getenv("LOCATION_POST_API_KEY"),
		Database: Database{
			Host:     os.Getenv("DB_HOST"),
			Port:     env("DB_PORT", "5432"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
		},
		Renault: Renault{
			Username:       os.Getenv("RENAULT_USERNAME"),
			Password:       os.Getenv("RENAULT_PASSWORD"),
			Country:        env("RENAULT_COUNTRY", "PT"),
			GigyaAPIKey:    os.Getenv("RENAULT_GIGYA_API_KEY"),
			KamereonAPIKey: os.Getenv("RENAULT_KAMEREON_API_KEY"),
		},
		WhatsApp: WhatsApp{
			Phone:  os.Getenv("WHATSAPP_PHONE"),
			APIKey: os.Getenv("WHATSAPP_API_KEY"),
		},
		EcoForestAuth:   os.Getenv("ECOFOREST_AUTH"),
		RohanAirTuyaID:  os.Getenv("ROHAN_AIR_TUYA_ID"),
		RohanAirTuyaKey: os.Getenv("ROHAN_AIR_TUYA_KEY"),
	}

	var err error
	if cfg.Latitude, err = envFloat("LATITUDE", -90, 90); err != nil {
		errs = append(errs, err)
	}
	if cfg.Longitude, err = envFloat("LONGITUDE", -180, 180); err != nil {
		errs = append(errs, err)
	}

	for _, name := range []string{"JWT_SECRET", "DB_HOST", "DB_NAME", "DB_USER"} {
		if os.Getenv(name) == "" {
			errs = append(errs, fmt.Errorf("%s environment variable is required", name))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return cfg, nil
}

func env(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a number, got %q", name, value)
	}
	return parsed, nil
}

// envFloat reads an optional number between min and max; empty is 0
func envFloat(name string, min float64, max float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < min || parsed > max {
		return 0, fmt.Errorf("%s must be a number between %g and %g, got %q", name, min, max, value)
	}
	return parsed, nil
}

// splitList splits "a, b,c" into ["a", "b", "c"], skipping empty items
func splitList(text string) []string {
	var items []string
	for _, item := range strings.Split(text, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
