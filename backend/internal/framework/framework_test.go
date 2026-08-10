package framework

import (
	"errors"
	"testing"
)

func TestDetectFromPackageJSON(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected string
	}{
		{"vite", `{"dependencies":{"vite":"^5.0.0"}}`, "vite"},
		{"next in dev deps", `{"devDependencies":{"next":"14.0.0"}}`, "next"},
		{"react-scripts", `{"dependencies":{"react-scripts":"5.0.0"}}`, "react"},
		{"sveltekit", `{"dependencies":{"@sveltejs/kit":"1.0.0"}}`, "sveltekit"},
		{"no framework falls back to static", `{"name":"plain-site"}`, "static"},
		{"invalid json returns empty", `not json`, ""},
		{"empty deps falls back to static", `{}`, "static"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := DetectFromPackageJSON([]byte(tc.content))
			if got != tc.expected {
				t.Fatalf("detected %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestDetectFromRepoAtRoot(t *testing.T) {
	fetch := func(path string) ([]byte, error) {
		if path == "package.json" {
			return []byte(`{"dependencies":{"next":"14.0.0"}}`), nil
		}
		return nil, errors.New("not found")
	}
	list := func(path string) ([]string, error) {
		if path == "" {
			return []string{"docs", "src"}, nil
		}
		return nil, errors.New("not found")
	}

	fw, appDir := DetectFromRepo(fetch, list, "")
	if fw != "next" {
		t.Fatalf("expected next, got %q", fw)
	}
	if appDir != "" {
		t.Fatalf("expected root app dir, got %q", appDir)
	}
}

func TestDetectFromRepoMonorepo(t *testing.T) {
	fetch := func(path string) ([]byte, error) {
		switch path {
		case "package.json":
			return nil, errors.New("not found")
		case "docs/package.json", "src/package.json", "backend/package.json":
			return nil, errors.New("not found")
		case "web/package.json":
			return []byte(`{"dependencies":{"vite":"^5.0.0"}}`), nil
		}
		return nil, errors.New("not found")
	}
	list := func(path string) ([]string, error) {
		if path == "" {
			return []string{"backend", "docs", "web", "src"}, nil
		}
		return nil, errors.New("not found")
	}

	fw, appDir := DetectFromRepo(fetch, list, "")
	if fw != "vite" {
		t.Fatalf("expected vite, got %q", fw)
	}
	if appDir != "web" {
		t.Fatalf("expected web subdir, got %q", appDir)
	}
}

func TestDetectFromRepoPrefersKnownSubdir(t *testing.T) {
	fetch := func(path string) ([]byte, error) {
		switch path {
		case "package.json":
			return nil, errors.New("not found")
		case "a/package.json":
			return []byte(`{"dependencies":{"vite":"^5.0.0"}}`), nil
		case "frontend/package.json":
			return []byte(`{"dependencies":{"react-scripts":"5.0.0"}}`), nil
		}
		return nil, errors.New("not found")
	}
	list := func(path string) ([]string, error) {
		if path == "" {
			return []string{"a", "frontend"}, nil
		}
		return nil, errors.New("not found")
	}

	fw, appDir := DetectFromRepo(fetch, list, "")
	if fw != "react" {
		t.Fatalf("expected react (preferred subdir wins), got %q", fw)
	}
	if appDir != "frontend" {
		t.Fatalf("expected frontend app dir, got %q", appDir)
	}
}

func TestDetectFromRepoRespectsRootDir(t *testing.T) {
	fetch := func(path string) ([]byte, error) {
		if path == "packages/app/package.json" {
			return []byte(`{"dependencies":{"astro":"^4.0.0"}}`), nil
		}
		return nil, errors.New("not found")
	}
	list := func(path string) ([]string, error) {
		return nil, errors.New("not found")
	}

	fw, appDir := DetectFromRepo(fetch, list, "packages/app")
	if fw != "astro" {
		t.Fatalf("expected astro, got %q", fw)
	}
	if appDir != "packages/app" {
		t.Fatalf("expected packages/app, got %q", appDir)
	}
}

func TestDetectFromRepoNothingFound(t *testing.T) {
	fetch := func(path string) ([]byte, error) {
		return nil, errors.New("not found")
	}
	list := func(path string) ([]string, error) {
		return []string{"docs"}, nil
	}

	fw, appDir := DetectFromRepo(fetch, list, "")
	if fw != "" || appDir != "" {
		t.Fatalf("expected empty result, got fw=%q appDir=%q", fw, appDir)
	}
}

func TestResolve(t *testing.T) {
	cfg := Resolve("vite", "", "", "")
	if cfg.InstallCmd != "npm install" || cfg.BuildCmd != "npm run build" || cfg.OutDir != "dist" {
		t.Fatalf("vite defaults wrong: %+v", cfg)
	}

	cfg = Resolve("static", "", "", "")
	if cfg.OutDir != "." {
		t.Fatalf("static defaults wrong: %+v", cfg)
	}

	cfg = Resolve("unknown-framework", "", "", "")
	if cfg.OutDir != "." {
		t.Fatalf("unknown framework should fall back to static: %+v", cfg)
	}

	cfg = Resolve("vite", "yarn", "yarn build", "public")
	if cfg.InstallCmd != "yarn" || cfg.BuildCmd != "yarn build" || cfg.OutDir != "public" {
		t.Fatalf("overrides not respected: %+v", cfg)
	}
}
