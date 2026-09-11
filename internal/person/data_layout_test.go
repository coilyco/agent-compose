package person

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

// entityDir is the smallest tree dataLayout will project: the order is read
// before anything decodes the rest of the fragment.
func entityDir(kind, slug string, order int) fstest.MapFS {
	fragment := kind + ": " + slug + "\norder: " + strconv.Itoa(order) + "\n"
	return fstest.MapFS{
		dataRoot + "/" + kind + "-" + slug + "/" + kind + yamlFragmentExt: {
			Data: []byte(fragment),
			Mode: 0o644,
		},
	}
}

func mergeFS(parts ...fstest.MapFS) fstest.MapFS {
	out := fstest.MapFS{}
	for _, part := range parts {
		for name, file := range part {
			out[name] = file
		}
	}
	return out
}

func TestDataLayoutRefusesTwoEntitiesClaimingOneOrder(t *testing.T) {
	source := mergeFS(
		entityDir("personality", "alpha", 11),
		entityDir("personality", "beta", 11),
	)
	_, _, err := dataLayout(source, "fixture")
	if err == nil {
		t.Fatal("two personalities claiming order 11 must be refused")
	}
	for _, want := range []string{"alpha", "beta", "order 11"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
}

// The order sequences one kind, so the four kinds each start at 1 in the
// shipped roster and a cross-kind collision is not one.
func TestDataLayoutAllowsOneOrderAcrossDifferentKinds(t *testing.T) {
	source := mergeFS(
		entityDir("boundary", "alpha", 1),
		entityDir("guardrail", "beta", 1),
	)
	if _, _, err := dataLayout(source, "fixture"); err != nil {
		t.Fatalf("one order in two kinds is not a collision: %v", err)
	}
}

func TestDataLayoutSequencesDistinctOrdersByFilename(t *testing.T) {
	source := mergeFS(
		entityDir("personality", "zulu", 2),
		entityDir("personality", "alpha", 9),
	)
	projected, _, err := dataLayout(source, "fixture")
	if err != nil {
		t.Fatalf("distinct orders must project: %v", err)
	}
	names, err := fs.Glob(projected, "personalities/*")
	if err != nil {
		t.Fatalf("glob personalities: %v", err)
	}
	want := []string{"personalities/02-zulu.yaml", "personalities/09-alpha.yaml"}
	if len(names) != len(want) {
		t.Fatalf("projected %v, want %v", names, want)
	}
	for i, name := range names {
		if name != want[i] {
			t.Fatalf("projected %v, want %v", names, want)
		}
	}
}

// The two-digit projected prefix caps the order at 99, and `personFragmentSlug`
// used to catch order 100 by refusing the filename it produced.
func TestDataLayoutRefusesAnOrderAboveTheCapByNamingTheOrder(t *testing.T) {
	_, _, err := dataLayout(entityDir("personality", "alpha", maxEntityOrder+1), "fixture")
	if err == nil {
		t.Fatalf("order %d must be refused", maxEntityOrder+1)
	}
	for _, want := range []string{"alpha", "order 100", "maximum of 99"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
	if strings.Contains(err.Error(), ".yaml") {
		t.Fatalf("error %q names a filename the author never wrote", err)
	}
}

func TestDataLayoutAcceptsTheOrderAtTheCap(t *testing.T) {
	if _, _, err := dataLayout(entityDir("personality", "alpha", maxEntityOrder), "fixture"); err != nil {
		t.Fatalf("order %d is the cap, not past it: %v", maxEntityOrder, err)
	}
}
