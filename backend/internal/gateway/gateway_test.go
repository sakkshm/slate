package gateway

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"slate-backend/pkg/config"
)

func TestIsDeploymentHost(t *testing.T) {
	base := "slate.example.com"

	cases := []struct {
		name   string
		host   string
		slug   string
		isDepl bool
	}{
		{"valid", "myapp.slate.example.com", "myapp", true},
		{"valid with port", "myapp.slate.example.com:8080", "myapp", true},
		{"no subdomain", "slate.example.com", "", false},
		{"bare domain", "example.com", "", false},
		{"uppercase slug rejected", "MyApp.slate.example.com", "", false},
		{"trailing dot host", "myapp.slate.example.com.", "", false},
		{"invalid slug chars", "my_app.slate.example.com", "", false},
		{"leading dash", "-myapp.slate.example.com", "", false},
		{"empty string", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slug, ok := IsDeploymentHost(tc.host, base)
			if ok != tc.isDepl {
				t.Fatalf("host %q: ok=%v want %v", tc.host, ok, tc.isDepl)
			}
			if ok && slug != tc.slug {
				t.Fatalf("host %q: slug=%q want %q", tc.host, slug, tc.slug)
			}
		})
	}
}

func TestIsDeploymentHostReserved(t *testing.T) {
	reserved := []string{"api", "www", "admin"}
	if _, ok := IsDeploymentHost("api.slate.example.com", "slate.example.com", reserved...); ok {
		t.Fatal("reserved host should not be treated as a deployment")
	}
	if _, ok := IsDeploymentHost("www.slate.example.com", "slate.example.com", reserved...); ok {
		t.Fatal("reserved host should not be treated as a deployment")
	}
	if _, ok := IsDeploymentHost("myapp.slate.example.com", "slate.example.com", reserved...); !ok {
		t.Fatal("non-reserved host should be a deployment")
	}
}

func TestDeploymentURL(t *testing.T) {
	cfg := &config.Config{
		SiteBaseDomain: "slate.example.com",
		SiteScheme:     "https",
	}
	if got := DeploymentURL(cfg, "myapp"); got != "https://myapp.slate.example.com" {
		t.Fatalf("got %q", got)
	}

	cfg.SitePort = "8443"
	if got := DeploymentURL(cfg, "myapp"); got != "https://myapp.slate.example.com:8443" {
		t.Fatalf("got %q", got)
	}
}

func TestIsWithin(t *testing.T) {
	root := "/var/cache/slate"
	if !isWithin(root, "/var/cache/slate/index.html") {
		t.Fatal("nested path should be within root")
	}
	if !isWithin(root, "/var/cache/slate") {
		t.Fatal("root itself should be within root")
	}
	if isWithin(root, "/var/cache/slate2/index.html") {
		t.Fatal("sibling with shared prefix should not be within root")
	}
	if isWithin(root, "/etc/passwd") {
		t.Fatal("outside path should not be within root")
	}
}

func TestExtractTarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	files := map[string]string{
		"index.html":        "<html>hi</html>",
		"assets/app.js":     "console.log('x')",
		"assets/deep/x.css": "body{}",
	}
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()

	dest := t.TempDir()
	if err := extractTarGz(&buf, dest); err != nil {
		t.Fatalf("extract: %v", err)
	}

	for name, content := range files {
		got, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != content {
			t.Fatalf("file %s content mismatch: got %q want %q", name, got, content)
		}
	}
}

func TestExtractTarGzRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	evil := "../evil.txt"
	if err := tw.WriteHeader(&tar.Header{Name: evil, Mode: 0o644, Size: 5}); err != nil {
		t.Fatal(err)
	}
	tw.Write([]byte("evil!"))
	tw.Close()
	gz.Close()

	dest := t.TempDir()
	if err := extractTarGz(&buf, dest); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if _, err := os.Stat(filepath.Join(dest, "..", "evil.txt")); !os.IsNotExist(err) {
		t.Fatal("traversal file should not exist outside dest")
	}
}

func TestPruneRemovesStaleDirs(t *testing.T) {
	root := t.TempDir()

	// fresh ready dir (should survive)
	fresh := filepath.Join(root, "proj1", "hash-fresh")
	mustMkdirReady(t, fresh)

	// stale ready dir (should be removed)
	stale := filepath.Join(root, "proj1", "hash-stale")
	mustMkdirReady(t, stale)
	setMtime(t, stale, time.Now().Add(-48*time.Hour))

	// abandoned temp extraction dir with no marker (should be removed)
	abandoned := filepath.Join(root, "proj2", ".extract-123")
	mustMkdirReady(t, abandoned)
	setMtime(t, abandoned, time.Now().Add(-48*time.Hour))

	if err := Prune(root, 24*time.Hour); err != nil {
		t.Fatalf("prune: %v", err)
	}

	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh dir should survive prune")
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale dir should have been removed")
	}
	if _, err := os.Stat(abandoned); !os.IsNotExist(err) {
		t.Fatal("abandoned dir should have been removed")
	}
}

func TestPruneRemovesEmptyProjectDirs(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "proj-only-stale")
	mustMkdirReady(t, filepath.Join(proj, "hash-stale"))
	setMtime(t, filepath.Join(proj, "hash-stale"), time.Now().Add(-48*time.Hour))

	if err := Prune(root, 24*time.Hour); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(proj); !os.IsNotExist(err) {
		t.Fatal("project dir with only stale hashes should be removed")
	}
}

func TestPruneMissingRoot(t *testing.T) {
	if err := Prune(filepath.Join(t.TempDir(), "does-not-exist"), time.Hour); err != nil {
		t.Fatalf("prune on missing root should be a no-op: %v", err)
	}
}

func mustMkdirReady(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".slate-ready")
	if strings.HasSuffix(dir, "/.extract-123") {
		return
	}
	if err := os.WriteFile(marker, []byte("ready"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setMtime(t *testing.T, dir string, mtime time.Time) {
	t.Helper()
	now := time.Now()
	if err := os.Chtimes(dir, now, mtime); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".slate-ready")
	if _, err := os.Stat(marker); err == nil {
		os.Chtimes(marker, now, mtime)
	}
}
