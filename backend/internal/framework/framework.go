package framework

import (
	"encoding/json"
	"sort"
	"strings"
)

type Config struct {
	InstallCmd string
	BuildCmd   string
	OutDir     string
}

var defaults = map[string]Config{
	"next":      {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "out"},
	"vite":      {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "dist"},
	"react":     {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "build"},
	"astro":     {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "dist"},
	"gatsby":    {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "public"},
	"nuxt":      {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: ".output/public"},
	"sveltekit": {InstallCmd: "npm install", BuildCmd: "npm run build", OutDir: "build"},
	"static":    {InstallCmd: "", BuildCmd: "", OutDir: "."},
}

var detectionOrder = []struct {
	framework string
	marker    string
}{
	{"next", "next"},
	{"gatsby", "gatsby"},
	{"nuxt", "nuxt"},
	{"sveltekit", "@sveltejs/kit"},
	{"astro", "astro"},
	{"vite", "vite"},
	{"react", "react-scripts"},
}

// preferredSubdirs orders candidate app directories when scanning a monorepo.
var preferredSubdirs = []string{"frontend", "web", "app", "client", "ui", "dashboard", "site"}

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func DetectFromPackageJSON(content []byte) (string, Config) {
	var pkg PackageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return "", Config{}
	}

	for _, d := range detectionOrder {
		if _, ok := pkg.Dependencies[d.marker]; ok {
			return d.framework, defaults[d.framework]
		}
		if _, ok := pkg.DevDependencies[d.marker]; ok {
			return d.framework, defaults[d.framework]
		}
	}
	return "static", defaults["static"]
}

func detectNameFromPkg(pkg *PackageJSON) string {
	for _, d := range detectionOrder {
		if _, ok := pkg.Dependencies[d.marker]; ok {
			return d.framework
		}
		if _, ok := pkg.DevDependencies[d.marker]; ok {
			return d.framework
		}
	}
	return ""
}

func detectName(content []byte) string {
	var pkg PackageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return ""
	}
	return detectNameFromPkg(&pkg)
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}

func prioritizeSubdirs(dirs []string) []string {
	rank := func(d string) int {
		for i, p := range preferredSubdirs {
			if d == p {
				return i
			}
		}
		return len(preferredSubdirs)
	}
	sort.SliceStable(dirs, func(i, j int) bool {
		ri, rj := rank(dirs[i]), rank(dirs[j])
		if ri != rj {
			return ri < rj
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

// DetectFromRepo locates a deployable web app within a repository. It first
// checks the package.json at basePath, then scans subdirectories (preferring
// common app directory names) for a package.json declaring a known framework.
// It returns the detected framework and the app directory relative to the repo
// root ("" when the app lives at the root), or ("", "") when nothing is found.
func DetectFromRepo(
	fetchPackage func(path string) ([]byte, error),
	listDirs func(path string) ([]string, error),
	basePath string,
) (string, string) {
	basePath = strings.Trim(basePath, "/")

	if content, err := fetchPackage(joinPath(basePath, "package.json")); err == nil {
		if fw := detectName(content); fw != "" {
			return fw, basePath
		}
	}

	dirs, err := listDirs(basePath)
	if err != nil {
		return "", ""
	}
	for _, dir := range prioritizeSubdirs(dirs) {
		content, err := fetchPackage(joinPath(joinPath(basePath, dir), "package.json"))
		if err != nil {
			continue
		}
		if fw := detectName(content); fw != "" {
			return fw, joinPath(basePath, dir)
		}
	}
	return "", ""
}

func Resolve(framework, installCmd, buildCmd, outDir string) Config {
	cfg, ok := defaults[framework]
	if !ok {
		cfg = defaults["static"]
	}
	if installCmd != "" {
		cfg.InstallCmd = installCmd
	}
	if buildCmd != "" {
		cfg.BuildCmd = buildCmd
	}
	if outDir != "" {
		cfg.OutDir = outDir
	}
	return cfg
}
