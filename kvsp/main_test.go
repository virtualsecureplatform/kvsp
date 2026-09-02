package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestChrysoberylProfileMatchesAlexandriteMemoryABI(t *testing.T) {
	chrysoberyl, err := getCPUProfile("Chrysoberyl")
	if err != nil {
		t.Fatal(err)
	}
	alexandrite := cpuProfiles["alexandrite"]

	got := []interface{}{chrysoberyl.ROMSize, chrysoberyl.RAMSize, chrysoberyl.PointerWidth, chrysoberyl.StackAlign, chrysoberyl.StackPointerOffset, chrysoberyl.RegCount, chrysoberyl.RegWidth, chrysoberyl.RuntimeName}
	want := []interface{}{alexandrite.ROMSize, alexandrite.RAMSize, alexandrite.PointerWidth, alexandrite.StackAlign, alexandrite.StackPointerOffset, alexandrite.RegCount, alexandrite.RegWidth, alexandrite.RuntimeName}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Chrysoberyl ABI = %#v, want Alexandrite ABI %#v", got, want)
	}
	if chrysoberyl.BlueprintName != "IYOKAN-BLUEPRINT-CHRYSOBERYL" {
		t.Fatalf("Chrysoberyl blueprint = %q", chrysoberyl.BlueprintName)
	}
}

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
