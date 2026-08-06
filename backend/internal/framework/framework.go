package framework

import (
	"encoding/json"
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
