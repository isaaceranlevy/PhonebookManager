package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHelpAndUnknown(t *testing.T) {
	if e := execute([]string{"help"}); e != nil {
		t.Fatal(e)
	}
	if e := execute([]string{"missing"}); e == nil {
		t.Fatal("unknown command")
	}
}
func TestUnsupportedLifecycleIgnoresBrokenConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires real SCM; manual acceptance")
	}
	for _, command := range []string{"status", "uninstall"} {
		d := t.TempDir()
		os.WriteFile(filepath.Join(d, "config.json"), []byte("broken"), 0600)
		e := execute([]string{command, "--data-dir", d})
		if e == nil || !strings.Contains(e.Error(), "requires Windows") {
			t.Fatalf("%s: %v", command, e)
		}
	}
}
func TestInstallNonWindowsDoesNotCreateData(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires real SCM")
	}
	d := filepath.Join(t.TempDir(), "not-created")
	execute([]string{"install", "--data-dir", d, "--subnet", "192.168.1.0/24"})
	if _, e := os.Stat(d); !os.IsNotExist(e) {
		t.Fatal("unsupported install created data")
	}
}
