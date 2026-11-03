package main

import (
	"os"
	"regexp"
	"testing"
)

// The version lives in build/config.yml; every file copied from it (exe resources, installer, packages,
// the status bar) must carry the same one.
func TestVersionsMatch(t *testing.T) {
	read := func(path, pattern string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m := regexp.MustCompile(pattern).FindSubmatch(data)
		if m == nil {
			t.Fatalf("%s: no version (%s)", path, pattern)
		}
		return string(m[1])
	}
	want := read("build/config.yml", `(?m)^\s*version:\s*"([\d.]+)"`)
	for path, pattern := range map[string]string{
		"build/windows/info.json":            `"file_version":\s*"([\d.]+)"`,
		"build/windows/nsis/wails_tools.nsh": `INFO_PRODUCTVERSION "([\d.]+)"`,
		"build/darwin/Info.plist":            `CFBundleShortVersionString</key>\s*<string>([\d.]+)<`,
		"build/linux/nfpm/nfpm.yaml":         `(?m)^version:\s*"([\d.]+)"`,
		"frontend/src/main.js":               `VERSION = 'v([\d.]+)'`,
	} {
		if got := read(path, pattern); got != want {
			t.Errorf("%s has version %s, build/config.yml has %s", path, got, want)
		}
	}
}
