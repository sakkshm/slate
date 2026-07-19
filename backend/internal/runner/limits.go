package runner

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	memoryBytes = int64(4 * 1024 * 1024 * 1024)
	cpuCores    = int64(2_000_000_000)
	NetworkName = "ci_isolated_sandbox"
)

func EnsureIsolatedNetwork(ctx context.Context, cli *client.Client) error {
	_, err := cli.NetworkInspect(ctx, NetworkName, types.NetworkInspectOptions{})
	if err == nil {
		return nil
	}

	networkConfig := types.NetworkCreate{
		Driver: "bridge",
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "false",
			"com.docker.network.bridge.name":       "br-ci-sandbox",
		},
	}

	_, err = cli.NetworkCreate(ctx, NetworkName, networkConfig)
	if err != nil {
		return fmt.Errorf("failed creating secure network: %w", err)
	}

	return nil
}

func BuildHostConfig(binds ...string) *container.HostConfig {
	isInitEnabled := true

	return &container.HostConfig{
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		Init:           &isInitEnabled,
		Tmpfs: map[string]string{
			"/tmp":  "rw,noexec,nosuid,size=512m",
			"/root": "rw,noexec,nosuid,size=512m",
			"/app":  "rw,exec,nosuid,size=2g",
		},
		NetworkMode: container.NetworkMode(NetworkName),
		Binds:       binds,
		Resources: container.Resources{
			Memory:         memoryBytes,
			NanoCPUs:       cpuCores,
			PidsLimit:      pointerToInt64(100),
			OomKillDisable: pointerToBool(false),
		},
	}
}

func pointerToBool(b bool) *bool    { return &b }
func pointerToInt64(i int64) *int64 { return &i }