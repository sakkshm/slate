package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"slate-backend/internal/build"
	"slate-backend/internal/clients"
	githubclient "slate-backend/internal/github"
	"slate-backend/internal/queue"
	"slate-backend/internal/runner"
	"slate-backend/pkg/config"
	"slate-backend/pkg/types"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func buildLogSink(cli *redis.Client, buildID string) func(line string) {
	return func(line string) {

		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()

		if err := queue.WriteLogLine(logCtx, cli, buildID, line); err != nil {
			fmt.Printf("[WORKER] log store error: %v\n", err)
		}

		if err := queue.PublishLogLine(logCtx, cli, buildID, line); err != nil {
			fmt.Printf("[WORKER] log publish error: %v\n", err)
		}
	}
}

func archiveBuildLogs(ctx context.Context, c *clients.Clients, buildID uuid.UUID) {
	lines, err := queue.GetLogLines(ctx, c.Redis, buildID.String())
	if err != nil {
		fmt.Printf("[WORKER] Failed to read logs for archival: %v\n", err)
	} else if len(lines) > 0 {
		if err := build.UpdateBuildLogContent(c.DB, buildID, strings.Join(lines, "\n"), ctx); err != nil {
			fmt.Printf("[WORKER] Failed to archive build logs: %v\n", err)
		}
	}

	if err := queue.PublishBuildDone(ctx, c.Redis, buildID.String()); err != nil {
		fmt.Printf("[WORKER] Failed to publish build done: %v\n", err)
	}
}

func reportCommitStatus(cfg *config.Config, event *types.BuildEvent, state, description string) {
	parts := strings.SplitN(event.RepoName, "/", 2)
	if len(parts) != 2 {
		fmt.Printf("[WORKER] Invalid repo name %q for commit status\n", event.RepoName)
		return
	}

	targetURL := fmt.Sprintf("%s/projects/%s/builds/%s", cfg.AppURL, event.ProjectID, event.BuildID)

	statusCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := githubclient.CreateCommitStatus(
		event.InstallationAccessToken,
		parts[0],
		parts[1],
		event.CommitSHA,
		state,
		description,
		"slate",
		targetURL,
		statusCtx,
	); err != nil {
		fmt.Printf("[WORKER] Commit status %s for build %s failed: %v\n", state, event.BuildID, err)
	}
}

func main() {
	cfg := config.LoadConfig()

	c, err := clients.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize clients: %v", err)
	}

	dockerClient, err := runner.NewDockerClient(cfg.DockerSocketPath)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	stagingRoot := filepath.Join(os.TempDir(), "slate-staging")
	if err := os.MkdirAll(stagingRoot, 0755); err != nil {
		log.Fatalf("Failed to create staging dir: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		fmt.Println("[WORKER] Shutting down...")
		cancel()
	}()

	fmt.Println("[WORKER] Ready. Listening for build jobs...")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[WORKER] Stopped.")
			return
		default:
		}

		event, msgID, err := queue.ClaimBuildRequest(ctx, c.Redis, "worker-1")
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			fmt.Printf("[WORKER] Error claiming build: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if event == nil {
			continue
		}

		buildID, err := uuid.Parse(event.BuildID)
		if err != nil {
			fmt.Printf("[WORKER] Invalid build ID %s: %v\n", event.BuildID, err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		projectID, err := uuid.Parse(event.ProjectID)
		if err != nil {
			fmt.Printf("[WORKER] Invalid project ID %s: %v\n", event.ProjectID, err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		claimed, err := build.UpdateBuildStatusIfQueued(c.DB, buildID, ctx)
		if err != nil {
			fmt.Printf("[WORKER] Failed to update build status: %v\n", err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}
		if !claimed {
			fmt.Printf("[WORKER] Build %s was cancelled while queued, skipping\n", buildID)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}
		reportCommitStatus(cfg, event, "pending", "Build started")

		startTime := time.Now()
		stagingDir := filepath.Join(stagingRoot, buildID.String())
		os.MkdirAll(stagingDir, 0755)

		buildReq := runner.BuildRequest{
			ID:                      buildID.String(),
			RepoURL:                 event.RepoURL,
			InstallationAccessToken: event.InstallationAccessToken,
			CommitSHA:               event.CommitSHA,
			RootDir:                 event.RootDir,
			InstallCmd:              event.InstallCmd,
			BuildCmd:                event.BuildCmd,
			OutDir:                  event.OutDir,
			Env:                     event.Env,
			StagingDir:              stagingDir,
			LogSink:                 buildLogSink(c.Redis, buildID.String()),
		}

		buildTimeout := time.Duration(cfg.BuildTimeout) * time.Second
		buildCtx, buildCancel := context.WithTimeout(ctx, buildTimeout)

		cancelSub := c.Redis.Subscribe(ctx, queue.CancelChannelKey(buildID.String()))
		go func() {
			for {
				if _, err := cancelSub.ReceiveMessage(ctx); err != nil {
					return
				}
				buildCancel()
				return
			}
		}()

		containerID, statusCode, _, err := runner.RunBuild(buildCtx, dockerClient, buildReq)
		buildCancel()
		cancelSub.Close()

		duration := time.Since(startTime).Milliseconds()

		if errors.Is(err, context.Canceled) {
			fmt.Printf("[WORKER] Build %s cancelled\n", buildID)
			build.UpdateBuildStatus(c.DB, buildID, "cancelled", ctx)
			build.UpdateBuildDuration(c.DB, buildID, duration, ctx)
			cleanupContainer(ctx, dockerClient, containerID)
			os.RemoveAll(stagingDir)
			archiveBuildLogs(ctx, c, buildID)
			reportCommitStatus(cfg, event, "failure", "Build cancelled")
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		if err != nil || statusCode != 0 {
			fmt.Printf("[WORKER] Build %s failed (exit %d): %v\n", buildID, statusCode, err)
			build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
			build.UpdateBuildDuration(c.DB, buildID, duration, ctx)
			cleanupContainer(ctx, dockerClient, containerID)
			os.RemoveAll(stagingDir)
			archiveBuildLogs(ctx, c, buildID)
			reportCommitStatus(cfg, event, "failure", "Build failed")
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		fmt.Printf("[WORKER] Build %s completed. Uploading artifacts...\n", buildID)

		hash, tempFile, err := createArtifactArchive(stagingDir)
		if err != nil {
			fmt.Printf("[WORKER] Failed to create artifact archive: %v\n", err)
			build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
			cleanupContainer(ctx, dockerClient, containerID)
			os.RemoveAll(stagingDir)
			archiveBuildLogs(ctx, c, buildID)
			reportCommitStatus(cfg, event, "failure", "Build failed")
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		assetKey := fmt.Sprintf("projects/%s/builds/%s.tar.gz", projectID, hash)

		exists, err := c.MinIO.Exists(ctx, assetKey)
		if err != nil {
			fmt.Printf("[WORKER] Failed to check artifact existence: %v\n", err)
			os.Remove(tempFile)
			build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
			cleanupContainer(ctx, dockerClient, containerID)
			os.RemoveAll(stagingDir)
			archiveBuildLogs(ctx, c, buildID)
			reportCommitStatus(cfg, event, "failure", "Build failed")
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		if !exists {
			f, err := os.Open(tempFile)
			if err != nil {
				fmt.Printf("[WORKER] Failed to open temp file: %v\n", err)
				os.Remove(tempFile)
				build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
				cleanupContainer(ctx, dockerClient, containerID)
				os.RemoveAll(stagingDir)
				archiveBuildLogs(ctx, c, buildID)
				reportCommitStatus(cfg, event, "failure", "Build failed")
				queue.AckBuild(ctx, c.Redis, msgID)
				continue
			}

			stat, _ := f.Stat()
			if err := c.MinIO.Upload(ctx, assetKey, f, stat.Size()); err != nil {
				fmt.Printf("[WORKER] Failed to upload artifacts: %v\n", err)
				f.Close()
				os.Remove(tempFile)
				build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
				cleanupContainer(ctx, dockerClient, containerID)
				os.RemoveAll(stagingDir)
				archiveBuildLogs(ctx, c, buildID)
				reportCommitStatus(cfg, event, "failure", "Build failed")
				queue.AckBuild(ctx, c.Redis, msgID)
				continue
			}
			f.Close()
		}

		os.Remove(tempFile)

		if err := build.UpdateBuildAssetLocation(c.DB, buildID, hash, ctx); err != nil {
			fmt.Printf("[WORKER] Failed to update asset location: %v\n", err)
		}

		build.UpdateBuildDuration(c.DB, buildID, duration, ctx)

		if err := build.UpdateBuildStatus(c.DB, buildID, "ready", ctx); err != nil {
			fmt.Printf("[WORKER] Failed to update build status: %v\n", err)
		}

		fmt.Printf("[WORKER] Build %s deployed. Hash: %s\n", buildID, hash)

		cleanupContainer(ctx, dockerClient, containerID)
		os.RemoveAll(stagingDir)
		archiveBuildLogs(ctx, c, buildID)
		reportCommitStatus(cfg, event, "success", "Build succeeded")
		queue.AckBuild(ctx, c.Redis, msgID)
	}
}

func cleanupContainer(ctx context.Context, dockerClient *client.Client, containerID string) {
	if containerID == "" {
		return
	}
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	runner.RemoveDockerContainer(cleanupCtx, dockerClient, containerID)
}

func createArtifactArchive(sourceDir string) (hash string, tempPath string, err error) {
	tmpFile, err := os.CreateTemp("", "slate-artifact-*.tar.gz")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath = tmpFile.Name()

	h := sha256.New()
	writer := io.MultiWriter(tmpFile, h)

	gzWriter := gzip.NewWriter(writer)
	tarWriter := tar.NewWriter(gzWriter)

	if err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}

		return nil
	}); err != nil {
		tmpFile.Close()
		os.Remove(tempPath)
		return "", "", fmt.Errorf("failed to walk source dir: %w", err)
	}

	if err := tarWriter.Close(); err != nil {
		tmpFile.Close()
		os.Remove(tempPath)
		return "", "", fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		tmpFile.Close()
		os.Remove(tempPath)
		return "", "", fmt.Errorf("failed to close gzip writer: %w", err)
	}
	tmpFile.Close()

	hash = fmt.Sprintf("%x", h.Sum(nil))
	return hash, tempPath, nil
}
