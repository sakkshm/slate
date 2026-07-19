package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type BuildRequest struct {
	ID         string
	RepoURL    string
	RootDir    string
	InstallCmd string
	BuildCmd   string
	OutDir     string
	Env        []string
	StagingDir string
}

func RunBuild(ctx context.Context, cli *client.Client, req BuildRequest) (string, int64, string, error) {
	if _, _, err := cli.ImageInspectWithRaw(ctx, ImageName); err != nil {
		if pullErr := PullDockerImage(ctx, cli, ImageName); pullErr != nil {
			return "", 0, "", fmt.Errorf("pre-flight base image check failed: %w", pullErr)
		}
	}

	templateCmd := fmt.Sprintf(
		"git clone --depth 1 %s . && cd ./%s",
		req.RepoURL,
		strings.TrimPrefix(req.RootDir, "/"),
	)

	if req.InstallCmd != "" {
		templateCmd += " && " + req.InstallCmd
	}
	if req.BuildCmd != "" {
		templateCmd += " && " + req.BuildCmd
	}
	if req.StagingDir != "" && req.OutDir != "" {
		templateCmd += fmt.Sprintf(" && mkdir -p /staging && cp -r /app/%s/* /staging/", strings.TrimPrefix(req.OutDir, "/"))
	}

	config := &container.Config{
		Image:      ImageName,
		Cmd:        []string{"sh", "-c", templateCmd},
		Env:        req.Env,
		WorkingDir: "/app",
		User:       "root",
	}

	containerName := fmt.Sprintf("%s-%d", req.ID, time.Now().UnixNano())

	var binds []string
	if req.StagingDir != "" {
		stagingTarget := req.StagingDir + ":/staging"
		binds = append(binds, stagingTarget)
	}

	containerID, err := StartDockerContainer(ctx, cli, ImageName, containerName, config, binds...)
	if err != nil {
		return "", 0, "", fmt.Errorf("container allocation error: %w", err)
	}

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	logStream, err := AttachContainerLogs(ctx, cli, containerID, &logOptions)
	if err != nil {
		safeCleanup(cli, containerID)
		return "", 0, "", fmt.Errorf("log stream connection rejected: %w", err)
	}

	var logBuffer bytes.Buffer
	logErrCh := make(chan error, 1)

	go func() {
		defer logStream.Close()
		_, demuxErr := stdcopy.StdCopy(&logBuffer, &logBuffer, logStream)
		if demuxErr == io.EOF {
			demuxErr = nil
		}
		logErrCh <- demuxErr
	}()

	statusCh, errCh := WaitForContainer(ctx, cli, containerID)
	select {
	case err := <-errCh:
		if err != nil {
			safeCleanup(cli, containerID)
			return "", 0, logBuffer.String(), fmt.Errorf("runtime waiting error: %w", err)
		}

	case status := <-statusCh:
		select {
		case <-logErrCh:
		case <-time.After(1 * time.Second):
		}
		return containerID, status.StatusCode, logBuffer.String(), nil

	case <-ctx.Done():
		safeCleanup(cli, containerID)
		return containerID, -1, logBuffer.String() + "\n[SYSTEM ERROR]: Execution context timeout exceeded.", ctx.Err()
	}

	return containerID, -1, logBuffer.String(), errors.New("unexpected runtime end state reached")
}

func safeCleanup(cli *client.Client, containerID string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stopTimeout := 5
	_ = StopDockerContainer(cleanupCtx, cli, containerID, stopTimeout)
	_ = RemoveDockerContainer(cleanupCtx, cli, containerID)
}

// CleanupStagingDir removes extracted build artifacts from the host staging directory.
// Uncomment in production to prevent disk space creep across sequential builds.
// When uncommenting, also add these imports: "os" and "path/filepath"
// func CleanupStagingDir(stagingDir, containerID string) error {
// 	target := filepath.Join(stagingDir, containerID)
// 	if err := os.RemoveAll(target); err != nil {
// 		return fmt.Errorf("failed cleaning staging dir %s: %w", target, err)
// 	}
// 	return nil
// }
