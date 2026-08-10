package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slate-backend/internal/build"
	"slate-backend/internal/clients"
	githubclient "slate-backend/internal/github"
	"slate-backend/internal/logging"
	"slate-backend/internal/prune"
	"slate-backend/internal/queue"
	"slate-backend/internal/runner"
	"slate-backend/pkg/config"
	"slate-backend/pkg/types"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var logger *slog.Logger

func buildLogSink(cli *redis.Client, buildID string) func(line string) {
	return func(line string) {

		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()

		if err := queue.WriteLogLine(logCtx, cli, buildID, line); err != nil {
			logger.Error("log store error", "error", err)
		}

		if err := queue.PublishLogLine(logCtx, cli, buildID, line); err != nil {
			logger.Error("log publish error", "error", err)
		}
	}
}

func archiveBuildLogs(ctx context.Context, c *clients.Clients, buildID uuid.UUID) {
	lines, err := queue.GetLogLines(ctx, c.Redis, buildID.String())
	if err != nil {
		logger.Error("failed to read logs for archival", "build_id", buildID, "error", err)
	} else if len(lines) > 0 {
		if err := build.UpdateBuildLogContent(c.DB, buildID, strings.Join(lines, "\n"), ctx); err != nil {
			logger.Error("failed to archive build logs", "build_id", buildID, "error", err)
		}
	}

	if err := queue.PublishBuildDone(ctx, c.Redis, buildID.String()); err != nil {
		logger.Error("failed to publish build done", "build_id", buildID, "error", err)
	}
}

func reportCommitStatus(cfg *config.Config, event *types.BuildEvent, state, description string) {
	parts := strings.SplitN(event.RepoName, "/", 2)
	if len(parts) != 2 {
		logger.Warn("invalid repo name for commit status", "repo", event.RepoName)
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
		logger.Error("commit status update failed", "state", state, "build_id", event.BuildID, "error", err)
	}
}

func main() {
	cfg := config.LoadConfig()
	slog.SetDefault(logging.New(cfg.Environment))
	logger = slog.Default().With("component", "worker")

	c, err := clients.New(cfg)
	if err != nil {
		logger.Error("failed to initialize clients", "error", err)
		os.Exit(1)
	}

	dockerClient, err := runner.NewDockerClient(cfg.DockerSocketPath)
	if err != nil {
		logger.Error("failed to create Docker client", "error", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	stagingRoot := filepath.Join(os.TempDir(), "slate-staging")
	if err := os.MkdirAll(stagingRoot, 0755); err != nil {
		logger.Error("failed to create staging dir", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		logger.Info("shutting down worker")
		cancel()
	}()

	startArtifactPruner(ctx, cfg, c)

	logger.Info("worker ready. listening for build jobs")

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return
		default:
		}

		event, msgID, err := queue.ClaimBuildRequest(ctx, c.Redis, "worker-1")
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("error claiming build", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if event == nil {
			continue
		}

		buildID, err := uuid.Parse(event.BuildID)
		if err != nil {
			logger.Error("invalid build ID", "build_id", event.BuildID, "error", err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		projectID, err := uuid.Parse(event.ProjectID)
		if err != nil {
			logger.Error("invalid project ID", "project_id", event.ProjectID, "error", err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		claimed, err := build.UpdateBuildStatusIfQueued(c.DB, buildID, ctx)
		if err != nil {
			logger.Error("failed to update build status", "build_id", buildID, "error", err)
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}
		if !claimed {
			logger.Info("build cancelled while queued, skipping", "build_id", buildID)
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
			logger.Info("build cancelled", "build_id", buildID)
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
			logger.Error("build failed", "build_id", buildID, "exit_code", statusCode, "error", err)
			build.UpdateBuildStatus(c.DB, buildID, "failed", ctx)
			build.UpdateBuildDuration(c.DB, buildID, duration, ctx)
			cleanupContainer(ctx, dockerClient, containerID)
			os.RemoveAll(stagingDir)
			archiveBuildLogs(ctx, c, buildID)
			reportCommitStatus(cfg, event, "failure", "Build failed")
			queue.AckBuild(ctx, c.Redis, msgID)
			continue
		}

		logger.Info("build completed, uploading artifacts", "build_id", buildID)

		hash, tempFile, err := createArtifactArchive(stagingDir)
		if err != nil {
			logger.Error("failed to create artifact archive", "build_id", buildID, "error", err)
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
			logger.Error("failed to check artifact existence", "build_id", buildID, "error", err)
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
				logger.Error("failed to open temp file", "build_id", buildID, "error", err)
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
				logger.Error("failed to upload artifacts", "build_id", buildID, "error", err)
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
			logger.Error("failed to update asset location", "build_id", buildID, "error", err)
		}

		build.UpdateBuildDuration(c.DB, buildID, duration, ctx)

		if err := build.UpdateBuildStatus(c.DB, buildID, "ready", ctx); err != nil {
			logger.Error("failed to update build status", "build_id", buildID, "error", err)
		}

		err = queue.PublishDeployment(ctx, c.Redis, event.Slug, queue.DeployEntry{
			ProjectID: projectID.String(),
			AssetHash: hash,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			logger.Error("failed to publish deployment", "build_id", buildID, "slug", event.Slug, "error", err)
		}

		logger.Info("build deployed", "build_id", buildID, "hash", hash)

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

func startArtifactPruner(ctx context.Context, cfg *config.Config, c *clients.Clients) {
	interval := time.Duration(cfg.PruneIntervalHours) * time.Hour
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -cfg.ArtifactRetentionDays)
				deleted, err := prune.PruneArtifacts(ctx, c.DB, c.MinIO, cutoff)
				if err != nil {
					logger.Error("artifact prune error", "error", err)
				} else if deleted > 0 {
					logger.Info("pruned old artifacts", "count", deleted, "older_than_days", cfg.ArtifactRetentionDays)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
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
