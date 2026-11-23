package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"isrvd/server/config"
)

type gatewayProbeTransport func(*http.Request) (*http.Response, error)

func (f gatewayProbeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRetryGatewaysPublishesIndependentSuccess(t *testing.T) {
	previous := config.Current()
	cfg := *previous
	cfg.Apisix = &config.ApisixConfig{AdminURL: "https://apisix.example.test"}
	cfg.Caddy = &config.CaddyConfig{AdminURL: "https://caddy.example.test"}
	cfg.Notify = &config.NotifyConfig{}
	config.Publish(&cfg)
	t.Cleanup(func() { config.Publish(previous) })
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	for _, healthy := range []string{"apisix", "caddy"} {
		t.Run(healthy, func(t *testing.T) {
			http.DefaultTransport = gatewayProbeTransport(func(r *http.Request) (*http.Response, error) {
				if !strings.HasPrefix(r.URL.Host, healthy+".") {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				body := "{}"
				if healthy == "apisix" {
					body = `{"list":[]}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			app := &App{servicesCtx: t.Context()}
			app.availability.Store(&availableServices{probes: map[string]func(context.Context) bool{}})
			app.retryGateways(t.Context())
			if !app.isServiceAvailable(healthy) {
				t.Fatal("successful gateway was discarded by sibling timeout")
			}
			if len(app.availability.Load().probes) != 1 {
				t.Fatal("unavailable sibling was marked ready")
			}
			app.faultWatcher.Stop()
		})
	}
	t.Run("parent cancellation", func(t *testing.T) {
		http.DefaultTransport = gatewayProbeTransport(func(r *http.Request) (*http.Response, error) {
			return nil, r.Context().Err()
		})
		app := &App{servicesCtx: t.Context()}
		initial := &availableServices{probes: map[string]func(context.Context) bool{}}
		app.availability.Store(initial)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		app.retryGateways(ctx)
		if app.availability.Load() != initial {
			t.Fatal("cancelled lifecycle published availability")
		}
	})
}
