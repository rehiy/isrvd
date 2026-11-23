package compose

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestProjectNetworksUsesActiveServiceReferences(t *testing.T) {
	project := &types.Project{
		Networks: types.Networks{
			"used":   {Name: "shared", External: true},
			"unused": {Name: "missing-network", External: true},
			"idle":   {Name: "idle-network"},
		},
		Services: types.Services{
			"app":   {Networks: map[string]*types.ServiceNetworkConfig{"used": {}}},
			"host":  {NetworkMode: "host"},
			"named": {NetworkMode: "existing-network"},
		},
	}
	networks := ProjectNetworks(project)
	if len(networks) != 2 || !networks["shared"].External || !networks["existing-network"].External {
		t.Fatalf("unexpected network references: %#v", networks)
	}
}

func TestProjectNetworksIgnoresDisabledProfiles(t *testing.T) {
	project, err := LoadProjectFromContentInDir(t.Context(), `
name: network-test
services:
  app:
    image: alpine
    network_mode: none
  disabled:
    image: alpine
    profiles: [optional]
    networks: [unused]
networks:
  unused:
    external: true
    name: missing-network
  idle: {}
`, t.TempDir(), "network-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Networks) == 0 {
		t.Fatal("test must retain top-level unused network declarations")
	}
	if networks := ProjectNetworks(project); len(networks) != 0 {
		t.Fatalf("unused network still collected: %#v", networks)
	}
}
