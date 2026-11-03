package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Opened from a Start menu shortcut the current folder is the install folder: no team is created there.
func TestStartupDir(t *testing.T) {
	plain, repo, proj := t.TempDir(), t.TempDir(), t.TempDir()
	_ = os.Mkdir(filepath.Join(repo, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(proj, "tao.json"), []byte("{}"), 0o644)
	for _, c := range []struct{ last, cwd, want string }{
		{"", plain, ""},
		{"", repo, repo},
		{"", proj, proj},
		{"", "", ""},
		{proj, plain, proj},
	} {
		if got := startupDir(c.last, c.cwd); got != c.want {
			t.Errorf("startupDir(%q, %q) = %q, want %q", c.last, c.cwd, got, c.want)
		}
	}
}

// With no project open every call the window can make answers without a panic and writes nothing.
func TestNoProjectIsSafe(t *testing.T) {
	cwd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer os.Chdir(cwd)
	a := NewApp()
	v := reflect.ValueOf(a)
	for i := 0; i < v.NumMethod(); i++ {
		m := v.Type().Method(i)
		switch m.Name {
		case "ServiceStartup", "ServiceShutdown", "ChooseProject", "DetectClients", "SetLanguage", "TestClient":
			continue // need the window, touch the settings file or start real processes
		}
		var args []reflect.Value
		for j := 1; j < m.Type.NumIn(); j++ {
			args = append(args, reflect.Zero(m.Type.In(j)))
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panics with no project: %v", m.Name, r)
				}
			}()
			v.Method(i).Call(args)
		}()
	}
	if files, _ := os.ReadDir(tmp); len(files) > 0 {
		t.Errorf("calls with no project wrote %v", files)
	}
}
