package compose

import (
	"testing"
)

func TestProjectValidateRequiresName(t *testing.T) {
	services := "\nservices:\n  app:\n    image: alpine\n"
	cases := []struct {
		name, yaml, env, want string
	}{
		{"missing", services, "", ""},
		{"null", "name: null" + services, "", ""},
		{"empty", "name: ''" + services, "", ""},
		{"spaces", "name: '   '" + services, "", ""},
		{"valid", "name: example" + services, "", "example"},
		{"interpolated", "name: ${APP_NAME}" + services, "APP_NAME=example", "example"},
		{"unset variable", "name: ${ISRVD_TEST_UNSET_NAME:-}" + services, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var env *string
			if tc.env != "" {
				env = &tc.env
			}
			project, err := ProjectValidateWithoutEnvFiles(t.Context(), tc.yaml, env)
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected error, got project %q", project.Name)
				}
			} else if err != nil || project.Name != tc.want {
				t.Fatalf("project = %v, err = %v, want %s", project, err, tc.want)
			}
		})
	}
}
