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
	if !chrysoberyl.InitializationOnlyReset {
		t.Fatal("Chrysoberyl must skip runtime reset evaluation")
	}
}

func TestCAHPProfilesUseFullSizeMemory(t *testing.T) {
	for _, name := range []string{"ruby", "pearl"} {
		profile := cpuProfiles[name]
		if profile.ROMSize != 4*1024 {
			t.Errorf("%s ROM size = %d, want 4096", name, profile.ROMSize)
		}
		if profile.RAMSize != 1024 {
			t.Errorf("%s RAM size = %d, want 1024", name, profile.RAMSize)
		}
		if profile.StackPointerOffset != profile.RAMSize-2 {
			t.Errorf("%s stack pointer slot = %d, want %d", name, profile.StackPointerOffset, profile.RAMSize-2)
		}
	}
}

func TestInitializationOnlyResetArgument(t *testing.T) {
	base := []string{"plain"}
	got := addResetMode(append([]string{}, base...), cpuProfiles["chrysoberyl"])
	want := []string{"plain", "--skip-reset"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Chrysoberyl reset arguments = %#v, want %#v", got, want)
	}

	got = addResetMode(append([]string{}, base...), cpuProfiles["alexandrite"])
	if !reflect.DeepEqual(got, base) {
		t.Fatalf("Alexandrite reset arguments = %#v, want %#v", got, base)
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
