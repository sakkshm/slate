package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slate-backend/internal/runner"
	"time"
)

func main() {
	fmt.Println("[WORKER]: Initializing slate sandbox execution environment...")

	socketURL := "unix:///var/run/docker.sock"
	cli, err := runner.NewDockerClient(socketURL)
	if err != nil {
		log.Fatalf("[FATAL]: Failed to instantiate Docker engine: %v", err)
	}
	defer cli.Close()

	stagingDir := filepath.Join(os.TempDir(), "slate-staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		log.Fatalf("[FATAL]: Failed allocating host staging zone: %v", err)
	}

	request := runner.BuildRequest{
		ID:         "task_dev_sample_build",
		RepoURL:    "https://github.com/sakkshm/slate-test.git",
		RootDir:    "/",
		InstallCmd: "npm install",
		BuildCmd:   "npm run build",
		OutDir:     "dist",
		Env:        []string{},
		StagingDir: stagingDir,
	}

	fmt.Printf("[WORKER]: Launching build sequence for task execution frame: %s\n", request.ID)

	buildCtx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	containerID, statusCode, logs, err := runner.RunBuild(buildCtx, cli, request)
	cancel()

	if err != nil {
		fmt.Printf("[BUILD EXHAUSTED ERROR]: Pipeline broken: %v\n", err)
		if logs != "" {
			fmt.Printf("--- Final Captured Stdout/Err Stream Buffer ---\n%s\n", logs)
		}
	} else {
		fmt.Printf("[WORKER]: Execution halted. Process structural Exit Status Code: %d\n", statusCode)
		fmt.Printf("--- Build Runtime Logs ---\n%s\n-------------------------\n", logs)

		if statusCode == 0 {
			fmt.Printf("[SUCCESS]: Build artifacts written to staging: %s\n", stagingDir)
		} else {
			fmt.Println("[FAIL]: Build pipeline process broken inside runner sandbox environment.")
		}
	}

	fmt.Println("[WORKER]: Sweeping transient execution container objects...")
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := runner.RemoveDockerContainer(cleanupCtx, cli, containerID); err != nil {
		fmt.Printf("[WARNING]: Structural sanitation sweep failed: %v\n", err)
	}
	cleanupCancel()

	// Uncomment in production to prevent disk space creep across builds
	// if err := runner.CleanupStagingDir(stagingDir, containerID); err != nil {
	// 	fmt.Printf("[WARNING]: Staging cleanup failed: %v\n", err)
	// }

	fmt.Println("[WORKER]: Pipeline execution cycle terminated.")
}