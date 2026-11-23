package agui

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTranslateSSEMultilineFrames(t *testing.T) {
	for _, delimiter := range []string{"\n", "\r\n", "\r"} {
		t.Run(strings.ReplaceAll(strings.ReplaceAll(delimiter, "\r", "CR"), "\n", "LF"), func(t *testing.T) {
			lines := []string{
				": keepalive",
				"",
				"event: completion.chunk",
				"data: {\"choices\":[",
				"data: {\"delta\":{\"content\":\"hello\"},\"finish_reason\":\"stop\"}",
				"data: ]}",
				"",
				"data: [DONE]",
				"",
			}
			src := strings.Join(lines, delimiter)
			var out bytes.Buffer
			if err := translate(context.Background(), NewEncoder(&out), strings.NewReader(src), RunAgentInput{RunID: "run"}); err != nil {
				t.Fatal(err)
			}
			var events []Event
			for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
				var ev Event
				if err := json.Unmarshal([]byte(line), &ev); err != nil {
					t.Fatal(err)
				}
				events = append(events, ev)
			}
			if len(events) != 5 {
				t.Fatalf("events=%d want 5: %s", len(events), out.String())
			}
			if events[2].Type != TextMessageContent || events[2].Delta != "hello" || events[4].Type != RunFinished {
				t.Fatalf("unexpected events: %#v", events)
			}
		})
	}
}

func TestSSEPayloadWithoutTrailingBlankLine(t *testing.T) {
	var out bytes.Buffer
	src := "data: {\"choices\":[{\"delta\":{\"content\":\"end\"},\"finish_reason\":\"stop\"}]}"
	err := translate(context.Background(), NewEncoder(&out), strings.NewReader(src), RunAgentInput{RunID: "run"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), string(RunFinished)) {
		t.Fatalf("final unterminated event lost: %s", out.String())
	}
}
