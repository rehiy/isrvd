package agui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestTranslateFinishReason(t *testing.T) {
	tests := []struct {
		reason string
		ok     bool
	}{
		{"stop", true},
		{"tool_calls", true},
		{"length", false},
		{"content_filter", false},
		{"unexpected", false},
	}
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			stream := fmt.Sprintf("data: {\"choices\":[{\"delta\":{\"content\":\"part\"},\"finish_reason\":%q}]}\n\n", tt.reason)
			var output bytes.Buffer
			err := translate(context.Background(), NewEncoder(&output), strings.NewReader(stream), RunAgentInput{
				ThreadID: "thread", RunID: "run",
			})
			if tt.ok && err != nil {
				t.Fatal(err)
			}
			if !tt.ok && (err == nil || !strings.Contains(err.Error(), tt.reason)) {
				t.Fatalf("expected finish-reason error, got %v", err)
			}

			var events []Event
			for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
				var event Event
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					t.Fatal(err)
				}
				events = append(events, event)
			}
			last := events[len(events)-1].Type
			if tt.ok && last != RunFinished {
				t.Fatalf("successful stream ended with %s", last)
			}
			if !tt.ok {
				for _, event := range events {
					if event.Type == RunFinished {
						t.Fatal("incomplete output was reported as successfully finished")
					}
				}
			}
		})
	}
}

func TestTranslateMissingTerminalEvent(t *testing.T) {
	stream := "data: {\"choices\":[{\"delta\":{\"content\":\"part\"}}]}\n\n"
	var output bytes.Buffer
	err := translate(context.Background(), NewEncoder(&output), strings.NewReader(stream), RunAgentInput{RunID: "run"})
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") {
		t.Fatalf("missing terminal marker must fail, got %v", err)
	}
	if strings.Contains(output.String(), string(RunFinished)) {
		t.Fatal("missing terminal marker was reported as finished")
	}
}
