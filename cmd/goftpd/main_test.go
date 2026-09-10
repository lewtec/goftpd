package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/goftpd"
)

func TestParseFlags(t *testing.T) {
	dir := t.TempDir()
	app, err := cmd.Parse[cmd.App[flags]]("--addr", ":9", "-d", dir, "--spa")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := app.Args.Addr.Value(); got != ":9" {
		t.Fatalf("addr got %q, want :9", got)
	}
	if got := app.Args.Dir.Value(); got != dir {
		t.Fatalf("dir got %q, want %q", got, dir)
	}
	if !app.Args.SPA.Value() {
		t.Fatal("spa got false, want true")
	}
}

func TestParseDefaults(t *testing.T) {
	app, err := cmd.Parse[cmd.App[flags]]()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := app.Args.Addr.Value(); got != ":8080" {
		t.Fatalf("addr got %q, want :8080", got)
	}
	if got := app.Args.Dir.Value(); got != "./" {
		t.Fatalf("dir got %q, want ./", got)
	}
	if app.Args.SPA.Value() {
		t.Fatal("spa got true, want false")
	}
}

func TestParseUnknownFlag(t *testing.T) {
	_, err := cmd.Parse[cmd.App[flags]]("--nope")
	if !errors.Is(err, cmd.ErrUnknownFlag) {
		t.Fatalf("got %v, want ErrUnknownFlag", err)
	}
}

func TestUsageListsProductFlags(t *testing.T) {
	text, err := cmd.Usage[cmd.App[flags]]("goftpd")
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	help := goftpd.NewCLICopy()
	for _, want := range []string{
		help.Short,
		"--addr",
		help.Addr + " (default: :8080)",
		"-d, --dir",
		help.Dir + " (default: ./)",
		"--spa",
		help.SPA,
		"--version",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("usage missing %q:\n%s", want, text)
		}
	}
}

func TestRunMissingDir(t *testing.T) {
	var f flags
	if err := f.Dir.Parse(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatal(err)
	}
	err := f.Run(t.Context())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Run missing dir: got %v, want os.ErrNotExist", err)
	}
}
