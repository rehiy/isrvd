package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"isrvd/server/service/copilot"
)

func TestDecodeAGUIInput(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		maxBytes   int64
		wantRunID  string
		wantTooBig bool
		wantError  bool
	}{
		{"valid", "{\"threadId\":\"thread\",\"runId\":\"run\",\"messages\":[]}", 1024, "run", false, false},
		{"trailing whitespace", " \n{\"runId\":\"run\"}\n\t ", 1024, "run", false, false},
		{"two requests", "{\"runId\":\"run\"}{\"runId\":\"second\"}", 1024, "", false, true},
		{"trailing text", "{\"runId\":\"run\"}invalid", 1024, "", false, true},
		{"incomplete", "{\"runId\":", 1024, "", false, true},
		{"empty", "", 1024, "", false, true},
		{"oversize JSON", "{\"runId\":\"run\"}", 4, "", true, true},
		{"oversize trailing whitespace", "{\"runId\":\"run\"}" + strings.Repeat(" ", 64), 32, "", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/copilot/agui", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			var input copilot.AGUIInput
			err := decodeAGUIInput(w, r, tt.maxBytes, &input)
			if (err != nil) != tt.wantError {
				t.Fatalf("decode error = %v, wantError %v", err, tt.wantError)
			}
			if input.RunID != tt.wantRunID && !tt.wantError {
				t.Fatalf("runId = %q, want %q", input.RunID, tt.wantRunID)
			}
			var maxError *http.MaxBytesError
			if errors.As(err, &maxError) != tt.wantTooBig {
				t.Fatalf("MaxBytesError = %v, want %v (error=%v)", errors.As(err, &maxError), tt.wantTooBig, err)
			}
		})
	}
}
