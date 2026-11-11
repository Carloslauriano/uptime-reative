package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTickCallsWebhookOnlyWhenHealthy(t *testing.T) {
	var status atomic.Int32
	status.Store(200)
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(status.Load()))
	}))
	defer health.Close()

	var hits atomic.Int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
	}))
	defer hook.Close()

	cfg := config{HealthURL: health.URL, WebhookURL: hook.URL, WebhookMethod: http.MethodPost}
	ctx := context.Background()
	var prev *bool

	if h := tick(ctx, http.DefaultClient, cfg, prev); !h || hits.Load() != 1 {
		t.Fatalf("healthy: want webhook called once, got healthy=%v hits=%d", h, hits.Load())
	}

	status.Store(503)
	if h := tick(ctx, http.DefaultClient, cfg, prev); h || hits.Load() != 1 {
		t.Fatalf("unhealthy: want webhook skipped, got healthy=%v hits=%d", h, hits.Load())
	}

	status.Store(200)
	if h := tick(ctx, http.DefaultClient, cfg, prev); !h || hits.Load() != 2 {
		t.Fatalf("recovered: want webhook called again, got healthy=%v hits=%d", h, hits.Load())
	}
}

func TestUnreachableIsUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	if ok, _ := isHealthy(context.Background(), http.DefaultClient, url); ok {
		t.Fatal("unreachable target must be unhealthy")
	}
}
