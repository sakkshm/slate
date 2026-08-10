package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

const (
	ImageName = "slate-base-runner:latest"
)

func NewDockerClient(socketURL string) (*client.Client, error) {

	if !strings.HasPrefix(socketURL, "unix://") && !strings.HasPrefix(socketURL, "tcp://") {
		socketURL = "unix://" + socketURL
	}
	os.Setenv("DOCKER_HOST", socketURL)

	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	// Config the network bridge
	err = EnsureIsolatedNetwork(context.Background(), dockerClient)
	if err != nil {
		return nil, err
	}

	// Pre Pull image
	_, _, inspectErr := dockerClient.ImageInspectWithRaw(context.Background(), ImageName)
	if inspectErr != nil {
		err = PullDockerImage(context.Background(), dockerClient, ImageName)
		if err != nil {
			return nil, err
		}
	}

	return dockerClient, err
}

func PullDockerImage(ctx context.Context, cli *client.Client, imageName string) error {

	pullCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	reader, err := cli.ImagePull(pullCtx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed pulling image %s: %w", imageName, err)
	}
	defer reader.Close()

	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func StartDockerContainer(ctx context.Context, cli *client.Client, imageName, containerName string, config *container.Config, binds ...string) (string, error) {

	hostConfig := BuildHostConfig(binds...)

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed creating container: %w", err)
	}
	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed starting container %s: %w", resp.ID, err)
	}

	return resp.ID, nil
}

func AttachContainerLogs(ctx context.Context, cli *client.Client, containerID string, logOptions *container.LogsOptions) (io.ReadCloser, error) {
	return cli.ContainerLogs(ctx, containerID, *logOptions)
}

func WaitForContainer(ctx context.Context, cli *client.Client, containerID string) (<-chan container.WaitResponse, <-chan error) {
	return cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
}

func StopDockerContainer(ctx context.Context, cli *client.Client, containerID string, gracePeriodSeconds int) error {
	apiTimeout := time.Duration(gracePeriodSeconds) * time.Second
	stopCtx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	stopOptions := container.StopOptions{
		Timeout: &gracePeriodSeconds,
	}

	err := cli.ContainerStop(stopCtx, containerID, stopOptions)
	if err != nil {
		return fmt.Errorf("failed stopping container %s: %w", containerID, err)
	}

	return nil
}

func RemoveDockerContainer(ctx context.Context, cli *client.Client, containerID string) error {

	removeOptions := container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	err := cli.ContainerRemove(ctx, containerID, removeOptions)
	if err != nil {
		return fmt.Errorf("failed removing container %s: %w", containerID, err)
	}

	return nil
}
