package compose

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/api/types/registry"
	dockerswarm "github.com/docker/docker/api/types/swarm"

	"isrvd/pkgs/docker"
	"isrvd/pkgs/swarm"
	"isrvd/server/config"
)

func TestSwarmComposeDelegatesImagePull(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yml")
	if err := os.WriteFile(configPath, []byte("server:\n  rootDirectory: "+root+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ISRVD_CONFIG", configPath)
	t.Setenv("ISRVD_MODE", "server")
	if err := config.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = config.Close() })

	const image = "registry.example.test/app:latest"
	const auth = "test-registry-auth"
	const pullError = "image pulls disabled on this daemon"
	firstDigest := "sha256:" + strings.Repeat("1", 64)
	secondDigest := "sha256:" + strings.Repeat("2", 64)
	var mu sync.Mutex
	var current dockerswarm.Service
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("API-Version", "1.51")
		_, path, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		var response any
		switch {
		case r.URL.Path == "/_ping":
			return
		case path == "services/app" && current.ID == "":
			http.Error(w, `{"message":"service not found"}`, http.StatusNotFound)
			return
		case path == "services/app" || path == "services/test-service":
			response = current
		case path == "networks/example_default":
			response = map[string]string{"Name": "example_default"}
		case path == "distribution/"+image+"/json":
			if r.Header.Get(registry.AuthHeader) != auth {
				t.Error("registry auth missing from digest lookup")
			}
			digest := firstDigest
			if current.ID != "" {
				digest = secondDigest
			}
			response = map[string]any{"Descriptor": map[string]string{"digest": digest}}
		case path == "services/create" || path == "services/test-service/update":
			if r.Header.Get(registry.AuthHeader) != auth {
				t.Error("registry auth missing from service spec submission")
			}
			var spec dockerswarm.ServiceSpec
			if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			previous := current.Spec
			current.PreviousSpec = &previous
			current.Spec = spec
			current.ID = "test-service"
			current.Version.Index++
			response = map[string]string{"ID": current.ID}
		case path == "containers/app/json":
			http.Error(w, `{"message":"container not found"}`, http.StatusNotFound)
			return
		case path == "images/create":
			http.Error(w, `{"message":"`+pullError+`"}`, http.StatusForbidden)
			return
		default:
			t.Errorf("unexpected Docker request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(engine.Close)

	dockerClient, err := docker.NewDockerService(&docker.DockerConfig{
		Host: strings.Replace(engine.URL, "http://", "tcp://", 1), ContainerRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dockerClient.Close() })
	swarmClient := swarm.NewSwarmService(dockerClient.Client(), func(string) string { return auth })
	svc, err := NewService(dockerClient, swarmClient)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	content := "name: example\nservices:\n  app:\n    image: " + image + "\n"
	assertImage := func(digest string) {
		t.Helper()
		service, err := swarmClient.ServiceInspectRaw(ctx, "test-service")
		if err != nil {
			t.Fatal(err)
		}
		if got := service.Spec.TaskTemplate.ContainerSpec.Image; got != image+"@"+digest {
			t.Fatalf("image = %q, want %s@%s", got, image, digest)
		}
	}
	if _, err := svc.SwarmDeploy(ctx, DeployRequest{Content: content, ForcePull: true}); err != nil {
		t.Fatalf("Swarm deploy: %v", err)
	}
	assertImage(firstDigest)
	if _, err := svc.SwarmRedeploy(ctx, "example", RedeployRequest{Content: &content, ForcePull: true}); err != nil {
		t.Fatalf("Swarm redeploy: %v", err)
	}
	assertImage(secondDigest)
	standalone := strings.Replace(content, "name: example", "name: standalone", 1)
	if _, err := svc.DockerDeploy(ctx, DeployRequest{Content: standalone, ForcePull: true}); err == nil || !strings.Contains(err.Error(), pullError) {
		t.Fatalf("Docker deploy should pre-pull the image, got %v", err)
	}
}
