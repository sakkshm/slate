package runner

import (
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
	ID string

	RepoURL                 string
	InstallationAccessToken string
	CommitSHA               string

	InstallCmd string
	BuildCmd   string

	RootDir    string
	OutDir     string
	StagingDir string
	WorkDir    string

	Env     []string
	LogSink func(line string)
}

func RunBuild(ctx context.Context, cli *client.Client, req BuildRequest) (string, int64, string, error) {
	if _, _, err := cli.ImageInspectWithRaw(ctx, ImageName); err != nil {
		if pullErr := PullDockerImage(ctx, cli, ImageName); pullErr != nil {
			return "", 0, "", fmt.Errorf("pre-flight base image check failed: %w", pullErr)
		}
	}

	repoURL := req.RepoURL
	if req.InstallationAccessToken != "" {
		repoURL = strings.Replace(repoURL, "https://", "https://x-access-token:"+req.InstallationAccessToken+"@", 1)
	}

	workDir := strings.TrimPrefix(req.RootDir, "/")
	if workDir == "" {
		workDir = "."
	}

	assetDir := strings.TrimPrefix(req.OutDir, "/")
	if req.RootDir != "" && assetDir != "" {
		assetDir = strings.TrimPrefix(req.RootDir, "/") + "/" + assetDir
	}

	var templateCmd string

	if req.CommitSHA != "" {
		templateCmd = fmt.Sprintf(
			"git init . && git remote add origin %s && git fetch --depth 1 origin %s && git checkout FETCH_HEAD && cd ./%s",
			repoURL,
			req.CommitSHA,
			workDir,
		)
	} else {
		templateCmd = fmt.Sprintf(
			"git clone --depth 1 %s . && cd ./%s",
			repoURL,
			workDir,
		)
	}

	if req.InstallCmd != "" {
		templateCmd += " && " + req.InstallCmd
	}
	if req.BuildCmd != "" {
		templateCmd += " && " + req.BuildCmd
	}
	if req.StagingDir != "" && assetDir != "" {
		templateCmd += fmt.Sprintf(" && mkdir -p /staging && cp -r /app/%s/* /staging/", assetDir)
	}

	config := &container.Config{
		Image:      ImageName,
		Cmd:        []string{"sh", "-c", templateCmd},
		Env:        append(req.Env, "npm_config_cache=/app/.npm-cache"),
		WorkingDir: "/app",
		User:       "root",
	}

	containerName := fmt.Sprintf("%s-%d", req.ID, time.Now().UnixNano())

	var binds []string
	if req.StagingDir != "" {
		stagingTarget := req.StagingDir + ":/staging"
		binds = append(binds, stagingTarget)
	}
	if req.WorkDir != "" {
		binds = append(binds, req.WorkDir+":/app")
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

	lw := lineWriter{sink: req.LogSink}
	logErrCh := make(chan error, 1)

	go func() {
		defer logStream.Close()
		_, demuxErr := stdcopy.StdCopy(&lw, &lw, logStream)

		if lw.sink != nil && len(lw.pending) > 0 {
			lw.sink(string(lw.pending))
			lw.pending = lw.pending[:0]
		}

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
			return "", 0, lw.buf.String(), fmt.Errorf("runtime waiting error: %w", err)
		}

	case status := <-statusCh:
		select {
		case <-logErrCh:
		case <-time.After(1 * time.Second):
		}
		return containerID, status.StatusCode, lw.buf.String(), nil

	case <-ctx.Done():
		safeCleanup(cli, containerID)
		return containerID, -1, lw.buf.String() + "\n[SYSTEM ERROR]: Execution context timeout exceeded.", ctx.Err()
	}

	return containerID, -1, lw.buf.String(), errors.New("unexpected runtime end state reached")
}

func safeCleanup(cli *client.Client, containerID string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	stopTimeout := 5
	_ = StopDockerContainer(cleanupCtx, cli, containerID, stopTimeout)
	_ = RemoveDockerContainer(cleanupCtx, cli, containerID)
}
