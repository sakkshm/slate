package runner

import (
	"context"
	"fmt"
	"strings"

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

	tmpfs := map[string]string{
		"/tmp":  "rw,noexec,nosuid,size=512m",
		"/root": "rw,noexec,nosuid,size=512m",
	}

	appBound := false
	for _, b := range binds {
		if strings.HasSuffix(b, ":/app") {
			appBound = true
			break
		}
	}
	if !appBound {
		tmpfs["/app"] = "rw,exec,nosuid,size=2g"
	}

	return &container.HostConfig{
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		Init:           &isInitEnabled,
		Tmpfs:          tmpfs,
		NetworkMode:    container.NetworkMode(NetworkName),
		Binds:          binds,
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
