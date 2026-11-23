package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"isrvd/server/config"
)

type webhookTransport func(*http.Request) (*http.Response, error)

func (f webhookTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func setupWebhookTest(t *testing.T, transport webhookTransport) {
	t.Helper()
	previous := webhookClient
	webhookClient = &http.Client{Transport: transport}
	t.Cleanup(func() {
		sendsCancel()
		sendsWG.Wait()
		sendsWG = new(sync.WaitGroup)
		webhookClient = previous
		sendsCtx, sendsCancel = context.WithCancel(context.Background())
		sendsClosed = false
	})
}

func webhookResponse() (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
}

func awaitWebhook(t *testing.T, started <-chan string, want string) {
	t.Helper()
	select {
	case got := <-started:
		if got != want {
			t.Fatalf("unexpected delivery %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("webhook did not start")
	}
}

func TestWebhookPendingStateIsBoundedAndLatest(t *testing.T) {
	started := make(chan string, 4)
	release := make(chan struct{})
	setupWebhookTest(t, func(r *http.Request) (*http.Response, error) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			return nil, err
		}
		started <- event.Title
		if event.Title == "first" {
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		}
		return webhookResponse()
	})
	hooks := []*config.WebhookConfig{{URL: "https://webhook.example.test"}}
	sendTo(hooks, "resource", &Event{Title: "first"})
	awaitWebhook(t, started, "first")
	for i := 0; i < 100; i++ {
		sendTo(hooks, "resource", &Event{Title: fmt.Sprint(i)})
	}
	sendsMu.Lock()
	pending := sendQueues[[2]string{"resource", hooks[0].URL}]
	active, queued := sendsActive, len(pending)
	sendsMu.Unlock()
	if active != 1 || queued != 1 {
		t.Fatalf("unbounded queue: active=%d pending=%d", active, queued)
	}
	sendTo(hooks, "other", &Event{Title: "independent"})
	awaitWebhook(t, started, "independent")
	close(release)
	awaitWebhook(t, started, "99")
	sendsWG.Wait()
	if len(started) != 0 || sendsActive != 0 || len(sendQueues) != 0 {
		t.Fatal("obsolete state sent or idle queue leaked")
	}
}

func TestWebhookCapacityAndShutdownCancellation(t *testing.T) {
	started := make(chan string, maxWebhookSends)
	setupWebhookTest(t, func(r *http.Request) (*http.Response, error) {
		started <- "started"
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	hooks := []*config.WebhookConfig{{URL: "https://webhook.example.test"}}
	for i := 0; i < maxWebhookSends+10; i++ {
		sendTo(hooks, "", &Event{})
	}
	sendsMu.Lock()
	active := sendsActive
	sendsMu.Unlock()
	if active != maxWebhookSends {
		t.Fatalf("send limit not enforced: %d", active)
	}
	awaitWebhook(t, started, "started")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if Shutdown(ctx) {
		t.Fatal("shutdown falsely reported completed blocked sends")
	}
	sendsWG.Wait()
	sendTo(hooks, "late", &Event{})
	if sendsActive != 0 || len(sendQueues) != 0 {
		t.Fatal("shutdown leaked sends or accepted a new event")
	}
}

func TestWebhookExpiredPendingIsSkipped(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	setupWebhookTest(t, func(r *http.Request) (*http.Response, error) {
		started <- "first"
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
		return webhookResponse()
	})
	hooks := []*config.WebhookConfig{{URL: "https://webhook.example.test"}}
	sendTo(hooks, "resource", &Event{})
	awaitWebhook(t, started, "first")
	sendsMu.Lock()
	sendQueues[[2]string{"resource", hooks[0].URL}] <- webhookDelivery{hook: hooks[0], deadline: time.Now().Add(-time.Second)}
	sendsMu.Unlock()
	close(release)
	sendsWG.Wait()
	if len(started) != 0 {
		t.Fatal("expired queued event was sent")
	}
}
