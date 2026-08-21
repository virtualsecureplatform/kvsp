package main

import (
	"flag"
	"testing"
)

func TestBackendFlagDefaultsToTangor(t *testing.T) {
	fs := flag.NewFlagSet("backend-default", flag.ContinueOnError)
	backend := addBackendFlag(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if *backend != "tangor" {
		t.Fatalf("default backend = %q, want tangor", *backend)
	}
}

func TestSelectBackend(t *testing.T) {
	previous := evaluatorBackend
	t.Cleanup(func() { evaluatorBackend = previous })

	for _, name := range []string{"tangor", "iyokan", "IYOKAN"} {
		if err := selectBackend(name); err != nil {
			t.Fatalf("selectBackend(%q): %v", name, err)
		}
	}
	if evaluatorBackend != "iyokan" {
		t.Fatalf("selected backend = %q, want iyokan", evaluatorBackend)
	}
	if err := selectBackend("unknown"); err == nil {
		t.Fatal("selectBackend accepted an unknown backend")
	}
}
