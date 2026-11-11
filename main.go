package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type config struct {
	HealthURL     string
	WebhookURL    string
	WebhookMethod string
	Interval      time.Duration
	Timeout       time.Duration
}

func loadConfig() (config, error) {
	c := config{
		HealthURL:     os.Getenv("HEALTH_URL"),
		WebhookURL:    os.Getenv("WEBHOOK_URL"),
		WebhookMethod: strings.ToUpper(envOr("WEBHOOK_METHOD", http.MethodPost)),
	}
	if c.HealthURL == "" || c.WebhookURL == "" {
		return c, errors.New("HEALTH_URL and WEBHOOK_URL are required")
	}
	var err error
	if c.Interval, err = durationEnv("CHECK_INTERVAL", time.Minute); err != nil {
		return c, err
	}
	if c.Timeout, err = durationEnv("REQUEST_TIMEOUT", 10*time.Second); err != nil {
		return c, err
	}
	return c, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid %s %q: use a positive duration like 1m", key, v)
	}
	return d, nil
}

// isHealthy returns true when the health endpoint answers with a 2xx status.
func isHealthy(ctx context.Context, client *http.Client, url string) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, resp.Status
}

func callWebhook(ctx context.Context, client *http.Client, method, url string) error {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "uptime-reative")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

// tick performs one check; it calls the webhook only when the target is healthy.
// prev is the previous health state (nil on first run); the new state is returned.
func tick(ctx context.Context, client *http.Client, c config, prev *bool) bool {
	healthy, detail := isHealthy(ctx, client, c.HealthURL)
	if prev == nil || *prev != healthy {
		slog.Info("health state changed", "healthy", healthy, "detail", detail)
	}
	if !healthy {
		slog.Debug("unhealthy, webhook skipped", "detail", detail)
		return false
	}
	if err := callWebhook(ctx, client, c.WebhookMethod, c.WebhookURL); err != nil {
		slog.Error("webhook failed", "error", err)
	} else {
		slog.Debug("webhook called")
	}
	return true
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if os.Getenv("LOG_LEVEL") == "debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := &http.Client{Timeout: cfg.Timeout}
	slog.Info("starting", "health_url", cfg.HealthURL, "interval", cfg.Interval.String(), "method", cfg.WebhookMethod)

	var prev *bool
	run := func() {
		h := tick(ctx, client, cfg, prev)
		prev = &h
	}
	run()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return
		case <-ticker.C:
			run()
		}
	}
}
