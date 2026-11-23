package compose

import (
	"testing"
)

func TestRedeployRequiresMatchingName(t *testing.T) {
	current := "name: example\nservices:\n  app:\n    image: alpine\n"
	cases := []struct {
		name, content string
		ok            bool
	}{
		{"missing", "services:\n  app:\n    image: alpine\n", false},
		{"empty", "name: ''\nservices:\n  app:\n    image: alpine\n", false},
		{"renamed", "name: other\nservices:\n  app:\n    image: alpine\n", false},
		{"matching", current, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := new(Service).resolveRedeployContent(t.Context(), "example", t.TempDir(), current, RedeployRequest{Content: &tc.content})
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want success %v", err, tc.ok)
			}
		})
	}
}
