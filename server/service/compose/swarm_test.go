package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"

	"isrvd/pkgs/compose"
)

func TestRestoreComposeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(path, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := restoreComposeFile(path, "old", true); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "old" {
		t.Fatalf("restore existing file: content=%q err=%v", got, err)
	}
	if err := restoreComposeFile(path, "old", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("originally absent compose file should be removed: %v", err)
	}
	if err := restoreComposeFile(path, "old", false); err != nil {
		t.Fatalf("removing an already absent file: %v", err)
	}
}

func TestSwarmServiceSpecUnchanged(t *testing.T) {
	project := &types.Project{
		Name: "demo",
		Services: types.Services{
			"api": {Name: "api", Image: "nginx:alpine"},
		},
	}
	spec, err := compose.ServiceToSwarmSpec(project, project.Services["api"])
	if err != nil {
		t.Fatal(err)
	}
	if !swarmServiceSpecUnchanged(project, spec, false) {
		t.Fatal("identical service spec should not be updated")
	}
	if swarmServiceSpecUnchanged(project, spec, true) {
		t.Fatal("forcePull must bypass the unchanged check")
	}
	spec.TaskTemplate.ContainerSpec.Image = "nginx:stable"
	if swarmServiceSpecUnchanged(project, spec, false) {
		t.Fatal("changed image should trigger service update")
	}
}
