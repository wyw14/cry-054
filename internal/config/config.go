package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address         string
	DatabaseURL     string
	RequestTimeout  time.Duration
	AttachmentRoot  string
	MaxUploadBytes  int64
	LogLevel        string
	AllowedOrigins  []string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	requestTimeout, err := durationEnv("APP_REQUEST_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := durationEnv("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	maxUploadBytes, err := int64Env("APP_MAX_UPLOAD_BYTES", 5*1024*1024)
	if err != nil {
		return Config{}, err
	}
	attachmentRoot := stringEnv("APP_ATTACHMENT_ROOT", "./runtime/attachments")
	absoluteRoot, err := filepath.Abs(attachmentRoot)
	if err != nil {
		return Config{}, fmt.Errorf("resolve attachment root: %w", err)
	}
	if maxUploadBytes < 1024 || maxUploadBytes > 50*1024*1024 {
		return Config{}, fmt.Errorf("APP_MAX_UPLOAD_BYTES must be between 1 KiB and 50 MiB")
	}
	logLevel := strings.ToLower(stringEnv("APP_LOG_LEVEL", "info"))
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
		return Config{}, fmt.Errorf("unsupported APP_LOG_LEVEL %q", logLevel)
	}
	origins := splitCSV(stringEnv("APP_ALLOWED_ORIGINS", "http://localhost:5173"))
	if len(origins) == 0 {
		return Config{}, fmt.Errorf("APP_ALLOWED_ORIGINS must contain at least one origin")
	}
	return Config{
		Address:         stringEnv("APP_ADDR", ":8080"),
		DatabaseURL:     strings.TrimSpace(os.Getenv("APP_DATABASE_URL")),
		RequestTimeout:  requestTimeout,
		AttachmentRoot:  absoluteRoot,
		MaxUploadBytes:  maxUploadBytes,
		LogLevel:        logLevel,
		AllowedOrigins:  origins,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func stringEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return value, nil
}

func int64Env(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return value, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	return values
}
