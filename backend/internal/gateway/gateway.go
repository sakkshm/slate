package gateway

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"slate-backend/internal/project"
	"slate-backend/internal/queue"
	"slate-backend/internal/storage"
	"slate-backend/pkg/config"
	"slate-backend/pkg/utils"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var CacheRoot = "/tmp/slate-deploy"

var (
	CacheTTL      = 24 * time.Hour
	pruneInterval = 30 * time.Minute
)

type Store interface {
	GetDeployment(ctx context.Context, slug string) (*queue.DeployEntry, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
}

type Resolver struct {
	DB    *gorm.DB
	Redis *redis.Client
	MinIO storage.Store
}

func (r *Resolver) GetDeployment(ctx context.Context, slug string) (*queue.DeployEntry, error) {
	entry, err := queue.GetDeployment(ctx, r.Redis, slug)
	if err != nil {
		return nil, err
	}
	if entry != nil {
		return entry, nil
	}

	proj, err := project.GetProjectBySlug(slug, r.DB, ctx)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, nil
	}

	latest, err := project.GetLatestReadyBuild(proj.ID, r.DB, ctx)
	if err != nil {
		return nil, err
	}
	if latest == nil || latest.AssetLocation == "" {
		return nil, nil
	}

	entry = &queue.DeployEntry{
		ProjectID: proj.ID.String(),
		AssetHash: latest.AssetLocation,
	}
	_ = queue.PublishDeployment(ctx, r.Redis, slug, *entry)
	return entry, nil
}

func (r *Resolver) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return r.MinIO.Download(ctx, key)
}

func DeploymentURL(cfg *config.Config, slug string) string {
	host := slug + "." + cfg.SiteBaseDomain
	if cfg.SitePort != "" {
		host = net.JoinHostPort(host, cfg.SitePort)
	}
	return cfg.SiteScheme + "://" + host
}

func IsDeploymentHost(host, baseDomain string, reserved ...string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" || baseDomain == "" {
		return "", false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	suffix := "." + baseDomain
	if !strings.HasSuffix(host, suffix) {
		return "", false
	}
	slug := strings.TrimSuffix(host, suffix)
	if slug == "" || !utils.ValidSlug(slug) {
		return "", false
	}
	for _, r := range reserved {
		if strings.EqualFold(slug, r) {
			return "", false
		}
	}
	return slug, true
}

func New(store Store, cfg *config.Config) http.Handler {
	startPruner()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug, ok := IsDeploymentHost(r.Host, cfg.SiteBaseDomain, cfg.ReservedHosts...)
		if !ok {
			http.NotFound(w, r)
			return
		}

		entry, err := store.GetDeployment(r.Context(), slug)
		if err != nil {
			http.Error(w, "Failed to resolve deployment", http.StatusBadGateway)
			return
		}
		if entry == nil {
			http.NotFound(w, r)
			return
		}

		root, err := ensureArtifact(r.Context(), store, entry)
		if err != nil {
			http.Error(w, "Failed to load deployment", http.StatusBadGateway)
			return
		}

		serveDeployment(w, r, root)
	})
}

func ensureArtifact(ctx context.Context, store Store, entry *queue.DeployEntry) (string, error) {
	root := filepath.Join(CacheRoot, entry.ProjectID, entry.AssetHash)
	if _, err := os.Stat(filepath.Join(root, ".slate-ready")); err == nil {
		return root, nil
	}

	key := fmt.Sprintf("projects/%s/builds/%s.tar.gz", entry.ProjectID, entry.AssetHash)
	rc, err := store.Download(ctx, key)
	if err != nil {
		return "", fmt.Errorf("download artifact: %w", err)
	}
	defer rc.Close()

	projectDir := filepath.Join(CacheRoot, entry.ProjectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", err
	}
	tmpDir, err := os.MkdirTemp(projectDir, ".extract-*")
	if err != nil {
		return "", err
	}

	if err := extractTarGz(rc, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".slate-ready"), []byte("ready"), 0o644); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err := os.Rename(tmpDir, root); err != nil {
		os.RemoveAll(tmpDir)
		if _, statErr := os.Stat(filepath.Join(root, ".slate-ready")); statErr != nil {
			return "", err
		}
	}
	return root, nil
}

func extractTarGz(src io.Reader, destDir string) error {
	gzReader, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := path.Clean(hdr.Name)
		if name == "." || name == "" {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(name))
		if !isWithin(destDir, target) {
			return fmt.Errorf("unsafe path in archive: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&os.ModePerm)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink, tar.TypeLink:
			continue
		}
	}
	return nil
}

func serveDeployment(w http.ResponseWriter, r *http.Request, root string) {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel == "" || rel == "." {
		rel = "index.html"
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	if !isWithin(root, target) {
		http.NotFound(w, r)
		return
	}
	if fi, err := os.Stat(target); err == nil && fi.IsDir() {
		target = filepath.Join(target, "index.html")
	}
	f, err := os.Open(target)
	if err != nil {
		target = filepath.Join(root, "index.html")
		f, err = os.Open(target)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, filepath.Base(target), stat.ModTime(), f)
}

func isWithin(root, target string) bool {
	rootClean := filepath.Clean(root)
	targetClean := filepath.Clean(target)
	if targetClean == rootClean {
		return true
	}
	return strings.HasPrefix(targetClean, rootClean+string(os.PathSeparator))
}

var (
	prunerOnce sync.Once
	stopOnce   sync.Once
	prunerStop chan struct{}
)

func startPruner() {
	prunerOnce.Do(func() {
		prunerStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(pruneInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := Prune(CacheRoot, CacheTTL); err != nil {
						slog.Error("cache prune error", "error", err)
					}
				case <-prunerStop:
					return
				}
			}
		}()
	})
}

func StopPruner() {
	if prunerStop != nil {
		stopOnce.Do(func() { close(prunerStop) })
	}
}

func Prune(root string, ttl time.Duration) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	for _, projDir := range entries {
		if !projDir.IsDir() {
			continue
		}
		projPath := filepath.Join(root, projDir.Name())
		hashes, err := os.ReadDir(projPath)
		if err != nil {
			continue
		}
		remaining := 0
		for _, hashDir := range hashes {
			hashPath := filepath.Join(projPath, hashDir.Name())
			if !hashDir.IsDir() {
				continue
			}
			stale, err := isStale(hashPath, now, ttl)
			if err != nil || !stale {
				if err == nil {
					remaining++
				}
				continue
			}
			if err := os.RemoveAll(hashPath); err != nil {
				slog.Error("failed to remove stale cache dir", "path", hashPath, "error", err)
				remaining++
			}
		}
		if remaining == 0 {
			_ = os.RemoveAll(projPath)
		}
	}
	return nil
}

func isStale(dir string, now time.Time, ttl time.Duration) (bool, error) {
	// Abandoned extraction temp dirs have no marker file; base staleness on dir mtime.
	marker := filepath.Join(dir, ".slate-ready")
	info, err := os.Stat(marker)
	if err != nil {
		if os.IsNotExist(err) {
			info, err = os.Stat(dir)
		}
		if err != nil {
			return false, err
		}
	}
	return now.Sub(info.ModTime()) > ttl, nil
}
